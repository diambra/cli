/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package arena

import (
	"fmt"

	"github.com/spf13/cobra"
)

var DownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop DIAMBRA Arena",
	Long:  `This stops a DIAMBRA Arena running in the background.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("down called")
	},
}
