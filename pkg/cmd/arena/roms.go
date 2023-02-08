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
	"os/exec"
	"strings"

	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/log"
	"github.com/diambra/cli/pkg/pyarena"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewScriptCmd(logger *log.Logger, name, script string, c *diambra.EnvConfig) *cobra.Command {
	var pythonPath string
	cmd := &cobra.Command{
		Use:   strings.ReplaceAll(name, " ", "-"),
		Short: name,
		Long:  "This command runs the " + name + " rom utility: " + script,
		Run: func(_ *cobra.Command, args []string) {
			level.Debug(logger).Log("msg", name)
			cmd := exec.Command(pythonPath, "-c", script)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Args = append(cmd.Args, args...)
			cmd.Env = append(os.Environ(),
				"DIAMBRAROMSPATH="+c.RomsPath,
			)
			if err := cmd.Run(); err != nil {
				level.Error(logger).Log("msg", "command failed", "err", err.Error())
				os.Exit(1)
			}
		},
	}
	c.AddRomsPathFlag(cmd.Flags())
	cmd.Flags().StringVar(&pythonPath, "python", pyarena.FindPython(), "Path to python executable")
	return cmd
}

func NewRomCmds(logger *log.Logger) ([]*cobra.Command, error) {
	c, err := diambra.NewConfig(logger)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}

	return []*cobra.Command{
		NewScriptCmd(logger, "check roms", pyarena.CheckRoms, c),
		NewScriptCmd(logger, "list roms", pyarena.ListRoms, c),
		NewScriptCmd(logger, "version", pyarena.GetDiambraEngineVersion, c),
	}, nil
}
