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
	"fmt"
	"maps"
	"slices"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/internal/backcompatasset"
	"github.com/spf13/cobra"
)

const (
	acceptBackCompatBreaksCmdName = "accept-backcompat-breaks"
)

var acceptBackcompatBreaksCmd = &cobra.Command{
	Use:   acceptBackCompatBreaksCmdName,
	Short: "Accept current backward compatibility breaks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackCompatCommand(
			cmd,
			func(project string, param conjureplugin.ConjureProjectParam, irFile string, execParam backcompatasset.ExecParam) error {
				return loadedAssets.ConjureBackcompat.AcceptBackCompatBreaks(param.GroupID, project, irFile, projectDirFlagVal, execParam)
			},
			func(failedProjects map[string]error) error {
				projects := slices.Collect(maps.Keys(failedProjects))
				slices.Sort(projects)

				msg := "failed to accept conjure breaks for projects:\n"
				for _, project := range projects {
					msg += fmt.Sprintf("  %q: %v\n", project, failedProjects[project])
				}
				return fmt.Errorf("%s", msg)
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(acceptBackcompatBreaksCmd)
}
