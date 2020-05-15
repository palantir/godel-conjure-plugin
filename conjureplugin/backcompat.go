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
	"github.com/palantir/distgo/distgo"
	"github.com/palantir/distgo/publisher"
	"github.com/palantir/godel-conjure-plugin/v5/backcompat-cli-bundler/conjurebackcompatcli"
	"github.com/pkg/errors"
	"io"
	"os/exec"
)

func BackCompat(params ConjureProjectParams, projectDir string, flagVals map[distgo.PublisherFlagName]interface{}, stdout io.Writer) error {
	k := 0
	for key, currParam := range params.Params {
		currentIRBytes, err := currParam.IRProvider.IRBytes()
		if err != nil {
			return err
		}

		//previousVersion, err := getLastTag(projectDir)
		//if err != nil {
		//    return err
		//}
		//irFileName := fmt.Sprintf("%s-%s.conjure.json", key, previousVersion)
		//keyAsDistID := distgo.DistID(key)
		//projectInfo := distgo.ProjectInfo{
		//    ProjectDir: "",
		//    Version:    previousVersion,
		//}
		//productOutputInfo := distgo.ProductOutputInfo{
		//    ID: distgo.ProductID(key),
		//    DistOutputInfos: &distgo.DistOutputInfos{
		//        DistIDs: []distgo.DistID{keyAsDistID},
		//        DistInfos: map[distgo.DistID]distgo.DistOutputInfo{
		//            keyAsDistID: {
		//                DistNameTemplateRendered: irFileName,
		//                DistArtifactNames: []string{
		//                    irFileName,
		//                },
		//                PackagingExtension: "json",
		//            },
		//        },
		//    },
		//    PublishOutputInfo: &distgo.PublishOutputInfo{
		//        // TODO: allow this to be specified in config?
		//        GroupID: "",
		//    },
		//}
		//directoryPath := distgo.ProductDistOutputDir(projectInfo, productOutputInfo, keyAsDistID)
		//irFilePath := path.Join(directoryPath, irFileName)

		//outputDir := currParam.OutputDir

		//outputConf := conjure.OutputConfiguration{OutputDir: path.Join(projectDir, outputDir), GenerateServer: currParam.Server}
		groupID, ok := flagVals[publisher.GroupIDFlag.Name]
		if !ok {
			return errors.New(fmt.Sprintf("%s flag is not specified", publisher.GroupIDFlag.Name))
		}

		if isCompatible, out, err := conjurebackcompatcli.CheckBackcompat(currentIRBytes, groupID.(string), key, projectDir); err != nil {
			return err
		} else if !isCompatible {
			return errors.New(fmt.Sprintf("check backcompat failed\n%s", out))
		}
		k++
	}

	return nil
}

func getLastTag(projectDir string) (string, error) {
	cmd := exec.Command("git", "describe", "--abbrev=0 --tags")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get last tag %v\nOutput:\n%s", cmd.Args, string(output))
	}
	return string(output), nil
}
