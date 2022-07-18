/*
 * Copyright 2022 The DIAMBRA Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package arena

import (
	"github.com/diambra/cli/pkg/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arena",
		Short: "Arena commands",
		Long:  `These are the arena related commands`,
	}
	cmd.AddCommand(NewUpCmd(logger))
	cmd.AddCommand(NewDownCmd(logger))
	cmd.AddCommand(StatusCmd)
	return cmd
}
