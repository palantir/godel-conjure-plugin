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
	"strings"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/spf13/cobra"
)

var acceptBackcompatBreaksCmd = &cobra.Command{
	Use:   "accept-backcompat-breaks",
	Short: "Accept current backward compatibility breaks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackCompatCommand(
			cmd,
			func(project string, param conjureplugin.ConjureProjectParam, irFile string) error {
				return loadedAssets.ConjureBackcompat.AcceptBackCompatBreaks(param.GroupID, project, irFile, projectDirFlagVal)
			},
			func(failedProjects map[string]error) error {
				projects := slices.Collect(maps.Keys(failedProjects))

				if len(projects) == 1 {
					return fmt.Errorf("failed to accept conjure breaks for project %q", projects[0])
				}

				slices.Sort(projects)

				return fmt.Errorf("failed to accept conjure breaks for projects: %s", strings.Join(projects, ", "))
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(acceptBackcompatBreaksCmd)
}
