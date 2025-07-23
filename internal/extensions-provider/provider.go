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
	"errors"
	"maps"
	"os"
	"os/exec"

	"github.com/palantir/pkg/safejson"
)

type ExtensionsProvider func(irBytesWithoutExtensions []byte, conjureProject string, version string) (map[string]any, error)

func NewExtensionsProvider(url string, groupID string, assets []string) ExtensionsProvider {
	return func(irBytesWithoutExtensions []byte, conjureProject string, version string) (_ map[string]any, rErr error) {
		irFilePathWithoutExtensions, err := writeBytesToFile(irBytesWithoutExtensions)
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

			if response.Type != "conjure-ir-extensions-provider" { // skip assets that do not support `extensions`
				continue
			}

			arg, err := safejson.Marshal(extensionsAssetArgs{
				Proposed: irFilePathWithoutExtensions,
				Version:  version,
				URL:      url,
				GroupID:  groupID,
				Project:  conjureProject,
			})
			if err != nil {
				return nil, err
			}

			extensionBytes, err := exec.Command(asset, string(arg)).Output()
			if err != nil {
				return nil, err
			}

			var extensions map[string]any // must be this way for merging purposes
			if err := safejson.Unmarshal(extensionBytes, &extensions); err != nil {
				return nil, err
			}

			maps.Copy(allExtensions, extensions)
		}

		return allExtensions, nil
	}
}

func writeBytesToFile(bytes []byte) (_ string, rErr error) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer func() {
		rErr = errors.Join(rErr, file.Close())
	}()

	if _, rErr = file.Write(bytes); rErr != nil {
		return
	}

	return file.Name(), nil
}

type extensionsAssetArgs struct {
	Proposed string `json:"proposed,omitempty"` // proposed IR (copying naming from conjure backcompat)
	Version  string `json:"version,omitempty"`  // take this version if you incompatible
	URL      string `json:"url,omitempty"`
	GroupID  string `json:"group-id,omitempty"`
	Project  string `json:"project,omitempty"`
}

type assetTypeResponse struct {
	Type string `json:"type"`
}
