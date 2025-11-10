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
	"path"
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
			OutputDir:            path.Join(projectDir, outputDir),
			GenerateServer:       currParam.Server,
			GenerateCLI:          currParam.CLI,
			GenerateFuncsVisitor: currParam.AcceptFuncs,
		}
		if verify {
			diff, err := diffOnDisk(conjureDef, projectDir, outputConf)
			if err != nil {
				return err
			}
			if len(diff.Diffs) > 0 {
				verifyFailedFn(k, diff.String())
			}
		} else {
			// Delete old generated files before regeneration unless skipped
			if !currParam.SkipDeleteGeneratedFiles {
				if err := deleteGeneratedFiles(outputConf.OutputDir); err != nil {
					return errors.Wrapf(err, "failed to delete old generated files in %s", outputConf.OutputDir)
				}
			}
			if err := conjure.Generate(conjureDef, outputConf); err != nil {
				return err
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

// deleteGeneratedFiles removes all Conjure-generated files (*.conjure.go and *.conjure.json)
// from the specified output directory and its subdirectories.
func deleteGeneratedFiles(outputDir string) error {
	// Check if the output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to delete
		return nil
	}

	// Walk the directory tree and delete files matching the Conjure-generated pattern
	return filepath.WalkDir(outputDir, deleteConjureFile)
}

// deleteConjureFile is a filepath.WalkDirFunc that deletes Conjure-generated files.
func deleteConjureFile(path string, d os.DirEntry, err error) error {
	if err != nil {
		// If we can't access a file/directory, skip it
		return nil
	}

	// Only process files, not directories
	if d.IsDir() {
		return nil
	}

	// Check if the file matches the Conjure-generated pattern
	name := d.Name()
	if strings.HasSuffix(name, ".conjure.go") || strings.HasSuffix(name, ".conjure.json") {
		if err := os.Remove(path); err != nil {
			return errors.Wrapf(err, "failed to delete generated file %s", path)
		}
	}

	return nil
}
