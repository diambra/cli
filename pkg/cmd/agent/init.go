/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package agent

import (
	"fmt"

	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Prepares local directory as agent for submission",
	Long:  `This creates all files needed to submit an agent.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
	},
}
