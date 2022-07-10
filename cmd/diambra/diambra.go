/*
Copyright © 2022 DIAMBRA <info@diambra.ai>

*/
package main

import (
	"fmt"
	"os"

	"github.com/diambra/cli/pkg/cmd"
)

func main() {
	if err := cmd.NewDiambraCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
