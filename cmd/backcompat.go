// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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
	"github.com/palantir/distgo/publisher"
	"github.com/palantir/distgo/publisher/artifactory"
	"os"

	"github.com/palantir/godel-conjure-plugin/v5/conjureplugin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	groupFlagVal          string
	repositoryNameFlagVal string
	artifactoryUrlFlagVal string
)

var backcompatCmd = &cobra.Command{
	Use:   "checkConjureBackCompat",
	Short: "Run conjure-backcompat based on project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		parsedConfigSet, err := toProjectParams(configFileFlag)
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlag); err != nil {
			return errors.Wrapf(err, "failed to set working directory")
		}

		return conjureplugin.BackCompat(parsedConfigSet, projectDirFlag, groupFlagVal, repositoryNameFlagVal, artifactoryUrlFlagVal, cmd.OutOrStdout())
	},
}

func init() {
	backcompatCmd.Flags().StringVar(&groupFlagVal, string(publisher.GroupIDFlag.Name), "", publisher.GroupIDFlag.Description)
	backcompatCmd.Flags().StringVar(&repositoryNameFlagVal, string(artifactory.PublisherRepositoryFlag.Name), "", artifactory.PublisherRepositoryFlag.Description)
	backcompatCmd.Flags().StringVar(&artifactoryUrlFlagVal, string(publisher.ConnectionInfoURLFlag.Name), "", publisher.ConnectionInfoURLFlag.Description)

	rootCmd.AddCommand(backcompatCmd)
}
