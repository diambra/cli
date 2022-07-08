package diambra

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/diambra/cli/pkg/container"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
)

// Implements Runner
type mockRunner struct {
	envs map[string]*container.Container
}

func (r *mockRunner) Start(c *container.Container) (*container.ContainerStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (r *mockRunner) LogLogs(id string, logger log.Logger) error {
	panic("not implemented") // TODO: Implement
}

func (r *mockRunner) Stop(id string) error {
	panic("not implemented") // TODO: Implement
}

func (r *mockRunner) Attach(id string) (io.WriteCloser, io.ReadCloser, error) {
	panic("not implemented") // TODO: Implement
}

func (r *mockRunner) Wait(id string) error {
	panic("not implemented") // TODO: Implement
}

func (r *mockRunner) Pull(c *container.Container, output *os.File) error {
	panic("not implemented") // TODO: Implement
}

func TestDiambra(t *testing.T) {
	var (
		logger = log.NewLogfmtLogger(os.Stderr)
		assert = assert.New(t)

		runner = &mockRunner{}
		config = &EnvConfig{
			AppArgs:        AppArgs{},
			Scale:          1,
			AutoRemove:     false,
			AgentImage:     "",
			PullImage:      false,
			RomsPath:       os.TempDir(),
			CredPath:       filepath.Join(os.TempDir(), "credfile"),
			Image:          "",
			User:           "",
			SeccompProfile: "",
			Tty:            false,
			Interactive:    false,
		}
	)
	_, err := NewDiambra(logger, nil, runner, config)
	assert.NoError(err)
}
