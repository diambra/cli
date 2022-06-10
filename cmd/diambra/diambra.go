/*
Copyright © 2022 DIAMBRA <info@diambra.ai>

*/
package main

import (
	"os"

	"github.com/diambra/cli/pkg/cmd"
	"github.com/go-kit/log"
)

func main() {
	cmd.NewDiambraCommand(log.NewLogfmtLogger(os.Stderr)).Execute()
}
