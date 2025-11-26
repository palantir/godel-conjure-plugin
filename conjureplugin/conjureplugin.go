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
			if err := verifyConjureOutput(files, outputConf.OutputDir, currParam.SkipDeleteGeneratedFiles, verifyFailedFn, k); err != nil {
				return err
			}
		} else {
			if err := applyConjureOutput(files, outputConf.OutputDir, currParam.SkipDeleteGeneratedFiles); err != nil {
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

// verifyConjureOutput compares expected conjure output against existing files and reports differences.
func verifyConjureOutput(files []*conjure.OutputFile, outputDir string, skipDeleteGeneratedFiles bool, reportFn func(int, string), index int) error {
	expectedChecksums, err := computeExpectedChecksums(files)
	if err != nil {
		return err
	}

	actualChecksums, err := computeActualChecksums(files, outputDir, skipDeleteGeneratedFiles)
	if err != nil {
		return err
	}

	expectedSet := dirchecksum.ChecksumSet{Checksums: expectedChecksums}
	actualSet := dirchecksum.ChecksumSet{Checksums: actualChecksums}
	diff := expectedSet.Diff(actualSet)

	if len(diff.Diffs) > 0 {
		reportFn(index, diff.String())
	}
	return nil
}

// applyConjureOutput writes conjure output files to disk, optionally cleaning up old generated files.
func applyConjureOutput(files []*conjure.OutputFile, outputDir string, skipDeleteGeneratedFiles bool) error {
	if !skipDeleteGeneratedFiles {
		filesToDelete, err := computeFilesToDelete(outputDir)
		if err != nil {
			return err
		}
		for _, path := range filesToDelete {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	for _, file := range files {
		if err := file.Write(); err != nil {
			return err
		}
	}
	return nil
}

// computeExpectedChecksums computes checksums for the generated conjure files.
func computeExpectedChecksums(files []*conjure.OutputFile) (map[string]dirchecksum.FileChecksumInfo, error) {
	checksums := make(map[string]dirchecksum.FileChecksumInfo)
	for _, file := range files {
		checksum, err := computeChecksumForFile(file)
		if err != nil {
			return nil, err
		}
		checksums[file.AbsPath()] = checksum
	}
	return checksums, nil
}

// computeActualChecksums computes checksums for existing files on disk.
// It includes both the expected output files and optionally all existing generated files in the output directory.
func computeActualChecksums(files []*conjure.OutputFile, outputDir string, skipDeleteGeneratedFiles bool) (map[string]dirchecksum.FileChecksumInfo, error) {
	checksums := make(map[string]dirchecksum.FileChecksumInfo)

	// Include all existing generated files if we're checking for extras
	if !skipDeleteGeneratedFiles {
		existingFiles, err := getAllGeneratedFiles(outputDir)
		if err != nil {
			return nil, err
		}
		for _, path := range existingFiles {
			checksums[path] = dirchecksum.FileChecksumInfo{Path: path}
		}
	}

	// Compute checksums for expected files from disk
	for _, file := range files {
		checksum, err := computeChecksumForExistingFile(file.AbsPath())
		if err != nil {
			return nil, err
		}
		checksums[file.AbsPath()] = checksum
	}

	return checksums, nil
}

// computeChecksumForFile computes the checksum for a generated conjure file.
func computeChecksumForFile(file *conjure.OutputFile) (dirchecksum.FileChecksumInfo, error) {
	h := sha256.New()
	bytes, err := file.Render()
	if err != nil {
		return dirchecksum.FileChecksumInfo{}, err
	}
	if _, err := h.Write(bytes); err != nil {
		return dirchecksum.FileChecksumInfo{}, err
	}
	return dirchecksum.FileChecksumInfo{
		Path:           file.AbsPath(),
		SHA256checksum: fmt.Sprintf("%x", h.Sum(nil)),
	}, nil
}

// computeChecksumForExistingFile computes the checksum for a file on disk.
// Returns a FileChecksumInfo with no checksum if the file doesn't exist.
func computeChecksumForExistingFile(path string) (dirchecksum.FileChecksumInfo, error) {
	bytes, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return dirchecksum.FileChecksumInfo{Path: path}, nil
	}
	if err != nil {
		return dirchecksum.FileChecksumInfo{}, err
	}

	h := sha256.New()
	if _, err := h.Write(bytes); err != nil {
		return dirchecksum.FileChecksumInfo{}, err
	}
	return dirchecksum.FileChecksumInfo{
		Path:           path,
		SHA256checksum: fmt.Sprintf("%x", h.Sum(nil)),
	}, nil
}

// computeFilesToDelete returns paths of generated files that should be deleted.
// This includes all existing generated files in the output directory.
func computeFilesToDelete(outputDir string) ([]string, error) {
	return getAllGeneratedFiles(outputDir)
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
