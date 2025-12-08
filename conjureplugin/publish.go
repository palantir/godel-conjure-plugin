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
	"os"
	"path/filepath"

	"github.com/palantir/distgo/distgo"
	gitversioner "github.com/palantir/distgo/projectversioner/git"
	"github.com/palantir/distgo/publisher/artifactory"
	"github.com/palantir/godel-conjure-plugin/v6/internal/extensionsprovider"
	"github.com/pkg/errors"
)

func Publish(params ConjureProjectParams, projectDir string, flagVals map[distgo.PublisherFlagName]interface{},
	dryRun bool, stdout io.Writer, extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string) error {
	var paramsToPublish []ConjureProjectParam
	for _, param := range params {
		if !param.Publish {
			continue
		}
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

	for _, param := range paramsToPublish {
		conjureProjectName := param.ProjectName
		currDir := filepath.Join(tmpDir, fmt.Sprintf("conjure-%s", conjureProjectName))
		irFileName := fmt.Sprintf("%s-%s.conjure.json", conjureProjectName, version)
		keyAsDistID := distgo.DistID(conjureProjectName)
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
			ID:   distgo.ProductID(conjureProjectName),
			Name: conjureProjectName,
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

		irBytes, err = addExtensionsToIRBytes(irBytes, extensionsProvider, groupID, conjureProjectName, version)
		if err != nil {
			return errors.WithStack(err)
		}

		irFilePath := filepath.Join(directoryPath, irFileName)
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

func PublisherFlags() ([]distgo.PublisherFlag, error) {
	return artifactory.NewArtifactoryPublisher().Flags()
}
