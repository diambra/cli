package diambra

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
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
	container.Runner
	Envs   []*Env
	config *EnvConfig
	// streamer *ui.Streamer
}

// func NewDiambra(logger log.Logger, config *EnvConfig, streamer *ui.Streamer) (*Diambra, error) {

func NewDiambra(logger log.Logger, runner container.Runner, config *EnvConfig) (*Diambra, error) {
	return &Diambra{
		Logger: logger,
		Runner: runner,
		Envs:   []*Env{},
		config: config,
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
		envs[i] = fmt.Sprintf("%s:%d", env.Address.Host, portn)
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
	agentLogger := d.Logger //e.Screen.NewTab())
	level.Debug(d.Logger).Log("msg", "creating env container", "envID", envId)
	randomSeed, err := d.RandInt()
	if err != nil {
		return fmt.Errorf("couldn't generate random seed: %w", err)
	}
	cs, err := d.Runner.Start(newEnvContainer(d.config, envId, randomSeed))
	if err != nil {
		return fmt.Errorf("couldn't start env container: %w", err)
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

		term := console.Current()
		if err := term.SetRaw(); err != nil {
			return err
		}
		ws, err := term.Size()
		if err != nil {
			return err
		}
		term.Resize(ws)

		go func() {
			io.Copy(wc, os.Stdin)
		}()
		go func() {
			io.Copy(os.Stdout, rc)
		}()

		level.Debug(d.Logger).Log("msg", "waiting for grpc")
		d.waitForGRPC(env.Address)
		level.Debug(d.Logger).Log("msg", "closing streamer")
		wc.Close()
		rc.Close()
		term.Reset()

		// FIXME: We should just call Render() automatically from the Writer
		/*
			go func() {
				ticker := time.NewTicker(500 * time.Millisecond) // ~30 fps
				for range ticker.C {
					e.Screen.Render()
				}
			}()*/
	}
	go func(id string) {
		level.Debug(d.Logger).Log("msg", "in go func")
		if err := d.Runner.LogLogs(id, log.With(agentLogger, "id", id)); err != nil {
			level.Warn(d.Logger).Log("msg", "LogLogs failed", "err", err.Error())
		}
		level.Debug(d.Logger).Log("msg", "end of go func")
	}(cs.ID)

	level.Debug(d.Logger).Log("msg", "logs copying..")
	return nil
}

func (d *Diambra) Start() error {
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

func newEnvContainer(config *EnvConfig, envID, randomSeed int) *container.Container {
	pm := &container.PortMapping{}
	pm.AddPortMapping(ContainerPort, "0/tcp", "127.0.0.1")

	args := config.AppArgs
	args.RandomSeed = randomSeed
	c := &container.Container{
		Name:        fmt.Sprintf("arena-%3d", envID),
		Image:       config.Image,
		User:        config.User,
		Args:        args.Args(),
		PortMapping: pm,
		BindMounts: []*container.BindMount{
			container.NewBindMount(config.CredPath, "/tmp/.diambraCred"),
			container.NewBindMount(config.RomsPath, "/opt/diambraArena/roms"),
		},
	}
	c.BindMounts = append(c.BindMounts, config.Mounts...)

	if config.AppArgs.Render {
		xauthority := filepath.Join(config.Home, ".Xauthority")
		if xap := os.Getenv("XAUTHORITY"); xap != "" {
			xauthority = xap
		}
		c.BindMounts = append(c.BindMounts,
			container.NewBindMount("/tmp/.X11-unix", "/tmp/.X11-unix"),
			container.NewBindMount(xauthority, "/tmp/.Xauthority"),
		)
		c.Hostname = config.Hostname
		c.Env = append(c.Env, "DISPLAY="+os.Getenv("DISPLAY"))
	}
	if config.SeccompProfile != "" {
		c.SecurityOpt = []string{"seccomp=" + config.SeccompProfile}
	}
	return c
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
	envs, err := e.EnvsStringContainer()
	if err != nil {
		return err
	}
	c := &container.Container{
		Name:  "agent",
		Image: image,
		Args:  args,
		Env:   []string{"DIAMBRA_ENVS=" + envs},
	}
	cs, err := e.Runner.Start(c)
	if err != nil {
		return err
	}
	wc, rc, err := e.Runner.Attach(cs.ID)
	if err != nil {
		return err
	}

	go func() {
		io.Copy(wc, os.Stdin)
	}()
	doneCh := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, rc)
		doneCh <- struct{}{}
	}()

	level.Debug(e.Logger).Log("msg", "waiting for container to exit")
	err = e.Runner.Wait(cs.ID)
	if err != nil {
		return fmt.Errorf("couldn't wait for container to finish: %w", err)
	}
	wc.Close()
	//rc.Close()
	level.Debug(e.Logger).Log("msg", "waiting for stdout to close")
	<-doneCh

	return nil
}
