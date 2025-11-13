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
	"os"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var acceptBackcompatBreaksCmd = &cobra.Command{
	Use:   "accept-backcompat-breaks",
	Short: "Accept current backward compatibility breaks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if loadedAssets.ConjureBackcompat != nil {
			return nil
		}

		projectParams, err := toProjectParams(configFileFlagVal, cmd.OutOrStdout())
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlagVal); err != nil {
			return errors.Wrapf(err, "failed to set working directory")
		}

		if err := projectParams.ForEach(func(project string, param conjureplugin.ConjureProjectParam) error {
			if !param.Publish {
				return nil
			}

			bytes, err := param.IRProvider.IRBytes()
			if err != nil {
				return err
			}

			file, err := tempfilecreator.WriteBytesToTempFile(bytes)
			if err != nil {
				return err
			}

			return loadedAssets.ConjureBackcompat.AcceptBackCompatBreaks(param.GroupID, project, file, projectDirFlagVal)
		}); err != nil {
			return fmt.Errorf(`failed to accept conjure breaks: %w\nto accept breaks run "./godelw conjure-accept-backcompat-breaks"`, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(acceptBackcompatBreaksCmd)
}
