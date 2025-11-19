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
	"github.com/pkg/errors"
)

const indentLen = 2

func Run(params ConjureProjectParams, verify bool, projectDir string, stdout io.Writer) error {
	var verifyFailedIndex []int
	verifyFailedErrors := make(map[int]string)
	verifyFailedFn := func(name int, errStr string) {
		verifyFailedIndex = append(verifyFailedIndex, name)
		verifyFailedErrors[name] = errStr
	}

	k := 0
	for _, currParam := range params.OrderedParams() {
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
		}

		files, err := conjure.GenerateOutputFiles(conjureDef, outputConf)
		if err != nil {
			return errors.Wrap(err, "failed to generate conjure output files")
		}

		var filesToDelete []string
		if !currParam.SkipDeleteGeneratedFiles {
			filesToDelete, err = computeObsoleteFiles(outputConf.OutputDir, files)
			if err != nil {
				return err
			}
		}

		if verify {
			diff, err := diffOnDisk(projectDir, files, outputConf.OutputDir, currParam.SkipDeleteGeneratedFiles)
			if err != nil {
				return err
			}

			if len(diff.Diffs) > 0 {
				verifyFailedFn(k, diff.String())
			}
		} else {
			for _, file := range filesToDelete {
				if err := os.Remove(file); err != nil {
					return errors.Wrapf(err, "failed to delete old generated files in %s", outputConf.OutputDir)
				}
			}
			for _, file := range files {
				if err := file.Write(); err != nil {
					return err
				}
			}
		}
		k++
	}

	if verify && len(verifyFailedIndex) > 0 {
		_, _ = fmt.Fprintf(stdout, "Conjure output differs from what currently exists: %v\n", verifyFailedIndex)
		for _, currKey := range verifyFailedIndex {
			_, _ = fmt.Fprintf(stdout, "%s%d:\n", strings.Repeat(" ", indentLen), currKey)
			for _, currErrLine := range strings.Split(verifyFailedErrors[currKey], "\n") {
				_, _ = fmt.Fprintf(stdout, "%s%s\n", strings.Repeat(" ", indentLen*2), currErrLine)
			}
		}
		return fmt.Errorf("conjure verify failed")
	}
	return nil
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

// computeObsoleteFiles identifies existing generated files that are no longer part of the
// current generation output and should be deleted. It compares all Conjure-generated files
// currently on disk in the output directory against the set of files that will be generated,
// returning those that exist but won't be regenerated.
func computeObsoleteFiles(outputDir string, filesToGenerate []*conjure.OutputFile) ([]string, error) {
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

	var files []string
	if err := filepath.WalkDir(outputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if isConjureGeneratedFile(d.Name()) {
			files = append(files, path)
		}

		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to walk output directory %s", outputDir)
	}

	abs := make([]string, 0, len(files))
	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get absolute path for file %s", file)
		}
		abs = append(abs, absPath)
	}

	return abs, nil
}

// isConjureGeneratedFile returns true if the filename matches the pattern for Conjure-generated files.
func isConjureGeneratedFile(filename string) bool {
	return strings.HasSuffix(filename, ".conjure.go") || strings.HasSuffix(filename, ".conjure.json")
}
