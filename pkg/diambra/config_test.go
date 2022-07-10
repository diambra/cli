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
