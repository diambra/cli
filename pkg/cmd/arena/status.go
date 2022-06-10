/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package arena

import (
	"fmt"

	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of DIAMBRA arena",
	Long:  `This shows the status of DIAMBRA arena`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("status called")
	},
}
