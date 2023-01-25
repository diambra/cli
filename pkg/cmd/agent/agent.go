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

package agent

import (
	"github.com/diambra/cli/pkg/log"

	"github.com/spf13/cobra"
)

func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent commands",
		Long:  `These are the agent related commands`,
	}
	cmd.AddCommand(NewInitCmd(logger))
	cmd.AddCommand(NewSubmitCmd(logger))
	cmd.AddCommand(NewTestCmd(logger))
	return cmd
}
