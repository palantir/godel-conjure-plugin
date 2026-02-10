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
	"strings"

	"github.com/palantir/conjure-go/v6/conjure"
	conjurego "github.com/palantir/conjure-go/v6/conjure"
	"github.com/palantir/conjure-go/v6/conjure-api/conjure/spec"
	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
)

const indentLen = 2

func Run(params ConjureProjectParams, verify bool, projectDir string, stdout io.Writer) error {
	type verifyFailedInfo struct {
		name       string
		diffOutput string
	}
	var verifyFailedInfos []verifyFailedInfo

	for _, currParam := range params {
		outputDir := currParam.OutputDir
		conjureDef, err := conjureDefinitionFromParam(currParam)
		if err != nil {
			return err
		}

		outputConf := conjure.OutputConfiguration{
			OutputDir:            filepath.Join(projectDir, outputDir),
			GenerateServer:       currParam.Server,
			GenerateCLI:          currParam.CLI,
			GenerateFuncsVisitor: currParam.AcceptFuncs,
			CGRModuleVersion:     currParam.CGRModuleVersion,
			WGSModuleVersion:     currParam.WGSModuleVersion,
			ExportErrorDecoder:   currParam.ExportErrorDecoder,
		}

		conjureFilesToGenerate, err := conjure.GenerateOutputFiles(conjureDef, outputConf)
		if err != nil {
			return errors.Wrapf(err, "failed to generate conjure output")
		}

		var filesToDelete []string
		if !currParam.SkipDeleteGeneratedFiles {
			filesToDelete, err = computeFilesToDelete(outputConf.OutputDir, conjureFilesToGenerate)
			if err != nil {
				return err
			}
		}

		if verify {
			// get files to checksum, which is combination of files that will be generated and files that will be deleted
			filesToChecksum := append(getOutputFileAbsPaths(conjureFilesToGenerate), filesToDelete...)
			diff, err := diffOnDisk(projectDir, filesToChecksum, conjureFilesToGenerate)
			if err != nil {
				return err
			}
			if len(diff.Diffs) > 0 {
				// set RootDir to empty so that output will be relative path
				diff.RootDir = ""
				verifyFailedInfos = append(verifyFailedInfos, verifyFailedInfo{
					name:       currParam.ProjectName,
					diffOutput: diff.String(),
				})
			}
		} else {
			// delete files marked for deletion
			for _, fileToDelete := range filesToDelete {
				if err := os.Remove(fileToDelete); err != nil {
					return errors.Wrapf(err, "failed to delete generated file %s", fileToDelete)
				}
			}
			if err := conjure.Generate(conjureDef, outputConf); err != nil {
				return err
			}
		}
	}

	if verify && len(verifyFailedInfos) > 0 {
		_, _ = fmt.Fprintf(stdout, "Conjure output differs from what currently exists for %d project(s)\n", len(verifyFailedInfos))
		for _, currVerifyFailedInfo := range verifyFailedInfos {
			_, _ = fmt.Fprintf(stdout, "%s%s:\n", strings.Repeat(" ", indentLen), currVerifyFailedInfo.name)
			for _, currDiffOutputLine := range strings.Split(currVerifyFailedInfo.diffOutput, "\n") {
				_, _ = fmt.Fprintf(stdout, "%s%s\n", strings.Repeat(" ", indentLen*2), currDiffOutputLine)
			}
		}
		return fmt.Errorf("conjure verify failed")
	}
	return nil
}

func getOutputFileAbsPaths(files []*conjure.OutputFile) []string {
	var out []string
	for _, currFile := range files {
		out = append(out, currFile.AbsPath())
	}
	return out
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

// computeFilesToDelete returns the absolute paths to Conjure-generated files within outputDir that are not contained
// within filesToGenerate.
//
// It does so by getting the absolute paths of all Conjure-generated files currently on disk in the output directory
// and removing all the absolute paths returned by the AbsPath function of the files in filesToGenerate.
func computeFilesToDelete(outputDir string, filesToGenerate []*conjure.OutputFile) ([]string, error) {
	existingFiles, err := getAllGeneratedFiles(outputDir)
	if err != nil {
		return nil, err
	}

	// Build a set of file paths that will be generated
	generatedPaths := make(map[string]struct{}, len(filesToGenerate))
	for _, file := range filesToGenerate {
		generatedPaths[file.AbsPath()] = struct{}{}
	}

	// Find files that exist but won't be regenerated
	var obsoleteFiles []string
	for _, existingFile := range existingFiles {
		if _, willBeGenerated := generatedPaths[existingFile]; !willBeGenerated {
			obsoleteFiles = append(obsoleteFiles, existingFile)
		}
	}

	return obsoleteFiles, nil
}

// getAllGeneratedFiles returns the absolute paths of all Conjure-generated files
// (files ending in .conjure.go or .conjure.json) within the specified output directory.
func getAllGeneratedFiles(outputDir string) ([]string, error) {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to stat output directory %s", outputDir)
	}

	// Match files ending in .conjure.go or .conjure.json
	include := matcher.Name(`.*\.conjure\.(go|json)$`)

	relPaths, err := matcher.ListFiles(outputDir, include, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list files in output directory %s", outputDir)
	}

	// Convert to absolute paths, filtering out directories
	// (matcher.ListFiles can return both files and directories that match)
	var absPaths []string
	for _, relPath := range relPaths {
		absPath := filepath.Join(outputDir, relPath)
		fileInfo, err := os.Stat(absPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to stat %s", absPath)
		}
		if !fileInfo.IsDir() {
			absPaths = append(absPaths, absPath)
		}
	}

	return absPaths, nil
}
