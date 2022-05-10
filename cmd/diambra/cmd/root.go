/*
Copyright © 2022 DIAMBRA <info@diambra.ai>

*/
package cmd

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

var (
	logger = log.NewLogfmtLogger(os.Stderr)
	debug  = false

	rootCmd = &cobra.Command{
		Use:   "diambra",
		Short: "The DIAMBRA cli",
		Long: `Quickstart:
- Run 'diambra agent init' to create a example agent.
- Run 'diambra run ./agent.py' to bring up DIAMBRA arena and run agent.py
`,
	}
)

func Execute() {
	if debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	logger = log.With(logger, "caller", log.Caller(3))

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
