/*
Copyright © 2022 DIAMBRA <info@diambra.ai>

*/
package cmd

import (
	"github.com/diambra/cli/pkg/cmd/agent"
	"github.com/diambra/cli/pkg/cmd/arena"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewDiambraCommand(logger log.Logger) *cobra.Command {
	var (
		debug = false
		cmd   = &cobra.Command{
			Use:   "diambra",
			Short: "The DIAMBRA cli",
			Long: `Quickstart:
- Run 'diambra agent init' to create a example agent.
- Run 'diambra run ./agent.py' to bring up DIAMBRA arena and run agent.py
`,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				if debug {
					level.Info(logger).Log("msg", "debug enabled")
					logger = level.NewFilter(logger, level.AllowDebug())
				} else {
					logger = level.NewFilter(logger, level.AllowInfo())
				}
				logger = log.With(logger, "caller", log.Caller(3))

			},
		}
	)

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	cmd.AddCommand(NewCmdRun(logger))
	cmd.AddCommand(agent.NewCommand(logger))
	cmd.AddCommand(arena.NewCommand(logger))
	return cmd
}
