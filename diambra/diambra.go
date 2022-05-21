package diambra

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/diambra/cli/container"
)

const (
	ContainerPort = "50051/tcp"
)

type EnvConfig struct {
	LockFPS    bool
	GUI        bool
	Audio      bool
	Scale      int
	AutoRemove bool
	PullImage  bool

	RomsPath string
	CredPath string
	Image    string

	User string

	Stdout io.Writer
	Stderr io.Writer
}

type Env struct {
	*container.ContainerStatus
	container.Address
}

type Diambra struct {
	log.Logger
	container.Runner
	Envs   []*Env
	config *EnvConfig
}

func NewDiambra(logger log.Logger, config *EnvConfig) (*Diambra, error) {
	runner, err := container.NewDockerRunner(logger, config.AutoRemove)
	if err != nil {
		return nil, err
	}
	if config.PullImage {
		reader, err := runner.PullImage(config.Image)
		if err != nil {
			return nil, fmt.Errorf("couldn't pull image %s: %w", config.Image, err)
		}
		defer reader.Close()
		io.Copy(os.Stderr, reader)
	}
	return &Diambra{
		Logger: logger,
		Runner: runner,
		Envs:   []*Env{},
		config: config,
	}, nil
}

func (e *Diambra) EnvsString() (string, error) {
	envs := make([]string, len(e.Envs))
	for i, env := range e.Envs {
		portn, err := env.Port.Number()
		if err != nil {
			return "", fmt.Errorf("invalid port %s: %w", env.Port, err)
		}
		envs[i] = fmt.Sprintf("127.0.0.1:%d", portn)
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

func (e *Diambra) Start() error {
	first := true
	for i := 0; i < e.config.Scale; i++ {
		level.Debug(e.Logger).Log("msg", "creating env container", "envID", i)
		cs, err := e.Runner.Start(newEnvContainer(e.config, i))
		if err != nil {
			return err
		}
		level.Debug(e.Logger).Log("msg", "started env container", "id", cs.ID)
		env := &Env{
			ContainerStatus: cs,
			Address:         (*cs.PortMapping)[ContainerPort],
		}
		e.Envs = append(e.Envs, env)

		// On first env we wait for the container to start, attach to it until the grpc port is open.
		// This allows diambraEngine to ask for credentials if they don't exist/are expired.
		if first {
			wc, rc, err := e.Runner.Attach(cs.ID)
			if err != nil {
				return err
			}
			streamer := container.NewStreamer(e.Logger, wc, rc)
			if err := streamer.Stream(); err != nil {
				return err
			}

			e.waitForGRPC(env.Address)
			streamer.Close()
			first = false
		}

		go func(id string) {
			level.Debug(e.Logger).Log("msg", "in go func")
			if err := e.Runner.LogLogs(id, log.With(e.Logger, "id", id)); err != nil {
				level.Warn(e.Logger).Log("msg", "LogLogs failed", "err", err.Error())
			}
			level.Debug(e.Logger).Log("msg", "end of go func")
		}(cs.ID)
		level.Debug(e.Logger).Log("msg", "logs copying..")
	}
	return nil
}

func newEnvContainer(config *EnvConfig, envID int) *container.Container {
	pm := &container.PortMapping{}
	pm.AddPortMapping(ContainerPort, "0/tcp", "127.0.0.1")

	return &container.Container{
		Image:       config.Image,
		User:        config.User,
		PortMapping: pm,
		BindMounts: []*container.BindMount{
			container.NewBindMount(config.CredPath, "/tmp/.diambraCred"),
			container.NewBindMount(config.RomsPath, "/opt/diambraArena/roms"),
		},
	}
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
