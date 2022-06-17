/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package agent

import (
	"fmt"

	"github.com/spf13/cobra"
)

// submitCmd represents the submit command
var SubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submits an agent for evaluation",
	Long:  `This takes a local agent, builds a container for it and submits it for evaluation.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("submit called")
	},
}