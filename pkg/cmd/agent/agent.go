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
	"fmt"
	"net/url"
	"time"

	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/diambra/cli/pkg/git"
	"github.com/diambra/cli/pkg/log"
	"github.com/go-kit/log/level"

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
	cmd.AddCommand(NewBuildCmd(logger))
	cmd.AddCommand(NewBuildAndPushCmd(logger))
	return cmd
}

func buildAndPush(logger *log.Logger, client *client.Client, context, name, version string) (string, error) {
	var err error
	if name == "" {
		name, err = container.TagFromDir(context)
		if err != nil {
			return "", fmt.Errorf("failed to get tag from dir: %w", err)
		}
	}

	if version == "" {
		version, err = git.GitHeadSHAShort(context, 0)
		if err != nil {
			version = time.Now().Format("20060102-150405")
			level.Warn(logger).Log("msg", fmt.Sprintf("failed to get git head sha, using timestamp %s", version), "err", err)
		}
	}

	credentials, err := client.Credentials()
	if err != nil {
		return "", fmt.Errorf("failed to get credentials: %w", err)
	}

	level.Info(logger).Log("msg", "Building agent", "name", name, "version", version)
	runner, err := container.NewDockerRunner(logger, false)
	if err != nil {
		return "", fmt.Errorf("failed to create docker runner: %w", err)
	}

	repositoryURL, err := url.Parse(credentials.Repository)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %w", err)
	}

	runner.Login(credentials.Username, credentials.Password, repositoryURL.Host)

	tag := fmt.Sprintf("%s%s:%s-%s", repositoryURL.Host, repositoryURL.Path, name, version)

	if exists, err := runner.TagExists(tag); err != nil {
		return "", fmt.Errorf("failed to check if tag exists: %w", err)
	} else if exists {
		return "", fmt.Errorf("tag %s already exists, use --name or --version to specify unused tag", tag)
	}

	if err := runner.Build(context, tag); err != nil {
		return "", fmt.Errorf("failed to build agent: %w", err)
	}
	if err := runner.Push(tag); err != nil {
		return "", fmt.Errorf("failed to push agent: %w", err)
	}
	return tag, nil
}
