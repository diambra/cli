/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package arena

import (
	"fmt"

	"github.com/spf13/cobra"
)

// upCmd represents the up command
var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start DIAMBRA arena",
	Long:  `This command starts DIAMBRA arena in the background`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("up called")
	},
}
