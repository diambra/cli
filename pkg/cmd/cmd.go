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

package cmd

import (
	"os"

	"github.com/diambra/cli/pkg/cmd/agent"
	"github.com/diambra/cli/pkg/cmd/arena"
	"github.com/diambra/cli/pkg/log"
	"github.com/diambra/cli/pkg/version"
	"github.com/go-kit/log/level"

	"github.com/spf13/cobra"
)

func NewDiambraCommand() *cobra.Command {
	var (
		logger = log.New()

		logFormat = ""
		debug     = false
		cmd       = &cobra.Command{
			Use:   "diambra",
			Short: "The DIAMBRA cli",
			Long: `Quickstart:
- Run 'diambra agent init path/to/agent' to create a example agent.
- Run 'diambra run path/to/agent/agent.py' to bring up DIAMBRA arena and run agent.py
- Run 'docker build -t registry/user/agent:latest path/to/agent' to build your agent's Docker image
- Run 'diambra agent submit registry/user/agent:latest' to submit your agent to DIAMBRA
`,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				if err := logger.SetOptions(debug, logFormat); err != nil {
					level.Error(logger).Log("msg", err.Error())
					os.Exit(1)
				}
			},
			Version: version.String(),
		}
	)

	cmd.PersistentFlags().BoolVarP(&debug, "log.debug", "d", false, "Enable debug logging")
	cmd.PersistentFlags().StringVar(&logFormat, "log.format", "fancy", "Set logging output format (logfmt, json, fancy)")

	cmd.AddCommand(NewCmdRun(logger))
	cmd.AddCommand(agent.NewCommand(logger))
	cmd.AddCommand(arena.NewCommand(logger))
	return cmd
}
