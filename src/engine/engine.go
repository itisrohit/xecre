package engine

import (
	"context"
	"github.com/docker/docker/client"
)

type DockerEngine struct {
	Client *client.Client
}

func NewDockerEngine() (*DockerEngine, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		Client: cli,
	}, nil

}
