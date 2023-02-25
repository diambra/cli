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

package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/diambra/agents"
	"github.com/diambra/cli/pkg/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewInitCmd(logger *log.Logger) *cobra.Command {
	config, err := agents.NewConfig()
	if err != nil {
		level.Error(logger).Log("msg", "failed to create config", "err", err)
		os.Exit(1)
	}
	cmd := &cobra.Command{
		Use:   "init path/to/agent",
		Short: "Prepares local directory as agent for submission",
		Long:  `This creates all files needed to submit an agent.`,
		Run: func(cmd *cobra.Command, args []string) {
			parts, err := diambra.GetInstalledPackageVersion("diambra-arena")
			if err != nil || len(parts) != 3 || (parts[0] == "0" && parts[1] == "0" && parts[2] == "0") {
				level.Info(logger).Log("msg", "can't find installed diambra-arena version, using latest", "err", err)
				parts, err = diambra.GetLatestDiambraArenaVersion()
				if err != nil {
					level.Error(logger).Log("msg", "failed to get latest diambra arena version", "err", err.Error())
					os.Exit(1)
				}
			}
			config.Arena.Version = strings.Join(parts, ".")
			if err := agents.Generate(logger, args[0], config); err != nil {
				level.Error(logger).Log("msg", "failed to initialize agent", "err", err.Error())
				os.Exit(1)
			}
			level.Info(logger).Log("msg", fmt.Sprintf("Agent initialized in %s", args[0]))
		},
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().StringVar(&config.Python.Version, "python.version", config.Python.Version, "Python version to use")
	cmd.Flags().BoolVar(&config.Secret, "secret", config.Secret, "Include secret in agent")

	return cmd
}
