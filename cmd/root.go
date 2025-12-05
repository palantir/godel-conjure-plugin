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
	"github.com/palantir/godel-conjure-plugin/v6/internal/assetloader"
	"github.com/palantir/godel/v2/framework/pluginapi"
	"github.com/palantir/pkg/cobracli"
	"github.com/spf13/cobra"
)

const VerifyFlagName = "verify"

var (
	debugFlagVal      bool
	projectDirFlagVal string
	configFileFlagVal string
	assetsFlagVal     []string

	// loadedAssets is the global set of loaded assets initialized by the PersistentPreRunE
	// function of the root command.
	loadedAssets assetloader.LoadedAssets
)

var rootCmd = &cobra.Command{
	Use:   "conjure-plugin",
	Short: "Run conjure-go based on project configuration",
}

func Execute() int {
	return cobracli.ExecuteWithDebugVarAndDefaultParams(rootCmd, &debugFlagVal)
}

func init() {
	pluginapi.AddDebugPFlagPtr(rootCmd.PersistentFlags(), &debugFlagVal)
	pluginapi.AddProjectDirPFlagPtr(rootCmd.PersistentFlags(), &projectDirFlagVal)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ProjectDirFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddConfigPFlagPtr(rootCmd.PersistentFlags(), &configFileFlagVal)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ConfigFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddAssetsPFlagPtr(rootCmd.PersistentFlags(), &assetsFlagVal)

	// load all assets before running any command
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		loadedAssets, err = assetloader.LoadAssets(assetsFlagVal, cmd.OutOrStdout(), cmd.OutOrStderr())
		if err != nil {
			return err
		}
		return nil
	}
}
