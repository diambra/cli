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
	"os"
	"path/filepath"
	"testing"

	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/diambra/cli/pkg/secretsources"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
)

func TestAppArgs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		appArgs  AppArgs
		expected []string
	}{
		{
			"empty",
			AppArgs{
				RandomSeed: 0,
				Render:     false,
				LockFPS:    false,
				Sound:      false,
			},
			[]string{},
		},
		{
			"full",
			AppArgs{
				RandomSeed: 23,
				Render:     true,
				LockFPS:    true,
				Sound:      true,
			},
			[]string{"--render", "--lockFps", "--sound", "--randomSeed", "23"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.appArgs.Args(), tc.expected)
		})
	}
}

func TestSubmissionConfig(t *testing.T) {
	envConfig := &EnvConfig{
		logger:   log.NewNopLogger(),
		CredPath: "",
	}
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	for _, tc := range []struct {
		name        string
		config      SubmissionConfig
		args        []string
		expected    *client.Submission
		expectedErr error
	}{
		{
			"empty",
			SubmissionConfig{},
			[]string{},
			nil,
			ErrInvalidArgs,
		},
		{
			"from args",
			SubmissionConfig{},
			[]string{"diambra/agent-random-1:main", "--gameId", "doapp"},
			&client.Submission{
				Manifest: client.Manifest{
					Image: "diambra/agent-random-1:main",
					Args:  []string{"--gameId", "doapp"},
				},
			},
			nil,
		},
		{
			"from file, overwrite args",
			SubmissionConfig{
				ManifestPath: "testdata/manifest.yaml",
			},
			[]string{"--gameId", "kof98umh"},
			&client.Submission{
				Manifest: client.Manifest{
					Image: "diambra/agent-random-1:main",
					Args:  []string{"--gameId", "kof98umh"},
				},
			},
			nil,
		},
		{
			"from file, setting command",
			SubmissionConfig{
				ManifestPath:  "testdata/manifest.yaml",
				ArgsIsCommand: true,
			},
			[]string{"python", "agent.py"},
			&client.Submission{
				Manifest: client.Manifest{
					Image:   "diambra/agent-random-1:main",
					Command: []string{"python", "agent.py"},
					Args:    []string{"--gameId", "doapp"},
				},
			},
			nil,
		},
		{
			"from args with sources and secrets",
			SubmissionConfig{
				ManifestPath:  "testdata/manifest.yaml",
				ArgsIsCommand: true,
				Sources:       map[string]string{"model.zip": "https://user:{{ .Secrets.foo }}@example.com/model.zip"},
				Secrets: map[string]string{
					"foo": "bar",
				},
			},
			[]string{"python", "agent.py"},
			&client.Submission{
				Manifest: client.Manifest{
					Image:   "diambra/agent-random-1:main",
					Command: []string{"python", "agent.py"},
					Args:    []string{"--gameId", "doapp"},
					Sources: map[string]string{
						"model.zip": "https://user:{{ .Secrets.foo }}@example.com/model.zip",
					},
				},
				Secrets: map[string]string{
					"foo": "bar",
				},
			},
			nil,
		},
		{
			"from args with sources and secrets from git",
			SubmissionConfig{
				ManifestPath:  "testdata/manifest.yaml",
				ArgsIsCommand: true,
				Sources:       map[string]string{"model.zip": "https://example.com/mode.zip"},
				SecretsFrom:   "git",
			},
			[]string{"python", "agent.py"},
			&client.Submission{
				Manifest: client.Manifest{
					Image:   "diambra/agent-random-1:main",
					Command: []string{"python", "agent.py"},
					Args:    []string{"--gameId", "doapp"},
					Sources: map[string]string{
						"model.zip": "https://{{ .Secrets.git_username_1 }}:{{ .Secrets.git_password_1 }}@example.com/mode.zip",
					},
				},
				Secrets: map[string]string{
					"git_username_1": "user1",
					"git_password_1": "pass1",
				},
			},
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.RegisterCredentialsProvider("git", &secretsources.GitCredentials{Helper: filepath.Join(cwd, "../../test/mock-credential-helper.sh")})
			submission, err := tc.config.Submission(envConfig, tc.args)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expected, submission)
		})
	}
}
