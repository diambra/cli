/*
 * Copyright 2022 The DIAMBRA Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/moby/term"
)

// DockerRunner implements Runner
type DockerRunner struct {
	log.Logger
	*client.Client
	TimeoutStop time.Duration
	AutoRemove  bool
}

func NewDockerRunner(logger log.Logger, client *client.Client, autoRemove bool) (*DockerRunner, error) {
	_, err := client.Ping(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to docker. Make sure your user has docker access: %w", err)
	}
	return &DockerRunner{
		Logger:      logger,
		Client:      client,
		TimeoutStop: 10 * time.Second,
		AutoRemove:  autoRemove,
	}, nil
}

func (r *DockerRunner) Pull(c *Container, output *os.File) error {
	reader, err := r.Client.ImagePull(context.TODO(), c.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("couldn't pull image %s: %w:\nTo disable pulling the image on start, retry with --images.pull=false", c.Image, err)
	}
	defer reader.Close()

	termFd, isTerm := term.GetFdInfo(output)
	return jsonmessage.DisplayJSONMessagesStream(reader, io.Writer(output), termFd, isTerm, nil)
}

func (r *DockerRunner) Start(c *Container) (*ContainerStatus, error) {
	var (
		ctx    = context.Background()
		config = &container.Config{
			Image:     c.Image,
			Hostname:  c.Hostname,
			Env:       c.Env,
			User:      c.User,
			Tty:       true,
			OpenStdin: true,
			Labels: map[string]string{
				"diambra": "env",
			},
			StopSignal: "SIGKILL", // FIXME: Make diambraApp handle SIGTERM insteads
			WorkingDir: c.WorkingDir,
			Cmd:        c.Args,
			Entrypoint: c.Command,
		}
		hostConfig = &container.HostConfig{
			AutoRemove:  r.AutoRemove,
			SecurityOpt: c.SecurityOpt,
			IpcMode:     container.IpcMode(c.IPCMode),
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
	level.Info(l).Log("msg", strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}

func (r *DockerRunner) LogLogs(id string, logger log.Logger) error {
	ctx := context.TODO()
	out, err := r.Client.ContainerLogs(ctx, id, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Since:      strconv.Itoa(int(time.Now().Unix())),
	})
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

func ptr[T any](t T) *T {
	return &t
}

func (r *DockerRunner) Stop(id string) error {
	ctx := context.TODO()
	return r.Client.ContainerStop(ctx, id, container.StopOptions{
		Timeout: ptr(int(r.TimeoutStop.Seconds())),
	})
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
		Logs:   true,
	})
	if err != nil {
		return nil, nil, err
	}

	return resp.Conn, &HijackedResponseReader{log.With(r.Logger, "in", "HijackedResponseReader"), resp}, nil
}

func (r *DockerRunner) Wait(id string) (int, error) {
	ctx := context.TODO()
	statusCh, errCh := r.Client.ContainerWait(ctx, id, container.WaitConditionNotRunning)

	var (
		err        error
		statusCode int
	)
	select {
	case e := <-errCh:
		level.Debug(r.Logger).Log("msg", "got error from errCh", "err", err)
		if e != nil {
			err = e
		}
	case status := <-statusCh:
		level.Debug(r.Logger).Log("msg", "got status from statusCh", "status", status)
		statusCode = int(status.StatusCode)
	}
	level.Debug(r.Logger).Log("msg", "done waiting", "err", err, "statusCode", statusCode)
	return statusCode, err
}

func (r *DockerRunner) StopAll() error {
	filters := filters.NewArgs()
	filters.Add("label", "diambra=env")
	ctx := context.TODO()
	containers, err := r.Client.ContainerList(ctx, types.ContainerListOptions{Filters: filters})
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		level.Info(r.Logger).Log("msg", "no containers to stop")
		os.Exit(0)
	}
	for _, c := range containers {
		level.Info(r.Logger).Log("msg", "stopping container", "id", c.ID)
		if err := r.Stop(c.ID); err != nil {
			return err
		}

		ci, err := r.Client.ContainerInspect(ctx, c.ID)
		if err != nil {
			return err
		}
		if ci.HostConfig.AutoRemove {
			continue
		}
		if err := r.Client.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true}); err != nil {
			return err
		}
	}
	return nil
}
