/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package agent

import (
	"github.com/go-kit/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent commands",
		Long:  `These are the agent related commands`,
	}
	cmd.AddCommand(InitCmd)
	cmd.AddCommand(SubmitCmd)
	return cmd
}
