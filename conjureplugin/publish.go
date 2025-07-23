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
	"os/exec"
	"path"

	"github.com/palantir/distgo/distgo"
	gitversioner "github.com/palantir/distgo/projectversioner/git"
	"github.com/palantir/distgo/publisher/artifactory"
	"github.com/palantir/pkg/safejson"
	"github.com/pkg/errors"
)

func Publish(params ConjureProjectParams, projectDir string, flagVals map[distgo.PublisherFlagName]interface{}, dryRun bool, stdout io.Writer, groupId string, url string, assets ...string) error {
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
		tmpIRFileName := fmt.Sprintf("%s-%s.conjure.without-extensions.json", key, version)
		keyAsDistID := distgo.DistID(key)
		if err := os.Mkdir(currDir, 0755); err != nil {
			return errors.WithStack(err)
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
				// TODO: allow this to be specified in config?
				GroupID: "",
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

		tmpIRFilePath := path.Join(directoryPath, tmpIRFileName)
		if err := os.WriteFile(tmpIRFileName, irBytes, 0644); err != nil {
			panic(errors.WithStack(err))
		}

		// url + "/artifactory/" + groupId + "/" + key is what is needed for resolvinng the older conjure IRs

		type assetArgs struct {
			Proposed string `json:"proposed"` // proposed IR (copying nameing from conjure backcompat)
			Version  string `json:"version"`  // take this version if you incompatible
			Url      string `json:"url"`
			GroupId  string `json:"group-id"`
			Project  string `json:"project"`
		}

		// discover assets that return the ir-plugin info thing
		var extensionsAssets []string
		for _, asset := range assets {
			var response struct {
				Type string `json:"type"`
			}
			bytes, err := exec.Command(asset, "_assetInfo").Output()
			if err != nil {
				panic(errors.WithStack(err))
			}

			if err := safejson.Unmarshal(bytes, &response); err != nil {
				panic(errors.WithStack(err))
			}

			if response.Type == "conjure-ir-extensions-provider" {
				extensionsAssets = append(extensionsAssets, asset)
			}
		}

		combinedExtensions := make(map[string]any)
		for _, asset := range extensionsAssets {
			arg, err := safejson.Marshal(assetArgs{
				Proposed: tmpIRFilePath,
				Version:  version,
				Url:      url,
				GroupId:  groupId,
				Project:  key,
			})
			if err != nil {
				panic(errors.WithStack(err))
			}

			extensionBytes, err := exec.Command(asset, string(arg)).Output()
			// os.Stderr.Write([]byte("saying hi"))
			// os.Stderr.Write(extensionBytes)
			if err != nil {
				panic(errors.WithStack(err))
			}
			// os.Stderr.Write([]byte("hello world"))

			var extensions map[string]any // must be this way for merging purposes
			if err := safejson.Unmarshal(extensionBytes, &extensions); err != nil {
				panic(errors.WithStack(err))
			}
			for k, v := range extensions {
				combinedExtensions[k] = v
			}

		}

		var theRest map[string]any
		if err := safejson.Unmarshal(irBytes, &theRest); err != nil {
			panic(errors.WithStack(err))
		}

		theRest["extensions"] = combinedExtensions // maybe defend against extensions already present

		irBytesWithExtensions, err := safejson.Marshal(theRest)
		if err != nil {
			panic(errors.WithStack(err))
		}

		_, err = os.Stderr.Write(irBytesWithExtensions)
		if err != nil {
			panic(errors.WithStack(err))
		}

		// send tmpIR file path to the asset, along with params to get the prior ir, that will print to stoud the extensions
		irFilePath := path.Join(directoryPath, irFileName)
		if err := os.WriteFile(irFilePath, irBytesWithExtensions, 0644); err != nil {
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
