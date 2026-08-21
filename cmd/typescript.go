// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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
	"path/filepath"

	"github.com/palantir/distgo/publisher"
	"github.com/palantir/godel-conjure-plugin/v7/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const typeScriptCmdName = "typescript"

var (
	typeScriptOutputDirFlagVal      string
	typeScriptPackageVersionFlagVal string
	typeScriptGroupIDFlagVal        string
	typeScriptURLFlagVal            string
)

var typeScriptCmd = &cobra.Command{
	Use:   typeScriptCmdName,
	Short: "Generate and package Conjure TypeScript clients",
	Long: "Generate, build, and create an npm package tarball for every Conjure project that declares a " +
		"\"typescript\" configuration block. Requires \"node\" and \"npm\" to be available on the PATH.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectParams, err := toProjectParams(configFileFlagVal, cmd.OutOrStdout())
		if err != nil {
			return err
		}
		absProjectDir, err := filepath.Abs(projectDirFlagVal)
		if err != nil {
			return errors.Wrap(err, "failed to resolve project directory")
		}
		if err := os.Chdir(absProjectDir); err != nil {
			return errors.Wrap(err, "failed to set working directory")
		}
		// Conjure IR extension-provider assets need a base URL to resolve against (matching conjure-publish),
		// so they are only invoked when both a URL and at least one asset is configured. It is valid to run
		// without the URL/extensions-provider, but the package will be generated without embedded product dependencies.
		var extensionsProvider extensionsprovider.ExtensionsProvider
		if len(loadedAssets.ConjureIRExtensionsProviders) > 0 {
			if typeScriptURLFlagVal == "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[WARNING]: %d Conjure IR extension provider asset(s) configured but --url not provided, packaging without resolving product dependencies\n", len(loadedAssets.ConjureIRExtensionsProviders))
			} else {
				extensionsProvider = extensionsprovider.NewAssetsExtensionsProvider(
					loadedAssets.ConjureIRExtensionsProviders,
					configFileFlagVal,
					typeScriptURLFlagVal,
				)
			}
		}

		opts := conjureplugin.PackageTypeScriptOptions{
			OutputDir:      typeScriptOutputDirFlagVal,
			PackageVersion: typeScriptPackageVersionFlagVal,
		}
		_, err = conjureplugin.PackageTypeScript(projectParams, absProjectDir, opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), extensionsProvider, typeScriptGroupIDFlagVal)
		return err
	},
}

func init() {
	typeScriptCmd.Flags().StringVar(&typeScriptOutputDirFlagVal, "output-dir", "", "output directory to write npm package tarballs (defaults to a new temporary directory)")
	typeScriptCmd.Flags().StringVar(&typeScriptPackageVersionFlagVal, "package-version", "", "exact npm package version override")
	typeScriptCmd.Flags().StringVar(&typeScriptGroupIDFlagVal, string(publisher.GroupIDFlag.Name), "", "group ID passed to Conjure IR extension providers (overrides project configuration)")
	typeScriptCmd.Flags().StringVar(&typeScriptURLFlagVal, string(publisher.ConnectionInfoURLFlag.Name), "", "base URL passed to Conjure IR extension providers (providers are skipped when omitted)")
	rootCmd.AddCommand(typeScriptCmd)
}
