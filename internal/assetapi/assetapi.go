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

package assetapi

import (
	"encoding/json"

	"github.com/palantir/godel-conjure-plugin/v6/assetapi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Deprecated: assets of this type are deprecated and no longer recommended. The plugin will continue to support
// them for backwards compatibility, but new projects should use an equivalent ConjureExtensionsProvider asset
// instead.
//
// New asset types should be declared as exported constants in the assetapi package. However, because this asset type is
// deprecated and does not use the assetapi interface, its declaration is in the internal assetapi package.
const ConjureIRExtensionsProvider assetapi.AssetType = "conjure-ir-extensions-provider"

// AllAssetsTypes returns a slice of all supported asset types.
func AllAssetsTypes() []assetapi.AssetType {
	return []assetapi.AssetType{
		assetapi.ConjureBackCompat,
		ConjureIRExtensionsProvider,
	}
}

// NewAssetRootCmd returns a new cobra.Command for the asset of the specified type. The provided name and description
// are used to set the name and description for the CLI, but are not directly used as part of the asset API. The
// returned root command has the AssetTypeCommand subcommand registered. When called, this command prints the JSON
// string representation of the assetType, which satisfied the asset discovery API.
//
// Asset types should generally define an exported function of their own that returns a new *cobra.Command that is
// derived from this one and adds the asset-specific commands that are expected.
func NewAssetRootCmd(assetType assetapi.AssetType, name, description string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   name,
		Short: description,
	}

	rootCmd.AddCommand(newAssetTypeCmd(assetType))

	return rootCmd
}

const AssetTypeCommand = "conjure-plugin-asset-type"

func newAssetTypeCmd(assetType assetapi.AssetType) *cobra.Command {
	return &cobra.Command{
		Use:   AssetTypeCommand,
		Short: "Prints the JSON representation of the asset type",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, err := json.Marshal(assetType)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal JSON")
			}

			_, err = cmd.OutOrStdout().Write(jsonOutput)
			if err != nil {
				return errors.Wrapf(err, "failed write JSON to stdout")
			}

			return nil
		},
	}
}
