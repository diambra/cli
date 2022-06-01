package container

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// DockerRunner implements Runner
type DockerRunner struct {
	log.Logger
	*client.Client
	TimeoutStop time.Duration
	AutoRemove  bool
}

func NewDockerRunner(logger log.Logger, client *client.Client, autoRemove bool) *DockerRunner {
	return &DockerRunner{
		Logger:      logger,
		Client:      client,
		TimeoutStop: 10 * time.Second,
		AutoRemove:  autoRemove,
	}
}

func (r *DockerRunner) PullImage(name string) (io.ReadCloser, error) {
	return r.Client.ImagePull(context.TODO(), name, types.ImagePullOptions{})
}

func (r *DockerRunner) Start(c *Container) (*ContainerStatus, error) {
	var (
		ctx    = context.Background()
		config = &container.Config{
			Image:      c.Image,
			Cmd:        c.Args,
			Env:        c.Env,
			User:       c.User,
			Tty:        true,
			OpenStdin:  true,
			StopSignal: "SIGKILL", // FIXME: Make diambraApp handle SIGTERM insteads
		}
		hostConfig = &container.HostConfig{
			AutoRemove:  r.AutoRemove,
			SecurityOpt: c.SecurityOpt,
		}
	)
	hostConfig.Mounts = make([]mount.Mount, len(c.BindMounts))

	if c.PortMapping != nil {
		hostConfig.PortBindings = make(nat.PortMap, len(*c.PortMapping))
		config.ExposedPorts = make(nat.PortSet, len(*c.PortMapping))
		for cp, ha := range *c.PortMapping {
			level.Debug(r.Logger).Log("msg", "mapping port", "containerPort", cp, "hostPort", ha.Port)
			hostConfig.PortBindings[nat.Port(cp)] = []nat.PortBinding{{HostIP: ha.Host, HostPort: string(ha.Port)}}
			config.ExposedPorts[nat.Port(cp)] = struct{}{}
		}
	}

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
	cj, err := r.Client.ContainerInspect(ctx, dc.ID)
	if err != nil {
		return nil, err
	}
	level.Debug(r.Logger).Log("msg", "container running")
	portMapping := make(PortMapping, len(cj.NetworkSettings.Ports))
	for p, pbs := range cj.NetworkSettings.Ports {
		portMapping.AddPortMapping(string(p), string(pbs[0].HostPort), pbs[0].HostIP)
	}
	return &ContainerStatus{ID: dc.ID, PortMapping: &portMapping, Address: cj.NetworkSettings.IPAddress}, nil
}

type logWriter struct {
	log.Logger
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	level.Info(l).Log("msg", string(p))
	return len(p), nil
}

func (r *DockerRunner) LogLogs(id string, logger log.Logger) error {
	ctx := context.TODO()
	out, err := r.Client.ContainerLogs(ctx, id, types.ContainerLogsOptions{ShowStdout: true, Follow: true})
	if err != nil {
		return err
	}
	lw := &logWriter{logger}
	level.Debug(r.Logger).Log("msg", "copying logs in LogLogs")
	_, err = io.Copy(lw, out)
	level.Debug(r.Logger).Log("msg", "done copying logs in LogLogs", "err", err)
	return err
	//_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

func (r *DockerRunner) Stop(id string) error {
	ctx := context.TODO()
	return r.Client.ContainerStop(ctx, id, &r.TimeoutStop)
}

type HijackedResponseReader struct {
	log.Logger
	types.HijackedResponse
}

func (r *HijackedResponseReader) Read(p []byte) (n int, err error) {
	n, err = r.HijackedResponse.Reader.Read(p)
	// level.Debug(r.Logger).Log("msg", "read", "p", string(p), "n", n, "err", err)
	return n, err
}
func (r *HijackedResponseReader) Close() error {
	level.Debug(r.Logger).Log("msg", "closing")
	r.HijackedResponse.Close()
	return nil
}

func (r *DockerRunner) Attach(id string) (io.WriteCloser, io.ReadCloser, error) {
	ctx := context.TODO()
	resp, err := r.Client.ContainerAttach(ctx, id, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   false, // FIXME?
	})
	if err != nil {
		return nil, nil, err
	}

	return resp.Conn, &HijackedResponseReader{log.With(r.Logger, "in", "HijackedResponseReader"), resp}, nil
}

func (r *DockerRunner) Wait(id string) error {
	ctx := context.TODO()
	statusCh, errCh := r.Client.ContainerWait(ctx, id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}
	return nil
}
