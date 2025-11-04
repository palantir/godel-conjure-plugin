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

package conjureplugin

import (
	"fmt"
	"maps"

	"github.com/palantir/godel-conjure-plugin/v6/internal/extensionsprovider"
	"github.com/palantir/pkg/safejson"
)

// addExtensionsToIRBytes takes a Conjure IR JSON byte slice, parses it, and updates its "extensions" block
// by merging in additional extensions provided by the given ExtensionsProvider.
//
// If the provided ExtensionsProvider returns no extensions, the original IR bytes are returned unmodified. Otherwise,
// the function returns the bytes that are the result of unmarshalling the provided IR as JSON, merging in the map
// entries provided by the ExtensionsProvider into the "extensions" block, marshalling the updated JSON back to bytes,
// and returning the marshalled bytes.
//
// Parameters:
//   - irBytes: The input Conjure IR as a JSON-encoded byte slice.
//   - extensionsProvider: A function that, given the IR, project name, and version, returns additional extensions.
//   - conjureProject: The name of the current Conjure project.
//   - version: The version string of the current Conjure project.
//
// Returns:
//   - []byte: The updated Conjure IR as a JSON-encoded byte slice (if the map returned by extensionsProvider does not
//     have any entries, the original irBytes are returned).
//   - error: Any error encountered during unmarshalling, extension provision, or marshalling.
//
// An error is returned if the input is not valid JSON, if the "extensions" block is missing or malformed,
// or if the extensionsProvider fails.
func addExtensionsToIRBytes(
	irBytes []byte,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	groupID, conjureProject, version string,
) ([]byte, error) {

	providedExtensions, err := extensionsProvider(irBytes, groupID, conjureProject, version)
	if err != nil {
		return nil, err
	}

	// if no extensions were provided, return the original IR bytes
	if len(providedExtensions) == 0 {
		return irBytes, nil
	}

	var irJSONMap map[string]any
	if err := safejson.Unmarshal(irBytes, &irJSONMap); err != nil {
		return nil, err
	}

	extensionsAccumulator := make(map[string]any)

	const extensionsMapName = "extensions"

	// add existing extensions from input IR if present
	if inputIRExtensions, ok := irJSONMap[extensionsMapName]; ok {
		inputIrExtensionsMap, ok := inputIRExtensions.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("the provided Conjure IR has an \"extensions\" field that is not a map, which is a violation of the Conjure spec; see https://github.com/palantir/conjure/blob/master/docs/spec/intermediate_representation.md#extensions for details")
		}
		maps.Copy(extensionsAccumulator, inputIrExtensionsMap)
	}

	// add extensions computed from provider
	maps.Copy(extensionsAccumulator, providedExtensions)

	if len(extensionsAccumulator) > 0 {
		irJSONMap[extensionsMapName] = extensionsAccumulator
	}

	return safejson.MarshalIndent(irJSONMap, "", "\t")
}
