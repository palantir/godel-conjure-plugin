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
	"path/filepath"

	"github.com/palantir/distgo/distgo"
	"github.com/palantir/distgo/publisher"
	"github.com/palantir/distgo/publisher/artifactory"
	"github.com/palantir/distgo/publisher/maven"
	"github.com/palantir/godel-conjure-plugin/v7/internal/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	publishCmdName = "publish"

	publishTypeScriptNpmUsernameEnv = "CONJURE_TYPESCRIPT_NPM_USERNAME"
	publishTypeScriptNpmPasswordEnv = "CONJURE_TYPESCRIPT_NPM_PASSWORD"
	publishTypeScriptNpmTokenEnv    = "CONJURE_TYPESCRIPT_NPM_TOKEN"
)

var (
	groupIDFlagVal    string
	urlFlagVal        string
	usernameFlagVal   string
	passwordFlagVal   string
	repositoryFlagVal string
	mavenNoPOMFlagVal bool
	dryRunFlagVal     bool

	publishTypeScriptPublishRegistryFlagVal string
	publishTypeScriptInstallRegistryFlagVal string
	publishTypeScriptNpmUsernameFlagVal     string
	publishTypeScriptNpmPasswordFlagVal     string
	publishTypeScriptNpmTokenFlagVal        string
	publishTypeScriptNpmrcFileFlagVal       string
	publishTypeScriptOutputDirFlagVal       string
	publishTypeScriptPackageVersionFlagVal  string
)

var publishCmd = &cobra.Command{
	Use:   publishCmdName,
	Short: "Publish Conjure IR and configured TypeScript clients",
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
			return errors.Wrapf(err, "failed to set working directory")
		}

		publisherFlags, err := conjureplugin.PublisherFlags()
		if err != nil {
			return err
		}

		flagVals := make(map[distgo.PublisherFlagName]any)
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

		extensionsProvider := extensionsprovider.NewAssetsExtensionsProvider(
			loadedAssets.ConjureIRExtensionsProviders, configFileFlagVal, urlFlagVal)

		publishOptions := conjureplugin.PublishParamOptions{
			ProjectDir:         absProjectDir,
			ExtensionsProvider: extensionsProvider,
			GroupIDOverride:    groupIDFlagVal,
		}
		publishParam, err := conjureplugin.NewPublishParam(projectParams, publishOptions)
		if err != nil {
			return err
		}
		if err := conjureplugin.Publish(publishParam, flagVals, dryRunFlagVal, cmd.OutOrStdout()); err != nil {
			return err
		}

		var typeScriptExtensionsProvider extensionsprovider.ExtensionsProvider
		if urlFlagVal != "" && len(loadedAssets.ConjureIRExtensionsProviders) > 0 {
			typeScriptExtensionsProvider = extensionsProvider
		}
		return conjureplugin.PublishTypeScript(projectParams, absProjectDir, conjureplugin.PublishTypeScriptOptions{
			PublishRegistry: publishTypeScriptPublishRegistryFlagVal,
			InstallRegistry: publishTypeScriptInstallRegistryFlagVal,
			NpmUsername:     flagOrEnvironment(publishTypeScriptNpmUsernameFlagVal, publishTypeScriptNpmUsernameEnv),
			NpmPassword:     flagOrEnvironment(publishTypeScriptNpmPasswordFlagVal, publishTypeScriptNpmPasswordEnv),
			NpmToken:        flagOrEnvironment(publishTypeScriptNpmTokenFlagVal, publishTypeScriptNpmTokenEnv),
			NpmrcFile:       publishTypeScriptNpmrcFileFlagVal,
			OutputDir:       publishTypeScriptOutputDirFlagVal,
			PackageVersion:  publishTypeScriptPackageVersionFlagVal,
			DryRun:          dryRunFlagVal,
		}, cmd.OutOrStdout(), cmd.ErrOrStderr(), typeScriptExtensionsProvider, groupIDFlagVal)
	},
}

func flagOrEnvironment(flagValue, environmentVariable string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(environmentVariable)
}

func init() {
	publishCmd.Flags().BoolVar(&dryRunFlagVal, "dry-run", false, "print the operations that would be performed")

	publishCmd.Flags().StringVar(&groupIDFlagVal, string(publisher.GroupIDFlag.Name), "", publisher.GroupIDFlag.Description)
	publishCmd.Flags().StringVar(&repositoryFlagVal, string(artifactory.PublisherRepositoryFlag.Name), "", artifactory.PublisherRepositoryFlag.Description)
	publishCmd.Flags().StringVar(&urlFlagVal, string(publisher.ConnectionInfoURLFlag.Name), "", publisher.ConnectionInfoURLFlag.Description)
	publishCmd.Flags().StringVar(&usernameFlagVal, string(publisher.ConnectionInfoUsernameFlag.Name), "", publisher.ConnectionInfoUsernameFlag.Description)
	publishCmd.Flags().StringVar(&passwordFlagVal, string(publisher.ConnectionInfoPasswordFlag.Name), "", publisher.ConnectionInfoPasswordFlag.Description)
	publishCmd.Flags().BoolVar(&mavenNoPOMFlagVal, string(maven.NoPOMFlag.Name), false, maven.NoPOMFlag.Description)
	publishCmd.Flags().StringVar(&publishTypeScriptPublishRegistryFlagVal, "publish-registry", "", "npm registry URL to publish configured TypeScript packages to")
	publishCmd.Flags().StringVar(&publishTypeScriptInstallRegistryFlagVal, "install-registry", "", "npm registry URL used to install TypeScript build dependencies (defaults to the publish registry)")
	publishCmd.Flags().StringVar(&publishTypeScriptNpmUsernameFlagVal, "npm-username", "", "npm registry username (prefer "+publishTypeScriptNpmUsernameEnv+" in CI)")
	publishCmd.Flags().StringVar(&publishTypeScriptNpmPasswordFlagVal, "npm-password", "", "npm registry password (prefer "+publishTypeScriptNpmPasswordEnv+" in CI)")
	publishCmd.Flags().StringVar(&publishTypeScriptNpmTokenFlagVal, "npm-token", "", "npm registry authentication token (prefer "+publishTypeScriptNpmTokenEnv+" in CI)")
	publishCmd.Flags().StringVar(&publishTypeScriptNpmrcFileFlagVal, "npmrc-file", "", "existing npm configuration file used by TypeScript projects configured with the npmrc publisher provider")
	publishCmd.Flags().StringVar(&publishTypeScriptOutputDirFlagVal, "output-dir", "", "directory in which to write published npm package tarballs (defaults to a new temporary directory)")
	publishCmd.Flags().StringVar(&publishTypeScriptPackageVersionFlagVal, "package-version", "", "exact npm package version override")
	rootCmd.AddCommand(publishCmd)
}
