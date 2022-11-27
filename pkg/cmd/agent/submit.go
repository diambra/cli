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
	"os"

	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewSubmitCmd(logger *log.Logger) *cobra.Command {
	var (
		mode         string
		envVars      map[string]string
		sources      map[string]string
		secrets      map[string]string
		manifestPath string
	)
	c, err := diambra.NewConfig(logger)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
	cmd := &cobra.Command{
		Use:   "submit docker-image",
		Short: "Submits an agent for evaluation",
		Long:  `This takes a local agent, builds a container for it and submits it for evaluation.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			image := ""
			if len(args) > 0 {
				image = args[0]
			} else if manifestPath == "" {
				level.Error(logger).Log("msg", "either image or manifest path must be provided")
				os.Exit(1)
			}
			if err := diambra.Submit(logger, image, diambra.Mode(mode), c.Home, envVars, sources, secrets, manifestPath); err != nil {
				level.Error(logger).Log("msg", "failed to submit agent", "err", err.Error())
				os.Exit(1)
			}
			level.Info(logger).Log("msg", "Agent submitted")
		},
	}
	cmd.Flags().StringVar(&mode, "mode", string(diambra.ModeAIvsCOM), "Mode to use for evaluation")
	cmd.Flags().StringToStringVarP(&envVars, "env", "e", envVars, "Environment variables to pass to the agent")
	cmd.Flags().StringToStringVarP(&sources, "source", "u", sources, "Source urls to pass to the agent")
	cmd.Flags().StringToStringVarP(&secrets, "secret", "s", secrets, "Secrets to pass to the agent")
	cmd.Flags().StringVar(&manifestPath, "manifest", manifestPath, "Path to manifest file.")
	return cmd
}
