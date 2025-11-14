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

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/spf13/cobra"
)

const acceptBackCompatBreaksCmdName = "accept-backcompat-breaks"

var acceptBackcompatBreaksCmd = &cobra.Command{
	Use:   acceptBackCompatBreaksCmdName,
	Short: "Accept current backward compatibility breaks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackCompatCommand(
			cmd,
			func(project string, param conjureplugin.ConjureProjectParam, irFile string) error {
				return loadedAssets.ConjureBackcompat.AcceptBackCompatBreaks(param.GroupID, project, irFile, projectDirFlagVal)
			},
			func(err error) error {
				return fmt.Errorf(`failed to accept conjure breaks: %w`, err)
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(acceptBackcompatBreaksCmd)
}
