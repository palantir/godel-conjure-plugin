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

package conjureplugin

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path"

	"github.com/palantir/distgo/distgo"
	gitversioner "github.com/palantir/distgo/projectversioner/git"
	"github.com/palantir/distgo/publisher/artifactory"
	extensionsprovider "github.com/palantir/godel-conjure-plugin/v6/internal/extensions-provider"
	"github.com/palantir/pkg/safejson"
	"github.com/pkg/errors"
)

func Publish(params ConjureProjectParams, projectDir string, flagVals map[distgo.PublisherFlagName]interface{},
	dryRun bool, stdout io.Writer, extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string) error {
	var paramsToPublishKeys []string
	var paramsToPublish []ConjureProjectParam
	for i, param := range params.OrderedParams() {
		if !param.Publish {
			continue
		}
		paramsToPublishKeys = append(paramsToPublishKeys, params.SortedKeys[i])
		paramsToPublish = append(paramsToPublish, param)
	}
	// nothing to publish
	if len(paramsToPublish) == 0 {
		return nil
	}

	// publishing at least 1 artifact: determine version. Note that this is currently hard-coded to use the Git
	// project versioner.
	versioner := gitversioner.New()
	version, err := versioner.ProjectVersion(projectDir)
	if err != nil {
		return err
	}

	publisher := artifactory.NewArtifactoryPublisher()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	for i, param := range paramsToPublish {
		key := paramsToPublishKeys[i]
		currDir := path.Join(tmpDir, fmt.Sprintf("conjure-%s", key))
		irFileName := fmt.Sprintf("%s-%s.conjure.json", key, version)
		keyAsDistID := distgo.DistID(key)
		if err := os.Mkdir(currDir, 0755); err != nil {
			return errors.WithStack(err)
		}

		var groupID string
		if param.GroupID != "" {
			groupID = param.GroupID
		}
		if cliGroupID != "" {
			groupID = cliGroupID
		}

		projectInfo := distgo.ProjectInfo{
			ProjectDir: currDir,
			Version:    version,
		}
		productOutputInfo := distgo.ProductOutputInfo{
			ID:   distgo.ProductID(key),
			Name: key,
			DistOutputInfos: &distgo.DistOutputInfos{
				DistIDs: []distgo.DistID{keyAsDistID},
				DistInfos: map[distgo.DistID]distgo.DistOutputInfo{
					keyAsDistID: {
						DistNameTemplateRendered: irFileName,
						DistArtifactNames: []string{
							irFileName,
						},
						PackagingExtension: "json",
					},
				},
			},
			PublishOutputInfo: &distgo.PublishOutputInfo{
				GroupID: groupID,
			},
		}

		// Use distgo to generate the path of the file we are going to publish
		directoryPath := distgo.ProductDistOutputDir(projectInfo, productOutputInfo, keyAsDistID)
		if err := os.MkdirAll(directoryPath, 0755); err != nil {
			return errors.WithStack(err)
		}

		irBytes, err := param.IRProvider.IRBytes()
		if err != nil {
			return err
		}

		irBytes, err = AddExtensionsToIrBytes(irBytes, extensionsProvider, groupID, key, version)
		if err != nil {
			return errors.WithStack(err)
		}

		irFilePath := path.Join(directoryPath, irFileName)
		if err := os.WriteFile(irFilePath, irBytes, 0644); err != nil {
			return errors.WithStack(err)
		}

		if err := publisher.RunPublish(distgo.ProductTaskOutputInfo{
			Project: projectInfo,
			Product: productOutputInfo,
		}, nil, flagVals, dryRun, stdout); err != nil {
			return err
		}
	}
	return nil
}

// AddExtensionsToIrBytes takes a Conjure IR JSON byte slice, parses it, and updates its "extensions" block
// by merging in additional extensions provided by the given ExtensionsProvider. The function then marshals
// and returns the updated IR as a byte slice.
//
// Parameters:
//   - irBytes: The original Conjure IR as a JSON-encoded byte slice.
//   - extensionsProvider: A function that, given the IR, project name, and version, returns additional extensions.
//   - conjureProject: The name of the current Conjure project.
//   - version: The version string of the current Conjure project.
//
// Returns:
//   - []byte: The updated Conjure IR as a JSON-encoded byte slice.
//   - error: Any error encountered during unmarshalling, extension provision, or marshalling.
//
// An error is returned if the input is not valid JSON, if the "extensions" block is missing or malformed,
// or if the extensionsProvider fails.
func AddExtensionsToIrBytes(
	irBytes []byte,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	groupID, conjureProject, version string,
) ([]byte, error) {
	var conjureCliIr map[string]any
	if err := safejson.Unmarshal(irBytes, &conjureCliIr); err != nil {
		return nil, err
	}

	extensionsAccumulator := make(map[string]any)

	// if there is a valid `extensions` block in irBytes...
	if conjureCliIrExtensions, ok := conjureCliIr["extensions"]; ok {
		conjureCliIrExtensionsObject, ok := conjureCliIrExtensions.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("conjure CLI generated Conjure IR with an extensions block that was not a json object violating the Conjure spec; see https://github.com/palantir/conjure/blob/master/docs/spec/intermediate_representation.md#extensions for more details")
		}

		// ...add it
		maps.Copy(extensionsAccumulator, conjureCliIrExtensionsObject)
	}

	providedExtensions, err := extensionsProvider(irBytes, groupID, conjureProject, version)
	if err != nil {
		return nil, err
	}

	maps.Copy(extensionsAccumulator, providedExtensions)

	if len(extensionsAccumulator) > 0 {
		conjureCliIr["extensions"] = extensionsAccumulator
	}

	return safejson.MarshalIndent(conjureCliIr, "", "\t")
}

func PublisherFlags() ([]distgo.PublisherFlag, error) {
	return artifactory.NewArtifactoryPublisher().Flags()
}
