package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// DockerRunner implements Runner
type DockerRunner struct {
	log.Logger
	*client.Client
	TimeoutStop time.Duration
}

func NewDockerRunner(logger log.Logger) (*DockerRunner, error) {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRunner{
		Logger:      logger,
		Client:      client,
		TimeoutStop: 10 * time.Second,
	}, nil
}

func (r *DockerRunner) Start(c *Container) (*ContainerStatus, error) {
	var (
		ctx    = context.Background()
		config = &container.Config{
			Image:      c.Image,
			Cmd:        c.Command,
			Env:        c.Env,
			User:       c.User,
			Tty:        false,
			StopSignal: "SIGKILL", // FIXME: Make diambraApp handle SIGTERM instead
		}
		hostConfig = &container.HostConfig{
			AutoRemove: true,
		}
	)
	hostConfig.Mounts = make([]mount.Mount, len(c.BindMounts))

	for i, m := range c.BindMounts {
		level.Debug(r.Logger).Log("msg", "adding bind mount", "source", m.HostPath, "target", m.ContainerPath)
		hostConfig.Mounts[i] = mount.Mount{
			Type:   mount.TypeBind,
			Source: m.HostPath,
			Target: m.ContainerPath,
		}
	}
	level.Debug(r.Logger).Log("msg", "creating container", "config", fmt.Sprintf("%#v", config), "hostConfig", fmt.Sprintf("%#v", hostConfig))
	dc, err := r.Client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}

	if err := r.Client.ContainerStart(ctx, dc.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	level.Debug(r.Logger).Log("msg", "container running")
	return &ContainerStatus{ID: dc.ID}, nil
}

func (r *DockerRunner) CopyLogs(id string, stdout, stderr io.Writer) error {
	ctx := context.TODO()
	out, err := r.Client.ContainerLogs(ctx, id, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return err
}

func (r *DockerRunner) Stop(id string) error {
	ctx := context.TODO()
	return r.Client.ContainerStop(ctx, id, &r.TimeoutStop)
}
