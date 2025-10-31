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
	backcompatvalidator "github.com/palantir/godel-conjure-plugin/v6/internal/backcompat-validator"
	"github.com/palantir/godel/v2/framework/pluginapi"
	"github.com/palantir/pkg/cobracli"
	"github.com/spf13/cobra"
)

const VerifyFlagName = "verify"

var (
	debugFlagVal   bool
	projectDirFlag string
	configFileFlag string
	assetsFlag     []string

	// backcompatAsset is initialized once when the plugin starts
	backcompatAsset *backcompatvalidator.BackCompatAsset
)

var rootCmd = &cobra.Command{
	Use:   "conjure-plugin",
	Short: "Run conjure-go based on project configuration",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Discover and validate backcompat assets once when the plugin starts
		// This ensures we fail fast if there are any issues with asset configuration
		var err error
		backcompatAsset, err = backcompatvalidator.New(configFileFlag, assetsFlag)
		return err
	},
}

func Execute() int {
	return cobracli.ExecuteWithDebugVarAndDefaultParams(rootCmd, &debugFlagVal)
}

func init() {
	pluginapi.AddDebugPFlagPtr(rootCmd.PersistentFlags(), &debugFlagVal)
	pluginapi.AddProjectDirPFlagPtr(rootCmd.PersistentFlags(), &projectDirFlag)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ProjectDirFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddConfigPFlagPtr(rootCmd.PersistentFlags(), &configFileFlag)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ConfigFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddAssetsPFlagPtr(rootCmd.PersistentFlags(), &assetsFlag)
}
