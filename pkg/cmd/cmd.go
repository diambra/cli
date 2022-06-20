/*
Copyright © 2022 DIAMBRA <info@diambra.ai>

*/
package cmd

import (
	"os"

	"github.com/diambra/cli/pkg/cmd/agent"
	"github.com/diambra/cli/pkg/cmd/arena"
	"github.com/diambra/cli/pkg/log"
	"github.com/go-kit/log/level"

	"github.com/spf13/cobra"
)

func NewDiambraCommand() *cobra.Command {
	var (
		logger = &log.Logger{}

		logFormat = ""
		debug     = false
		cmd       = &cobra.Command{
			Use:   "diambra",
			Short: "The DIAMBRA cli",
			Long: `Quickstart:
- Run 'diambra agent init' to create a example agent.
- Run 'diambra run ./agent.py' to bring up DIAMBRA arena and run agent.py
`,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				if err := logger.SetOptions(debug, logFormat); err != nil {
					level.Error(logger).Log("msg", err.Error())
					os.Exit(1)
				}
			},
		}
	)

	cmd.PersistentFlags().BoolVarP(&debug, "log.debug", "d", false, "Enable debug logging")
	cmd.PersistentFlags().StringVar(&logFormat, "log.format", "fancy", "Set logging output format (logfmt, json, fancy)")
	cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	cmd.AddCommand(NewCmdRun(logger))
	cmd.AddCommand(agent.NewCommand(logger))
	cmd.AddCommand(arena.NewCommand(logger))
	return cmd
}
