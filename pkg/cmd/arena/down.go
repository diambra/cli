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

package arena

import (
	"os"

	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/log"
	"github.com/docker/docker/client"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewDownCmd(logger *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop DIAMBRA Arena",
		Long:  `This stops a DIAMBRA Arena running in the background.`,
		Run: func(_ *cobra.Command, _ []string) {

			client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				level.Error(logger).Log("msg", "failed to create docker client", "err", err.Error())
				os.Exit(1)
			}
			runner := container.NewDockerRunner(logger, client, true)
			if err := runner.StopAll(); err != nil {
				level.Error(logger).Log("msg", "failed to stop all containers", "err", err.Error())
				os.Exit(1)
			}
		},
	}
}
