/*
 * Copyright 2025 The DIAMBRA Authors
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

package agent

import (
	"os"

	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/log"
	dclient "github.com/docker/docker/client"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewBuildCmd(logger *log.Logger) *cobra.Command {
	tag := ""
	cmd := &cobra.Command{
		Use:   "build [path/to/agent]",
		Short: "Build a container image for submission",
		Long:  `This builds a container image from the given path.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				args = []string{"."}
			}
			client, err := dclient.NewClientWithOpts(dclient.FromEnv, dclient.WithAPIVersionNegotiation())
			if err != nil {
				level.Error(logger).Log("msg", "failed to create docker client", "err", err)
				os.Exit(1)
			}

			runner, err := container.NewDockerRunner(logger, client, false)
			if err != nil {
				level.Error(logger).Log("msg", "failed to create docker runner", "err", err)
				os.Exit(1)
			}

			if tag == "" {
				var err error
				tag, err = container.TagFromDir(args[0])
				if err != nil {
					level.Error(logger).Log("msg", "failed to get tag from dir", "err", err)
					os.Exit(1)
				}
			}

			if err := runner.Build(args[0], tag); err != nil {
				level.Error(logger).Log("msg", "failed to build agent", "err", err)
				os.Exit(1)
			}
		},
		Args: cobra.MaximumNArgs(1),
	}
	cmd.Flags().StringVarP(&tag, "tag", "t", tag, "Tag for the image")

	return cmd
}
