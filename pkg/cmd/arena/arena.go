/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package arena

import (
	"github.com/diambra/cli/pkg/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arena",
		Short: "Arena commands",
		Long:  `These are the arena related commands`,
	}
	cmd.AddCommand(DownCmd)
	cmd.AddCommand(StatusCmd)
	return cmd
}
