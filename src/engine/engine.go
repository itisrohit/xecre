package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/itisrohit/xecre/src/models"
	"github.com/itisrohit/xecre/src/runner"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const (
	poolSize      = 2
	maxContainers = 6
	maxRequests   = 4
	appID         = "xecre-local"
)

var ErrBusy = errors.New("execution capacity full; try again shortly")

type DockerEngine struct {
	Client     *client.Client
	Pools      map[string]chan string
	cancel     context.CancelFunc
	workers    sync.WaitGroup
	containers sync.Map
	slots      chan struct{}
	admission  chan struct{}
	session    string
}

func NewDockerEngine() (*DockerEngine, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}

	e := &DockerEngine{
		Client:    cli,
		Pools:     make(map[string]chan string),
		slots:     make(chan struct{}, maxContainers),
		admission: make(chan struct{}, maxRequests),
		session:   fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	// This local deployment has one API instance. Reconcile only its app label.
	reconcileCtx, reconcileCancel := context.WithTimeout(context.Background(), 10*time.Second)
	items, err := cli.ContainerList(reconcileCtx, client.ContainerListOptions{All: true})
	reconcileCancel()
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("reconcile pools: %w", err)
	}
	for _, item := range items.Items {
		if item.Labels["io.xecre.app"] != appID {
			continue
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := cli.ContainerRemove(cleanupCtx, item.ID, client.ContainerRemoveOptions{Force: true})
		cleanupCancel()
		if err != nil {
			cli.Close()
			return nil, fmt.Errorf("remove stale pool %s: %w", item.ID, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	for lang := range runner.SupportedLanguages {
		e.Pools[lang] = make(chan string, poolSize)
		e.workers.Add(1)
		go func(language string) { defer e.workers.Done(); e.replenishPool(ctx, language) }(lang)
	}

	return e, nil
}

func (e *DockerEngine) replenishPool(ctx context.Context, lang string) {
	config := runner.SupportedLanguages[lang]
	for {
		if ctx.Err() != nil {
			return
		}
		if len(e.Pools[lang]) == cap(e.Pools[lang]) {
			if !pause(ctx, 100*time.Millisecond) {
				return
			}
			continue
		}
		select {
		case e.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		// Finish each bounded create call before shutdown reconciliation.
		opCtx, opCancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := e.Client.ContainerCreate(opCtx, client.ContainerCreateOptions{
			Config: &container.Config{
				Image:  config.Image,
				Cmd:    []string{"cat"},
				Tty:    true,
				Labels: map[string]string{"io.xecre.managed": "true", "io.xecre.app": appID, "io.xecre.session": e.session},
			},
			HostConfig: &container.HostConfig{NetworkMode: "none", Resources: container.Resources{Memory: 256 * 1024 * 1024, NanoCPUs: 500000000, PidsLimit: func() *int64 { n := int64(64); return &n }()}},
		})
		opCancel()
		if err != nil {
			<-e.slots
			log.Printf("Create failed: %v", err)
			if !pause(ctx, time.Second) {
				return
			}
			continue
		}

		e.containers.Store(resp.ID, true)
		if _, err := e.Client.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
			log.Printf("Start failed: %v", err)
			e.remove(resp.ID)
			if !pause(ctx, time.Second) {
				return
			}
			continue
		}

		select {
		case e.Pools[lang] <- resp.ID:
		case <-ctx.Done():
			return
		}
	}
}

func (e *DockerEngine) remove(id string) {
	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := e.Client.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
		cancel()
		if err == nil || errdefs.IsNotFound(err) {
			if _, loaded := e.containers.LoadAndDelete(id); loaded {
				<-e.slots
			}
			return
		}
	}
	log.Printf("Cleanup pending for owned container %s; capacity retained", id)
}

// Close stops pool builders before removing this engine's disposable containers.
func (e *DockerEngine) Close() {
	e.cancel()
	e.workers.Wait()
	var removals sync.WaitGroup
	e.containers.Range(func(key, _ any) bool {
		removals.Add(1)
		go func(id string) {
			defer removals.Done()
			e.remove(id)
		}(key.(string))
		return true
	})
	removals.Wait()
	// A create request may succeed in Docker after its client deadline expired.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	items, err := e.Client.ContainerList(ctx, client.ContainerListOptions{All: true})
	cancel()
	if err != nil {
		log.Printf("Shutdown reconciliation failed: %v", err)
	} else {
		for _, item := range items.Items {
			if item.Labels["io.xecre.app"] == appID && item.Labels["io.xecre.session"] == e.session {
				e.remove(item.ID)
			}
		}
	}
	e.Client.Close()
}

func (e *DockerEngine) Execute(ctx context.Context, req models.ExecutionRequest) (*models.ExecutionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	select {
	case e.admission <- struct{}{}:
		defer func() { <-e.admission }()
	default:
		return nil, ErrBusy
	}
	pool, ok := e.Pools[req.Language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	var containerID string
	select {
	case containerID = <-pool:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	poolWait := time.Since(started)

	defer func() {
		go e.remove(containerID)
	}()

	config := runner.SupportedLanguages[req.Language]
	execConfig := client.ExecCreateOptions{
		Cmd:          append(append([]string{}, config.Command...), req.Code),
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := e.Client.ExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, err
	}

	resp, err := e.Client.ExecAttach(ctx, execID.ID, client.ExecAttachOptions{})
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			resp.Close()
		case <-finished:
		}
	}()

	var stdout, stderr bytes.Buffer
	remaining := 1024 * 1024
	if _, err := stdcopy.StdCopy(&boundedOutput{&stdout, &remaining}, &boundedOutput{&stderr, &remaining}, resp.Reader); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("collect output: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	inspection, err := e.Client.ExecInspect(ctx, execID.ID, client.ExecInspectOptions{})
	if err != nil {
		return nil, err
	}

	total := time.Since(started)
	return &models.ExecutionResult{
		Stdout:        stdout.String(),
		Stderr:        stderr.String(),
		TimeTaken:     total,
		PoolWait:      poolWait,
		ExecutionTime: total - poolWait,
		ExitCode:      inspection.ExitCode,
	}, nil
}

func pause(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type boundedOutput struct {
	buffer    *bytes.Buffer
	remaining *int
}

func (w *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > *w.remaining {
		return 0, fmt.Errorf("output exceeds 1 MiB: %w", io.ErrShortWrite)
	}
	n, err := w.buffer.Write(p)
	*w.remaining -= n
	return n, err
}
