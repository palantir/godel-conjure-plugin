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
	"os"

	"github.com/palantir/distgo/distgo"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	dryRunFlagVal bool
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish Conjure IR",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectParams, err := toProjectParams(configFileFlag)
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlag); err != nil {
			return errors.Wrapf(err, "failed to set working directory")
		}

		publisherFlags, err := conjureplugin.PublisherFlags()
		if err != nil {
			return err
		}

		flagVals := make(map[distgo.PublisherFlagName]interface{})
		for _, currFlag := range publisherFlags {
			// if flag was not explicitly provided, don't add it to the flagVals map
			if !cmd.Flags().Changed(string(currFlag.Name)) {
				continue
			}
			val, err := currFlag.GetFlagValue(cmd.Flags())
			if err != nil {
				return err
			}
			flagVals[currFlag.Name] = val
		}
		return conjureplugin.Publish(projectParams, projectDirFlag, flagVals, dryRunFlagVal, cmd.OutOrStdout())
	},
}

func init() {
	publishCmd.Flags().BoolVar(&dryRunFlagVal, "dry-run", false, "print the operations that would be performed")

	rootCmd.AddCommand(publishCmd)
}
