package engine

import (
	"bytes"
	"context"
	"fmt"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/itisrohit/xecre/src/models"
	"github.com/itisrohit/xecre/src/runner"
)

type DockerEngine struct {
	Client *client.Client
}

func NewDockerEngine() (*DockerEngine, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		Client: cli,
	}, nil
}

func (e *DockerEngine) Execute(ctx context.Context, req models.ExecutionRequest) (*models.ExecutionResult, error) {
	config, ok := runner.SupportedLanguages[req.Language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	// Pull image if not present (Wait for it to finish)
	res, err := e.Client.ImagePull(ctx, config.Image, client.ImagePullOptions{})
	if err == nil {
		for msg := range res.JSONMessages(ctx) {
			if msg.Error != nil {
				return nil, msg.Error
			}
		}
	}

	resp, err := e.Client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image: config.Image,
			Cmd:   []string{config.RunCmd, "-c", req.Code},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	defer e.Client.ContainerRemove(ctx, resp.ID, client.ContainerRemoveOptions{Force: true})

	_, err = e.Client.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	waitRes := e.Client.ContainerWait(ctx, resp.ID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})
	select {
	case err := <-waitRes.Error:
		if err != nil {
			return nil, err
		}
	case <-waitRes.Result:
	}

	out, err := e.Client.ContainerLogs(ctx, resp.ID, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, err
	}
	defer out.Close()

	var stdout, stderr bytes.Buffer
	stdcopy.StdCopy(&stdout, &stderr, out)

	return &models.ExecutionResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}
