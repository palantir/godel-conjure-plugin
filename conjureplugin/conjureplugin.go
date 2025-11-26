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
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/palantir/conjure-go/v6/conjure"
	conjurego "github.com/palantir/conjure-go/v6/conjure"
	"github.com/palantir/conjure-go/v6/conjure-api/conjure/spec"
	"github.com/palantir/godel/v2/pkg/dirchecksum"
	"github.com/palantir/pkg/matcher"
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
		conjureDef, err := conjureDefinitionFromParam(currParam)
		if err != nil {
			return err
		}

		outputConf := conjure.OutputConfiguration{
			OutputDir:            filepath.Join(projectDir, currParam.OutputDir),
			GenerateServer:       currParam.Server,
			GenerateCLI:          currParam.CLI,
			GenerateFuncsVisitor: currParam.AcceptFuncs,
		}

		files, err := conjure.GenerateOutputFiles(conjureDef, outputConf)
		if err != nil {
			return errors.Wrap(err, "failed to generate conjure output files")
		}

		if verify {
			diffStr, err := verifyConjureOutput(files, outputConf.OutputDir, currParam.SkipDeleteGeneratedFiles)
			if err != nil {
				return err
			}
			if diffStr != "" {
				verifyFailedFn(k, diffStr)
			}
		} else {
			if !currParam.SkipDeleteGeneratedFiles {
				allGeneratedFilePaths_Abs, err := getAllGeneratedFiles(outputConf.OutputDir)
				if err != nil {
					return err
				}
				for _, abs := range allGeneratedFilePaths_Abs {
					if err := os.Remove(abs); err != nil {
						return err
					}
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

// checksumSetBuilder builds a ChecksumSet using a fluent builder pattern.
type checksumSetBuilder struct {
	checksums map[string]dirchecksum.FileChecksumInfo
}

// newChecksumSetBuilder creates a new checksumSetBuilder.
func newChecksumSetBuilder() *checksumSetBuilder {
	return &checksumSetBuilder{
		checksums: make(map[string]dirchecksum.FileChecksumInfo),
	}
}

// addFromGeneratedFiles adds checksums for generated conjure files.
func (b *checksumSetBuilder) addFromGeneratedFiles(files []*conjure.OutputFile) error {
	for _, file := range files {
		bytes, err := file.Render()
		if err != nil {
			return err
		}
		b.checksums[file.AbsPath()] = dirchecksum.FileChecksumInfo{
			Path:           file.AbsPath(),
			SHA256checksum: computeChecksum(bytes),
		}
	}
	return nil
}

// addFromExistingFiles adds checksums for files that exist on disk.
func (b *checksumSetBuilder) addFromExistingFiles(files []*conjure.OutputFile) error {
	for _, file := range files {
		bytes, err := os.ReadFile(file.AbsPath())
		if errors.Is(err, fs.ErrNotExist) {
			b.checksums[file.AbsPath()] = dirchecksum.FileChecksumInfo{Path: file.AbsPath()}
		} else if err != nil {
			return err
		} else {
			b.checksums[file.AbsPath()] = dirchecksum.FileChecksumInfo{
				Path:           file.AbsPath(),
				SHA256checksum: computeChecksum(bytes),
			}
		}
	}
	return nil
}

// addFromDirectory adds all generated files from a directory without computing checksums.
func (b *checksumSetBuilder) addFromDirectory(outputDir string) error {
	existingFiles, err := getAllGeneratedFiles(outputDir)
	if err != nil {
		return err
	}
	for _, path := range existingFiles {
		b.checksums[path] = dirchecksum.FileChecksumInfo{Path: path}
	}
	return nil
}

// build returns the constructed ChecksumSet.
func (b *checksumSetBuilder) build() dirchecksum.ChecksumSet {
	return dirchecksum.ChecksumSet{Checksums: b.checksums}
}

// verifyConjureOutput compares expected conjure output files against existing files on disk.
// Returns a diff string if differences are found, or empty string if files match.
func verifyConjureOutput(files []*conjure.OutputFile, outputDir string, skipDeleteGeneratedFiles bool) (string, error) {
	expectedBuilder := newChecksumSetBuilder()
	if err := expectedBuilder.addFromGeneratedFiles(files); err != nil {
		return "", err
	}
	expectedSet := expectedBuilder.build()

	actualBuilder := newChecksumSetBuilder()
	if !skipDeleteGeneratedFiles {
		if err := actualBuilder.addFromDirectory(outputDir); err != nil {
			return "", err
		}
	}
	if err := actualBuilder.addFromExistingFiles(files); err != nil {
		return "", err
	}
	actualSet := actualBuilder.build()

	diff := expectedSet.Diff(actualSet)
	if len(diff.Diffs) > 0 {
		return diff.String(), nil
	}
	return "", nil
}

// computeChecksum computes the SHA256 checksum for the given bytes.
func computeChecksum(data []byte) string {
	h := sha256.New()
	_, _ = h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
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
