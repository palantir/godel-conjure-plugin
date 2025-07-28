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
	"github.com/pkg/errors"
)

type ExtensionsProvider func(irBytes []byte, conjureProject, version string) (map[string]any, error)

// New returns an ExtensionsProvider that, when invoked, collects and merges
// extension data from a set of external assets. Each asset is queried to determine if it is a
// "conjure-ir-extensions-provider" and, if so, is executed with extensionsAssetArgs; arguments
// describing the current IR, configuration, and project metadata.
// The extensions returned by each asset are combined into a single map.
//
// Parameters:
//   - config:      The configuration string to be passed to each asset.
//   - assets:      A list of asset executable paths to be queried for extensions.
//   - url:         The Conjure IR URL associated with the current project.
//   - groupID:     The group ID of the current project.
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
func New(configFile string, assets []string, url, repo, groupID string) ExtensionsProvider {
	return func(irBytes []byte, conjureProject, version string) (map[string]any, error) {
		irFile, err := tempfilecreator.WriteBytesToTempFile(irBytes)
		if err != nil {
			return nil, err
		}

		allExtensions := make(map[string]any)
		for _, asset := range assets {
			cmd := exec.Command(asset, "_assetInfo")
			assetInfoOutput, err := cmd.Output()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to execute %v\nOutput:\n%s", cmd.Args, string(assetInfoOutput))
			}

			var response assetInfoResponse
			if err := safejson.Unmarshal(assetInfoOutput, &response); err != nil {
				return nil, err
			}

			if response.Type == nil {
				return nil, fmt.Errorf("invalid response from calling %v; wanted a JSON object with a `type` key; but got:\n%v", cmd.Args, string(assetInfoOutput))
			}

			if *response.Type != "conjure-ir-extensions-provider" {
				continue
			}

			arg, err := safejson.Marshal(extensionsAssetArgs{
				PluginConfigFile: &configFile,
				CurrentIRFile:    &irFile,
				URL:              &url,
				Repo:             &repo,
				GroupID:          &groupID,
				ProjectName:      &conjureProject,
				Version:          &version,
			})
			if err != nil {
				return nil, err
			}

			additionExtensionsBytes, err := exec.Command(asset, string(arg)).Output()
			if err != nil {
				return nil, err
			}

			var additionalExtensions map[string]any
			if err := safejson.Unmarshal(additionExtensionsBytes, &additionalExtensions); err != nil {
				return nil, err
			}

			maps.Copy(allExtensions, additionalExtensions)
		}

		return allExtensions, nil
	}
}

type extensionsAssetArgs struct {
	PluginConfigFile *string `json:"config,omitempty"`
	CurrentIRFile    *string `json:"current-ir-file,omitempty"`
	URL              *string `json:"url,omitempty"`
	Repo             *string `json:"repo,omitempty"`
	GroupID          *string `json:"group-id,omitempty"`
	ProjectName      *string `json:"project-name,omitempty"`
	Version          *string `json:"version,omitempty"`
}

type assetInfoResponse struct {
	Type *string `json:"type"`
}
