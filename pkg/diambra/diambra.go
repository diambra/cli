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

package diambra

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/containerd/console"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/diambra/cli/pkg/container"
)

const (
	ContainerPort = "50051/tcp"
)

type Env struct {
	*container.ContainerStatus
	container.Address
}

type Diambra struct {
	log.Logger
	console console.Console
	container.Runner
	Envs   []*Env
	config *EnvConfig
	// streamer *ui.Streamer
}

// func NewDiambra(logger log.Logger, config *EnvConfig, streamer *ui.Streamer) (*Diambra, error) {

func NewDiambra(logger log.Logger, console console.Console, runner container.Runner, config *EnvConfig) (*Diambra, error) {
	return &Diambra{
		Logger:  logger,
		console: console,
		Runner:  runner,
		Envs:    []*Env{},
		config:  config,
	}, nil
}

// FIXME: check errors earlier so we don't have to here
func (e *Diambra) EnvsString() (string, error) {
	envs := make([]string, len(e.Envs))
	for i, env := range e.Envs {
		portn, err := env.Port.Number()
		if err != nil {
			return "", fmt.Errorf("invalid port %s: %w", env.Port, err)
		}
		host := env.Address.Host
		if e.config.UseContainerIP {
			host = env.ContainerStatus.Address
			portn = 50051
		} else {
			if !net.ParseIP(host).IsLoopback() {

				host = "127.0.0.1"
			}
		}
		envs[i] = fmt.Sprintf("%s:%d", host, portn)
	}
	return strings.Join(envs, " "), nil
}

// FIXME: Merge with above
func (e *Diambra) EnvsStringContainer() (string, error) {
	portn, err := container.Port(ContainerPort).Number()
	if err != nil {
		return "", err
	}
	envs := make([]string, len(e.Envs))
	for i, env := range e.Envs {
		envs[i] = fmt.Sprintf("%s:%d", env.ContainerStatus.Address, portn)
	}
	return strings.Join(envs, " "), nil
}

func (e *Diambra) waitForGRPC(addr container.Address) error {
	_, hp, err := addr.ProtoAddress()
	if err != nil {
		return err
	}
	for {
		_, err := grpc.Dial(hp, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			level.Debug(e.Logger).Log("msg", "couldn't connect to endpoint", "endpoint", hp, "err", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		return nil
	}
}
func (d *Diambra) RandInt() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(0xFFFF))
	if err != nil {
		return 0, err
	}
	return int(n.Uint64()), nil
}

func (d *Diambra) start(envId int, first bool) error {
	envLogger := log.With(d.Logger, "source", "env")

	level.Debug(d.Logger).Log("msg", "creating env container", "envID", envId)
	randomSeed, err := d.RandInt()
	if err != nil {
		return fmt.Errorf("couldn't generate random seed: %w", err)
	}

	ec, err := newEnvContainer(d.config, envId, randomSeed)
	if err != nil {
		return fmt.Errorf("couldn't create env container: %w", err)
	}

	if first && !d.config.NoPullImage {
		if err := d.Runner.Pull(ec, d.config.Output); err != nil {
			return err
		}
	}

	cs, err := d.Runner.Start(ec)
	if err != nil {
		return err
	}
	level.Debug(d.Logger).Log("msg", "started env container", "id", cs.ID)
	env := &Env{
		ContainerStatus: cs,
		Address:         (*cs.PortMapping)[ContainerPort],
	}
	d.Envs = append(d.Envs, env)

	// On first env we wait for the container to start, attach to it until the grpc port is open.
	// This allows diambraEngine to ask for credentials if they don't exist/are expired.
	if first && d.config.Tty && d.config.Interactive {
		wc, rc, err := d.Runner.Attach(cs.ID)
		if err != nil {
			return err
		}

		if err := d.console.SetRaw(); err != nil {
			return err
		}
		ws, err := d.console.Size()
		if err != nil {
			return err
		}
		if runtime.GOOS != "windows" {
			if err := d.console.Resize(ws); err != nil {
				return fmt.Errorf("couldn't resize console: %w", err)
			}
		}

		done := false
		d.copyLogs(&done, wc, os.Stdin, os.Stdout, rc)

		level.Debug(d.Logger).Log("msg", "waiting for grpc")
		if err := d.waitForGRPC(env.Address); err != nil {
			return fmt.Errorf("error waiting for grpc: %w", err)
		}
		level.Debug(d.Logger).Log("msg", "closing streamer")
		done = true
		wc.Close()
		rc.Close()
		if err := d.console.Reset(); err != nil {
			level.Error(d.Logger).Log("msg", "error resetting console", "err", err.Error())
		}

	}
	go func(id string) {
		level.Debug(d.Logger).Log("msg", "in go func")
		if err := d.Runner.LogLogs(id, log.With(envLogger, "id", id)); err != nil {
			level.Warn(d.Logger).Log("msg", "LogLogs failed", "err", err.Error())
		}
		level.Debug(d.Logger).Log("msg", "end of go func")
	}(cs.ID)

	level.Debug(d.Logger).Log("msg", "logs copying..")
	return nil
}

func (d *Diambra) copyLogs(done *bool, wc io.WriteCloser, in io.Reader, out io.Writer, rc io.ReadCloser) {
	go func() {
		if _, err := io.Copy(wc, os.Stdin); err != nil {
			if *done {
				return
			}
			level.Error(d.Logger).Log("msg", "error copying stdin to container stdin", "err", err.Error())
		}
	}()
	go func() {
		if _, err := io.Copy(os.Stdout, rc); err != nil {
			if *done {
				return
			}
			level.Error(d.Logger).Log("msg", "error copying container stdout to stdout", "err", err.Error())
		}
	}()
}

func (d *Diambra) Start() error {
	level.Debug(d.Logger).Log("msg", "starting diambra", "config", fmt.Sprintf("%+v", d.config))
	if err := d.config.Validate(); err != nil {
		return err
	}
	first := true
	for i := 0; i < d.config.Scale; i++ {
		if err := d.start(i, first); err != nil {
			return err
		}
		first = false

	}
	return nil
}

func newEnvContainer(config *EnvConfig, envID, randomSeed int) (*container.Container, error) {
	pm := &container.PortMapping{}
	hostPort := "0/tcp"
	if config.PreallocatePort {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, err
		}
		hostPort = fmt.Sprintf("%d/tcp", listener.Addr().(*net.TCPAddr).Port)
	}

	pm.AddPortMapping(ContainerPort, hostPort, config.Host)

	args := config.AppArgs
	args.RandomSeed = randomSeed
	c := &container.Container{
		Name:        fmt.Sprintf("arena-%3d", envID),
		Image:       config.Image,
		User:        config.User,
		Args:        args.Args(),
		PortMapping: pm,
		BindMounts: []*container.BindMount{
			container.NewBindMount(config.CredPath, "/tmp/.diambra/credentials"),
			container.NewBindMount(config.RomsPath, "/opt/diambraArena/roms"),
		},
	}
	c.BindMounts = append(c.BindMounts, config.Mounts...)

	if config.AppArgs.Render {
		if err := configureRender(config, c); err != nil {
			return nil, fmt.Errorf("error configuring render: %w", err)
		}
	}
	if config.SeccompProfile != "" {
		c.SecurityOpt = []string{"seccomp=" + config.SeccompProfile}
	}
	return c, nil
}

func (e *Diambra) Cleanup() error {
	var rerr error
	for _, env := range e.Envs {
		level.Debug(e.Logger).Log("msg", "stopping container", "id", env.ContainerStatus.ID)
		if err := e.Runner.Stop(env.ContainerStatus.ID); err != nil {
			rerr = err
			level.Warn(e.Logger).Log("msg", "couldn't stop container", "err", err.Error())
		}
	}
	return rerr
}

func (e *Diambra) RunAgentImage(image string, args []string) error {
	level.Debug(e.Logger).Log("msg", "running in container", "image", image, "args", fmt.Sprintf("%v", args))
	statusCode, err := e.RunAgentContainer(&container.Container{
		Name:  "agent",
		Image: image,
		Args:  args,
	})
	if err != nil {
		return err
	}
	if statusCode != 0 {
		return fmt.Errorf("agent exited with status code %d", statusCode)
	}
	return nil
}

func (e *Diambra) RunAgentContainer(c *container.Container) (int, error) {
	if !e.config.NoPullImage {
		if err := e.Runner.Pull(c, e.config.Output); err != nil {
			return 1, err
		}
	}
	envs, err := e.EnvsStringContainer()
	if err != nil {
		return 1, err
	}
	c.Env = append(c.Env, "DIAMBRA_ENVS="+envs)

	cs, err := e.Runner.Start(c)
	if err != nil {
		return 1, err
	}
	wc, rc, err := e.Runner.Attach(cs.ID)
	if err != nil {
		return 1, err
	}

	done := false
	e.copyLogs(&done, wc, os.Stdin, os.Stdout, rc)

	level.Debug(e.Logger).Log("msg", "waiting for container to exit")
	statusCode, err := e.Runner.Wait(cs.ID)
	if err != nil {
		return 1, fmt.Errorf("couldn't wait for container to finish: %w", err)
	}
	wc.Close()
	level.Debug(e.Logger).Log("msg", "waiting for stdout to close")
	done = true

	return statusCode, nil
}
