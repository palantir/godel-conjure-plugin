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
	"path/filepath"
	"strings"

	"github.com/palantir/conjure-go/v7/conjure"
	conjurego "github.com/palantir/conjure-go/v7/conjure"
	"github.com/palantir/conjure-go/v7/conjure-api/conjure/spec"
	"github.com/palantir/pkg/codegenfiles"
	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
)

const indentLen = 2

// generatedFileMatcher matches the files Conjure owns within an output directory. Ownership is
// established by the ".conjure." infix rather than by a marker inside the file, because a
// .conjure.json output cannot carry a comment; a Project therefore leaves ContentsMatcher unset and
// relies on this instead.
var generatedFileMatcher = matcher.Name(`.*\.conjure\.(go|json)$`)

func Run(params ConjureProjectParams, verify bool, projectDir string, stdout io.Writer) error {
	type verifyFailedInfo struct {
		name       string
		diffOutput string
	}
	var verifyFailedInfos []verifyFailedInfo

	for _, currParam := range params {
		conjureDef, err := conjureDefinitionFromParam(currParam)
		if err != nil {
			return err
		}

		outputConf := conjure.OutputConfiguration{
			OutputDir:                filepath.Join(projectDir, currParam.OutputDir),
			GenerateServer:           currParam.Server,
			GenerateCLI:              currParam.CLI,
			GenerateFuncsVisitor:     currParam.AcceptFuncs,
			ExportErrorDecoder:       currParam.ExportErrorDecoder,
			ErrorParameterFormatJSON: currParam.ErrorParameterFormatJSON,
		}

		out, err := renderOutput(conjureDef, outputConf)
		if err != nil {
			return err
		}

		// Each Conjure project owns its own output directory and decides for itself whether obsolete
		// output is removed, so each is reconciled as a separate project. Dir is the project directory
		// rather than the output directory so that reported paths are the ones a developer recognizes;
		// the matcher, not Dir, is what confines the project to its own output.
		p := &codegenfiles.Project{
			Dir:         projectDir,
			FileMatcher: matcher.All(matcher.Path(currParam.OutputDir), generatedFileMatcher),
			DeleteStale: !currParam.SkipDeleteGeneratedFiles,
		}
		changes, err := p.Plan(out)
		if err != nil {
			return errors.Wrapf(err, "failed to reconcile Conjure output for %s", currParam.ProjectName)
		}

		if verify {
			if !changes.Empty() {
				verifyFailedInfos = append(verifyFailedInfos, verifyFailedInfo{
					name:       currParam.ProjectName,
					diffOutput: changes.String(),
				})
			}
			continue
		}
		if err := changes.Apply(); err != nil {
			return errors.Wrapf(err, "failed to write Conjure output for %s", currParam.ProjectName)
		}
	}

	if verify && len(verifyFailedInfos) > 0 {
		_, _ = fmt.Fprintf(stdout, "Conjure output differs from what currently exists for %d project(s)\n", len(verifyFailedInfos))
		for _, currVerifyFailedInfo := range verifyFailedInfos {
			_, _ = fmt.Fprintf(stdout, "%s%s:\n", strings.Repeat(" ", indentLen), currVerifyFailedInfo.name)
			for currDiffOutputLine := range strings.SplitSeq(currVerifyFailedInfo.diffOutput, "\n") {
				_, _ = fmt.Fprintf(stdout, "%s%s\n", strings.Repeat(" ", indentLen*2), currDiffOutputLine)
			}
		}
		return fmt.Errorf("conjure verify failed")
	}
	return nil
}

// renderOutput renders every file the definition produces into a single Output. Rendering is what
// conjure.Generate does before writing, so the content is identical; collecting it up front is what
// allows the output directory to be reconciled rather than overwritten in place.
func renderOutput(conjureDef spec.ConjureDefinition, outputConf conjure.OutputConfiguration) (*codegenfiles.Output, error) {
	files, err := conjure.GenerateOutputFiles(conjureDef, outputConf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate conjure output")
	}
	out := codegenfiles.NewOutput()
	for _, file := range files {
		rendered, err := file.Render()
		if err != nil {
			return nil, err
		}
		// Conjure reports absolute paths; Plan resolves them against the output directory.
		out.Add(file.AbsPath(), rendered)
	}
	return out, nil
}

func conjureDefinitionFromParam(param ConjureProjectParam) (spec.ConjureDefinition, error) {
	bytes, err := param.IRProvider.IRBytes()
	if err != nil {
		return spec.ConjureDefinition{}, err
	}
	conjureDefinition, err := conjurego.FromIRBytes(bytes)
	if err != nil {
		return spec.ConjureDefinition{}, err
	}
	return conjureDefinition, nil
}
