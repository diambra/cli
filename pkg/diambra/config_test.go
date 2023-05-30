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
	"testing"

	"github.com/diambra/cli/pkg/diambra/client"
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
			"from args, with secrets",
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			submission, err := tc.config.Submission("", tc.args)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expected, submission)
		})
	}
}
