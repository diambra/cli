/*
Copyright © 2022 DIAMBRA <info@diambra.ai>

*/
package main

import (
	"github.com/diambra/cli/pkg/cmd"
	"github.com/diambra/cli/pkg/log"
)

func main() {
	cmd.NewDiambraCommand(log.New()).Execute()
}
