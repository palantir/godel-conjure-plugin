// Copyright (c) 2025 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	backcompatvalidator "github.com/palantir/godel-conjure-plugin/v6/internal/backcompat-validator"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/spf13/cobra"
)

var acceptBackcompatBreaksCmd = &cobra.Command{
	Use:   "accept-backcompat-breaks",
	Short: "Accept backward compatibility breaks by writing lockfile entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackcompatOperation(
			cmd.OutOrStdout(),
			projectFlag,
			func(asset *backcompatvalidator.BackCompatAsset, projectName string, param conjureplugin.ConjureProjectParam, projectDir string) error {
				return asset.AcceptBackCompatBreaks(projectName, param, projectDir)
			},
			"accept backcompat breaks",
		)
	},
}

func init() {
	acceptBackcompatBreaksCmd.Flags().StringVar(&projectFlag, "project", "", "accept breaks for a specific project")
	rootCmd.AddCommand(acceptBackcompatBreaksCmd)
}
