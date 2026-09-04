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
)

var (
	groupIDFlagVal    string
	urlFlagVal        string
	usernameFlagVal   string
	passwordFlagVal   string
	repositoryFlagVal string
	mavenNoPOMFlagVal bool
	dryRunFlagVal     bool

	npmPublishRegistryFlagVal string
	npmInstallRegistryFlagVal string
	npmTokenFlagVal           string
)

var publishCmd = &cobra.Command{
	Use:   publishCmdName,
	Short: "Publish Conjure IR and TypeScript npm packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectParams, err := toProjectParams(configFileFlagVal, cmd.OutOrStdout())
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlagVal); err != nil {
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

		publishOptions := conjureplugin.PublishParamOptions{
			ProjectDir:         projectDirFlagVal,
			ExtensionsProvider: extensionsprovider.NewAssetsExtensionsProvider(loadedAssets.ConjureIRExtensionsProviders, configFileFlagVal, urlFlagVal),
			GroupIDOverride:    groupIDFlagVal,
		}
		publishParam, err := conjureplugin.NewPublishParam(projectParams, publishOptions)
		if err != nil {
			return err
		}

		npmUsername := usernameFlagVal
		npmPassword := passwordFlagVal
		if npmTokenFlagVal != "" {
			// A token overrides the shared username/password for npm only.
			npmUsername = ""
			npmPassword = ""
		}
		publishTypescriptOpts := conjureplugin.PublishTypeScriptOptions{
			PublishRegistry: npmPublishRegistryFlagVal,
			InstallRegistry: npmInstallRegistryFlagVal,
			NpmUsername:     npmUsername,
			NpmPassword:     npmPassword,
			NpmToken:        npmTokenFlagVal,
		}
		// Prepare typescript packages for publishing before publishing Conjure IR so that a misconfigured npm registry or a packaging failure
		// is caught before any artifacts are uploaded. This is a noop if no typescript packages are declared or publishing of typescript packages is explicitly disabled.
		preparedTypeScript, err := conjureplugin.PrepareTypeScriptPublish(publishParam.TypeScript, publishTypescriptOpts, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		defer preparedTypeScript.Cleanup()

		if err := conjureplugin.Publish(publishParam, flagVals, dryRunFlagVal, cmd.OutOrStdout()); err != nil {
			return err
		}

		return conjureplugin.PublishPreparedTypeScript(preparedTypeScript, dryRunFlagVal, cmd.OutOrStdout())
	},
}

func init() {
	publishCmd.Flags().BoolVar(&dryRunFlagVal, "dry-run", false, "print the operations that would be performed")

	publishCmd.Flags().StringVar(&groupIDFlagVal, string(publisher.GroupIDFlag.Name), "", publisher.GroupIDFlag.Description)
	publishCmd.Flags().StringVar(&repositoryFlagVal, string(artifactory.PublisherRepositoryFlag.Name), "", artifactory.PublisherRepositoryFlag.Description)
	publishCmd.Flags().StringVar(&urlFlagVal, string(publisher.ConnectionInfoURLFlag.Name), "", publisher.ConnectionInfoURLFlag.Description)
	publishCmd.Flags().StringVar(&usernameFlagVal, string(publisher.ConnectionInfoUsernameFlag.Name), "", publisher.ConnectionInfoUsernameFlag.Description)
	publishCmd.Flags().StringVar(&passwordFlagVal, string(publisher.ConnectionInfoPasswordFlag.Name), "", publisher.ConnectionInfoPasswordFlag.Description)
	publishCmd.Flags().BoolVar(&mavenNoPOMFlagVal, string(maven.NoPOMFlag.Name), false, maven.NoPOMFlag.Description)

	publishCmd.Flags().StringVar(&npmPublishRegistryFlagVal, "npm-publish-registry", "", "npm registry URL to publish to")
	publishCmd.Flags().StringVar(&npmInstallRegistryFlagVal, "npm-install-registry", "", "npm registry URL used to install dependencies (defaults to the publish registry)")
	publishCmd.Flags().StringVar(&npmTokenFlagVal, "npm-token", "", "npm registry authentication token (bypasses Artifactory authentication)")
	rootCmd.AddCommand(publishCmd)
}
