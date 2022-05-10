/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent commands",
	Long:  `These are the agent related commands`,
}

func init() {
	rootCmd.AddCommand(agentCmd)
}
