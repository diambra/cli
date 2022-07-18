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

func (r *mockRunner) StopAll() error {
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
