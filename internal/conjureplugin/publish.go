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
	"path/filepath"

	"github.com/palantir/distgo/distgo"
	"github.com/palantir/distgo/publisher"
	"github.com/palantir/distgo/publisher/artifactory"
	"github.com/pkg/errors"
)

func Publish(param PublishParam, flagVals map[distgo.PublisherFlagName]any, dryRun bool, stdout io.Writer) error {
	if len(param.ConjureIR) == 0 {
		return nil
	}
	// distgo gives GroupIDFlag precedence over each product's PublishOutputInfo.GroupID but PublishParam has already
	// resolved the CLI override into each project, so remove the flag to ensure the resolved value is used.
	publishFlagVals := maps.Clone(flagVals)
	delete(publishFlagVals, publisher.GroupIDFlag.Name)
	return publishConjureIR(param.ConjureIR, param.Version, publishFlagVals, dryRun, stdout)
}

func publishConjureIR(
	params []ConjureIRPublishParam,
	version string,
	flagVals map[distgo.PublisherFlagName]any,
	dryRun bool,
	stdout io.Writer,
) error {
	artifactoryPublisher := artifactory.NewArtifactoryPublisher()
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	var publishInputs []distgo.ProductPublishInfo
	for _, param := range params {
		conjureProjectName := param.ProjectName
		currDir := filepath.Join(tmpDir, fmt.Sprintf("conjure-%s", conjureProjectName))
		irFileName := fmt.Sprintf("%s-%s.conjure.json", conjureProjectName, version)
		keyAsDistID := distgo.DistID(conjureProjectName)
		if err := os.Mkdir(currDir, 0755); err != nil {
			return errors.WithStack(err)
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
				GroupID: param.GroupID,
			},
		}

		// Use distgo to generate the path of the file we are going to publish
		directoryPath := distgo.ProductDistOutputDir(projectInfo, productOutputInfo, keyAsDistID)
		if err := os.MkdirAll(directoryPath, 0755); err != nil {
			return errors.WithStack(err)
		}

		irFilePath := filepath.Join(directoryPath, irFileName)
		if err := os.WriteFile(irFilePath, []byte(param.IR), 0644); err != nil {
			return errors.WithStack(err)
		}

		publishInputs = append(publishInputs, distgo.ProductPublishInfo{
			ProductTaskOutputInfo: distgo.ProductTaskOutputInfo{
				Project: projectInfo,
				Product: productOutputInfo,
			},
		})
	}

	if err := artifactoryPublisher.RunPublish(publishInputs, flagVals, dryRun, stdout); err != nil {
		return err
	}
	return nil
}

func PublisherFlags() ([]distgo.PublisherFlag, error) {
	return artifactory.NewArtifactoryPublisher().Flags()
}
