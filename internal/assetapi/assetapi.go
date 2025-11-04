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
	"fmt"
	"os/exec"
	"slices"

	"github.com/palantir/pkg/safejson"
	"github.com/pkg/errors"
)

type AssetType string

const (
	Backcompat                  AssetType = "backcompat"
	ConjureIRExtensionsProvider AssetType = "conjure-ir-extensions-provider"
)

type LoadedAssets struct {
	BackCompatAssets                  []string
	ConjureIRExtensionsProviderAssets []string
}

// LoadAssets takes a list of asset paths and returns a LoadedAssets struct that contains the typed assets. Returns an
// error if any of the provided assets are not valid according to the plugin asset specification.
func LoadAssets(assets []string) (LoadedAssets, error) {
	loadedAssets, err := loadAssets(assets)
	if err != nil {
		return LoadedAssets{}, err
	}
	return LoadedAssets{
		BackCompatAssets:                  loadedAssets[Backcompat],
		ConjureIRExtensionsProviderAssets: loadedAssets[ConjureIRExtensionsProvider],
	}, nil
}

// loadAssets takes a list of asset paths, determines their types, and returns a map from AssetType to the list of
// assets of that type. Returns an error if any asset cannot be executed to determine its type, or if any asset is of an
// unsupported type.
func loadAssets(assets []string) (map[AssetType][]string, error) {
	assetMap := make(map[AssetType][]string)
	for _, asset := range assets {
		assetType, err := getAssetTypeForAsset(asset)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get asset type for asset %s", asset)
		}

		assetTypes := []AssetType{
			Backcompat,
			ConjureIRExtensionsProvider,
		}

		if !slices.Contains(assetTypes, assetType) {
			return nil, fmt.Errorf("unsupported asset type %s for asset %s: the only asset types that are supported are %v", assetType, asset, assetTypes)
		}
		assetMap[assetType] = append(assetMap[assetType], asset)
	}
	return assetMap, nil
}

// getAssetTypeForAsset returns the AssetType for the provided asset by executing the asset with the _assetInfo command,
// parsing the output printed to stdout as JSON, and returning the value of the "type" field. Returns an error if the
// asset cannot be executed, if the response cannot be parsed as JSON, or if the "type" field is missing.
func getAssetTypeForAsset(asset string) (AssetType, error) {
	cmd := exec.Command(asset, "_assetInfo")
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", errors.Wrapf(err, "failed to execute %v\nstdout:\n%s\nstderr:\n%s", cmd.Args, string(stdout), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("%w: failed to execute %v\nstdout:\n%s", err, cmd.Args, string(stdout))
	}

	var response assetInfoResponse
	if err := safejson.Unmarshal(stdout, &response); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal asset info")
	}

	if response.Type == nil {
		return "", fmt.Errorf("invalid response from calling %v; wanted a JSON object with a `type` key; but got:\n%v", cmd.Args, string(stdout))
	}

	return AssetType(*response.Type), nil
}

type assetInfoResponse struct {
	Type *string `json:"type"`
}
