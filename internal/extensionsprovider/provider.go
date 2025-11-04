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

package extensionsprovider

import (
	"fmt"
	"maps"
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/palantir/pkg/safejson"
)

type ExtensionsProvider func(irBytes []byte, groupID, conjureProject, version string) (map[string]any, error)

// NewAssetsExtensionsProvider returns an ExtensionsProvider that, when invoked, collects and merges
// extension data from the provided extensionsProviderAssets assets.
//
// Parameters:
//   - extensionsProviderAssets: Paths to the extensions provider assets. Must be known to be valid ExtensionsProvider assets.
//   - configFile:               The path to the godel-conjure-plugin YAML configuration file.
//   - url:                      The base URL provided to the Conjure "publish" operation.
//   - groupID:                  The group ID of the current project.
//
// Returns:
//   - ExtensionsProvider: A function that, given the IR bytes, project name, and version, returns a merged
//     map of all extensions provided by the external assets, or an error if any step fails.
//
// The returned ExtensionsProvider will:
//   - Write the provided IR bytes to a temporary file for use by assets.
//   - For each asset, check if it is an extensions provider by invoking it with the "_assetInfo" argument.
//   - If the asset is a provider, invoke it with the relevant arguments to retrieve extension data.
//   - Merge all extension maps from the assets into a single result.
//   - Return the combined extensions map or an error if any asset invocation or JSON parsing fails.
func NewAssetsExtensionsProvider(extensionsProviderAssets []string, configFile string, url string) ExtensionsProvider {
	return func(irBytes []byte, groupID, conjureProject, version string) (map[string]any, error) {
		// return nil if there are no assets
		if len(extensionsProviderAssets) == 0 {
			return nil, nil
		}

		irFile, err := tempfilecreator.WriteBytesToTempFile(irBytes)
		if err != nil {
			return nil, err
		}

		allExtensions := make(map[string]any)
		for _, asset := range extensionsProviderAssets {
			additionalExtensions, err := getExtensionsFromExtensionProviderAsset(asset, extensionsAssetArgs{
				PluginConfigFile: &configFile,
				CurrentIRFile:    &irFile,
				URL:              &url,
				GroupID:          &groupID,
				ProjectName:      &conjureProject,
				Version:          &version,
			})
			if err != nil {
				return nil, err
			}
			maps.Copy(allExtensions, additionalExtensions)
		}

		return allExtensions, nil
	}
}

// getExtensionsFromExtensionProviderAsset returns the extensions provided by the given extensions provider asset given
// the provided extensionsAssetArgs argument. Invokes the asset with the JSON representation of the provided arg as the
// argument and returns the result of unmarshalling the stdout output as JSON. Returns an error if there are any errors
// with invoking the asset or if the stdout output cannot be parsed as JSON.
func getExtensionsFromExtensionProviderAsset(asset string, arg extensionsAssetArgs) (map[string]any, error) {
	argJSONBytes, err := safejson.Marshal(arg)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(asset, string(argJSONBytes))
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%w: failed to execute %v\nstdout:\n%s\nstderr:\n%s", err, cmd.Args, string(stdout), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("%w: failed to execute %v\nstdout:\n%s", err, cmd.Args, string(stdout))
	}

	var additionalExtensions map[string]any
	if err := safejson.Unmarshal(stdout, &additionalExtensions); err != nil {
		return nil, err
	}
	return additionalExtensions, nil
}

type extensionsAssetArgs struct {
	PluginConfigFile *string `json:"config,omitempty"`
	CurrentIRFile    *string `json:"current-ir-file,omitempty"`
	URL              *string `json:"url,omitempty"`
	GroupID          *string `json:"group-id,omitempty"`
	ProjectName      *string `json:"project-name,omitempty"`
	Version          *string `json:"version,omitempty"`
}
