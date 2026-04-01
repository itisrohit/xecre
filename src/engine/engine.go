package engine

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/itisrohit/xecre/src/models"
	"github.com/itisrohit/xecre/src/runner"
)

type DockerEngine struct {
	Client *client.Client
	Pools  map[string]chan string
}

func NewDockerEngine() (*DockerEngine, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}

	e := &DockerEngine{
		Client: cli,
		Pools:  make(map[string]chan string),
	}

	for lang := range runner.SupportedLanguages {
		e.Pools[lang] = make(chan string, 10) // Optimized Pool Size
		go e.replenishPool(context.Background(), lang)
	}

	return e, nil
}

func (e *DockerEngine) replenishPool(ctx context.Context, lang string) {
	config := runner.SupportedLanguages[lang]
	for {
		resp, err := e.Client.ContainerCreate(ctx, client.ContainerCreateOptions{
			Config: &container.Config{
				Image: config.Image,
				Cmd:   []string{"cat"},
				Tty:   true,
			},
		})
		if err != nil {
			log.Printf("Create failed: %v", err)
			continue
		}

		if _, err := e.Client.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
			log.Printf("Start failed: %v", err)
			continue
		}

		e.Pools[lang] <- resp.ID
	}
}

func (e *DockerEngine) Execute(ctx context.Context, req models.ExecutionRequest) (*models.ExecutionResult, error) {
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

	defer func() {
		go e.Client.ContainerRemove(context.Background(), containerID, client.ContainerRemoveOptions{Force: true})
	}()

	config := runner.SupportedLanguages[req.Language]
	execConfig := client.ExecCreateOptions{
		Cmd:          []string{"sh", "-c", config.RunCmd, "exe", req.Code},
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

	var stdout, stderr bytes.Buffer
	stdcopy.StdCopy(&stdout, &stderr, resp.Reader)

	return &models.ExecutionResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}
