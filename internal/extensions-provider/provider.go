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
	"maps"
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/palantir/pkg/safejson"
)

type ExtensionsProvider func(irBytes []byte, conjureProject, version string) (map[string]any, error)

func NewExtensionsProvider(config string, assets []string, url, groupID string) ExtensionsProvider {
	return func(irBytes []byte, conjureProject, version string) (map[string]any, error) {
		irFile, err := tempfilecreator.WriteBytesToTempFile(irBytes)
		if err != nil {
			return nil, err
		}

		allExtensions := make(map[string]any)
		for _, asset := range assets {
			bytes, err := exec.Command(asset, "_assetInfo").Output()
			if err != nil {
				return nil, err
			}

			var response assetTypeResponse
			if err := safejson.Unmarshal(bytes, &response); err != nil {
				return nil, err
			}

			if response.Type != "conjure-ir-extensions-provider" {
				continue
			}

			arg, err := safejson.MarshalIndent(extensionsAssetArgs{
				Config:         config,
				ProposedIRFile: irFile,
				URL:            url,
				GroupID:        groupID,
				Project:        conjureProject,
				Version:        version,
			}, "", "\t")
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
	Config         string `json:"config,omitempty"`
	ProposedIRFile string `json:"proposed,omitempty"`
	URL            string `json:"url,omitempty"`
	GroupID        string `json:"group-id,omitempty"`
	Project        string `json:"project,omitempty"`
	Version        string `json:"version,omitempty"`
}

type assetTypeResponse struct {
	Type string `json:"type"`
}
