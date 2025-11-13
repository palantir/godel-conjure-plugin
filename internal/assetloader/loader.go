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

package assetloader

import (
	"errors"
	"io"
	"os/exec"
	"slices"

	"github.com/palantir/godel-conjure-plugin/v6/assetapi"
	assetapiinternal "github.com/palantir/godel-conjure-plugin/v6/internal/assetapi"
	"github.com/palantir/godel-conjure-plugin/v6/internal/backcompatasset"
	"github.com/palantir/godel-conjure-plugin/v6/internal/cmdutils"
	pkgerrors "github.com/pkg/errors"
)

type LoadedAssets struct {
	ConjureBackcompat            backcompatasset.BackCompatChecker
	ConjureIRExtensionsProviders []string
}

// LoadAssets takes a list of asset paths and returns a LoadedAssets struct that contains the typed assets. Returns an
// error if any of the provided assets are not valid according to the plugin asset specification.
// In addition to general validation, this function performs the following additional checks:
//   - Ensures that at most one "backcompat" asset is configured; returns an error if more than one is provided.
func LoadAssets(assets []string, stdout, stderr io.Writer) (LoadedAssets, error) {
	assetTypeToAssetsMap, err := createAssetTypeToAssetsMap(assets)
	if err != nil {
		return LoadedAssets{}, err
	}

	var conjureBackCompat backcompatasset.BackCompatChecker
	backcompatAssets := assetTypeToAssetsMap[assetapi.ConjureBackcompat]
	switch len(backcompatAssets) {
	case 0:
		// Do nothing
	case 1:
		conjureBackCompat = backcompatasset.New(backcompatAssets[0], stdout, stderr)
	default:
		return LoadedAssets{}, pkgerrors.Errorf(`only 0 or 1 "backcompat" can be configured, detected %d: %v`, len(backcompatAssets), backcompatAssets)
	}

	return LoadedAssets{
		ConjureBackcompat:            conjureBackCompat,
		ConjureIRExtensionsProviders: assetTypeToAssetsMap[assetapiinternal.ConjureIRExtensionsProvider],
	}, nil
}

// createAssetTypeToAssetsMap takes a slice of asset paths, determines their types using the getAssetTypeForAsset
// function, and returns a map from AssetType to the list of assets of that type. Returns an error if any asset cannot
// be executed to determine its type, or if any asset is of an unsupported type.
//
// Note that this function does not perform any semantic validation of the assets: if the asset reports its type in a
// supported manner, it is considered valid and included in the returned map. Any constraints (such as only allowing one
// asset of a given type etc.) must be enforced by the caller.
func createAssetTypeToAssetsMap(assets []string) (map[assetapi.AssetType][]string, error) {
	validAssetTypes := assetapiinternal.AllAssetsTypes()

	assetMap := make(map[assetapi.AssetType][]string)
	for _, asset := range assets {
		assetType, err := getAssetTypeForAsset(asset)
		if err != nil {
			return nil, pkgerrors.Wrapf(err, "failed to get asset type for asset %s", asset)
		}

		if !slices.Contains(validAssetTypes, assetType) {
			return nil, pkgerrors.Errorf("asset %s has unrecognized type %s: supported asset types are %v", asset, assetType, validAssetTypes)
		}
		assetMap[assetType] = append(assetMap[assetType], asset)
	}
	return assetMap, nil
}

// getAssetTypeForAsset returns the AssetType for the provided asset.
//
// This function supports determining the asset type in 2 different ways:
//  1. Invoking the asset with the "conjure-plugin-asset-type" command and parsing the output as a JSON string
//  2. Invoking the asset with the "_assetInfo" command and parsing the output as JSON to get the "type" field
//
// The function will attempt to use both methods in order and will return the result of the first one that succeeds.
// Returns an error if the asset cannot be executed or if neither method succeeds in determining the asset type.
func getAssetTypeForAsset(asset string) (assetapi.AssetType, error) {
	// if asset type can be determined using the new command, return it
	assetType, assetTypeErr := cmdutils.GetCommandOutputAsJSON[string](exec.Command(asset, assetapiinternal.AssetTypeCommand))
	if assetTypeErr == nil {
		return assetapi.AssetType(assetType), nil
	}

	legacyAssetInfo, legacyAssetInfoErr := cmdutils.GetCommandOutputAsJSON[assetInfoResponse](exec.Command(asset, "_assetInfo"))
	if legacyAssetInfoErr == nil {
		if legacyAssetInfo.Type == nil {
			return "", pkgerrors.Errorf("asset information response did not contain type field, which is required for legacy _assetInfo command")
		}
		return assetapi.AssetType(*legacyAssetInfo.Type), nil
	}

	return "", pkgerrors.Wrapf(errors.Join(assetTypeErr, legacyAssetInfoErr), "failed to determine asset type for asset %s", asset)
}

type assetInfoResponse struct {
	Type *string `json:"type"`
}
