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
	"os"
	"path/filepath"

	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/diambra/cli/pkg/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewSubmitCmd(logger *log.Logger) *cobra.Command {
	var (
		dump             = false
		submissionConfig = diambra.SubmissionConfig{}
		name             = ""
		version          = ""
	)

	c, err := diambra.NewConfig(logger)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
	submissionConfig.RegisterCredentialsProviders(logger, c.Home)

	cmd := &cobra.Command{
		Use:   "submit [flags] (directory | --submission.manifest=submission-manifest.yaml | docker-image) [args/command(s) ...]",
		Short: "Submits an agent for evaluation",
		Long:  `This takes a directory, existing docker image or submission manifest and submits it for evaluation.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := diambra.EnsureCredentials(logger, c.CredPath); err != nil {
				level.Error(logger).Log("msg", err.Error())
				os.Exit(1)
			}
			submission, err := submissionConfig.Submission(c, args)
			if err != nil {
				level.Error(logger).Log("msg", "failed to configure manifest", "err", err.Error())
				os.Exit(1)
			}
			if dump {
				b, err := yaml.Marshal(submission)
				if err != nil {
					level.Error(logger).Log("msg", "failed to marshal manifest", "err", err.Error())
					os.Exit(1)
				}
				fmt.Println(string(b))
				return
			}

			cl, err := client.NewClient(logger, c.CredPath)
			if err != nil {
				level.Error(logger).Log("msg", "failed to create client", "err", err.Error())
				os.Exit(1)
			}
			// If submission.Image is a directory, we build and push it, then update the name to the resulting image
			if stat, err := os.Stat(submission.Manifest.Image); err == nil && stat.IsDir() {
				context := submission.Manifest.Image
				level.Info(logger).Log("msg", "Building and pushing image", "context", context)
				tag, err := buildAndPush(logger, cl, context, name, version)
				if err != nil {
					level.Error(logger).Log("msg", "failed to build and push agent", "err", err.Error())
					os.Exit(1)
				}

				submission.Manifest.Image = tag
			} else {
				level.Warn(logger).Log("msg", "Using existing images or submission manifest is not recommended and might get deprecated in the future")
			}

			id, err := cl.Submit(submission)
			if err != nil {
				level.Error(logger).Log("msg", "failed to submit agent", "err", err.Error())
				os.Exit(1)
			}
			level.Info(logger).Log("msg", fmt.Sprintf("Agent submitted: https://diambra.ai/submission/%d", id), "id", id)
		},
	}
	submissionConfig.AddFlags(cmd.Flags())
	// FIXME: Split this out of EnvConfig
	cmd.Flags().StringVar(&c.CredPath, "path.credentials", filepath.Join(c.Home, ".diambra/credentials"), "Path to credentials file")
	cmd.Flags().BoolVar(&dump, "dump", false, "Dump the manifest to stdout instead of submitting")
	cmd.Flags().SetInterspersed(false)
	cmd.Flags().StringVar(&name, "name", name, "Name of the agent image (only used when giving a directory)")
	cmd.Flags().StringVar(&version, "version", version, "Version of the agent image (only used when giving a directory)")
	return cmd
}
