/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

var arenaCmd = &cobra.Command{
	Use:   "arena",
	Short: "Arena commands",
	Long:  `These are the arena related commands`,
}

func init() {
	rootCmd.AddCommand(arenaCmd)
}
