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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type AssetType string

const (
	// ConjureExtensionsProvider represents an asset that provides Conjure extensions when generating Conjure IR from
	// YAML.
	ConjureExtensionsProvider AssetType = "conjure-extensions-provider"

	// Deprecated: assets of this type are deprecated and no longer recommended. The plugin will continue to support
	// them for backwards compatibility, but new projects should use an equivalent ConjureExtensionsProvider asset
	// instead.
	ConjureIRExtensionsProvider AssetType = "conjure-ir-extensions-provider"
)

// AllAssetsTypes returns a slice of all supported asset types.
func AllAssetsTypes() []AssetType {
	return []AssetType{
		ConjureExtensionsProvider,
		ConjureIRExtensionsProvider,
	}
}

func AssetRootCmd(assetType AssetType, name, description string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   name,
		Short: description,
	}

	rootCmd.AddCommand(newAssetTypeCmd(assetType))

	return rootCmd
}

const AssetTypeCommand = "conjure-plugin-asset-type"

func newAssetTypeCmd(assetType AssetType) *cobra.Command {
	return &cobra.Command{
		Use:   AssetTypeCommand,
		Short: "Prints the JSON representation of the asset type",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, err := json.Marshal(assetType)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal JSON")
			}
			cmd.Print(string(jsonOutput))
			return nil
		},
	}
}
