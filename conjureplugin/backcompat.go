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
	"github.com/palantir/godel-conjure-plugin/v5/backcompat-cli-bundler/conjurebackcompatcli"
	"github.com/pkg/errors"
	"io"
)

func BackCompat(params ConjureProjectParams, projectDir, groupFlagVal, repositoryNameFlagVal, artifactoryUrlFlagVal string, stdout io.Writer) error {
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

		if isCompatible, out, err := conjurebackcompatcli.CheckBackcompat(currentIRBytes, groupFlagVal, key, projectDir); err != nil {
			return err
		} else if !isCompatible {
			return errors.New(fmt.Sprintf("check backcompat failed\n%s", out))
		}
		k++
	}

	return nil
}
