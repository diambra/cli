/*
 * Copyright 2024 The DIAMBRA Authors
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
	"fmt"
	"os"
	"path/filepath"

	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/diambra/cli/pkg/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

const defaultCredPath = "~/.diambra/credentials"

func NewBuildAndPushCmd(logger *log.Logger) *cobra.Command {
	var (
		name     = ""
		version  = ""
		credPath = defaultCredPath
	)

	cmd := &cobra.Command{
		Use:   "build-and-push [path/to/agent]",
		Short: "Build a container image and push it to the DIAMBRA registry",
		Long:  `This builds a container image from the given path, then pushes it to the DIAMBRA registry.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				args = []string{"."}
			}

			if credPath == defaultCredPath {
				homedir, err := os.UserHomeDir()
				if err != nil {
					level.Error(logger).Log("msg", "couldn't get homedir", "err", err.Error())
					os.Exit(1)
				}
				credPath = filepath.Join(homedir, ".diambra", "credentials")
			}

			cl, err := client.NewClient(logger, credPath)
			if err != nil {
				level.Error(logger).Log("msg", "failed to create client", "err", err.Error())
				os.Exit(1)
			}

			tag, err := buildAndPush(logger, cl, args[0], name, version)
			if err != nil {
				level.Error(logger).Log("msg", "failed to build and push agent", "err", err)
				os.Exit(1)
			}
			level.Info(logger).Log("msg", fmt.Sprintf("Agent built and pushed: %s", tag), "tag", tag)
		},
		Args: cobra.MaximumNArgs(1),
	}
	cmd.Flags().StringVar(&credPath, "path.credentials", defaultCredPath, "Path to credentials file")
	cmd.Flags().StringVar(&name, "name", name, "Name of the agent image (only used when giving a directory)")
	cmd.Flags().StringVar(&version, "version", version, "Version of the agent image (only used when giving a directory)")

	return cmd
}
