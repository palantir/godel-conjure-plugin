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

	"github.com/palantir/godel-conjure-plugin/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/conjureplugin/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	verifyFlag bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run conjure-go based on project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		parsedConfigSet, err := toProjectParams(configFileFlag)
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlag); err != nil {
			return errors.Wrapf(err, "failed to set working directory")
		}
		return conjureplugin.Run(parsedConfigSet, verifyFlag, projectDirFlag, cmd.OutOrStdout())
	},
}

func init() {
	runCmd.Flags().BoolVar(&verifyFlag, VerifyFlagName, false, "verify that current project matches output of conjure")
	rootCmd.AddCommand(runCmd)
}

func toProjectParams(cfgFile string) (conjureplugin.ConjureProjectParams, error) {
	config, err := config.ReadConfigFromFile(cfgFile)
	if err != nil {
		return conjureplugin.ConjureProjectParams{}, err
	}
	return config.ToParams()
}
