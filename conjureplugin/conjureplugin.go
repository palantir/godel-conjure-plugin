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

		checksumsOfFilesToBeCreated, err := getChecksumsFromConjureGoFiles(files)
		if err != nil {
			return err
		}

		allConjureGoFiles, err := getAllConjureGoFilesInOutputDir(outputConf.OutputDir)
		if err != nil {
			return err
		}

		onDiskChecksums, err := getChecksumsFromOnDiskFiles(allConjureGoFiles)
		if err != nil {
			return err
		}

		diff := (&dirchecksum.ChecksumSet{Checksums: checksumsOfFilesToBeCreated}).Diff(dirchecksum.ChecksumSet{Checksums: onDiskChecksums})
		if currParam.SkipDeleteGeneratedFiles {
			// When skipping delete, remove "extra" files from diff since we don't care about them
			for k, v := range diff.Diffs {
				if v == "extra" {
					delete(diff.Diffs, k)
				}
			}
		}

		if verify {
			if len(diff.Diffs) > 0 {
				verifyFailedFn(k, diff.String())
			}
		} else {
			requiresUpdating := make(map[string]bool)
			var requiresDeleting []string
			for k, v := range diff.Diffs {
				if v == "extra" {
					requiresDeleting = append(requiresDeleting, k)
				} else {
					requiresUpdating[k] = true
				}
			}

			for _, f := range requiresDeleting {
				if err := os.Remove(f); err != nil {
					return err
				}
			}
			for _, file := range files {
				if requiresUpdating[file.AbsPath()] {
					if err := file.Write(); err != nil {
						return err
					}
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

// getChecksumsFromConjureGoFiles computes checksums for generated conjure files.
func getChecksumsFromConjureGoFiles(files []*conjure.OutputFile) (map[string]dirchecksum.FileChecksumInfo, error) {
	result := make(map[string]dirchecksum.FileChecksumInfo)
	for _, file := range files {
		bytes, err := file.Render()
		if err != nil {
			return nil, err
		}
		checksum, err := computeSHA256Hash(bytes)
		if err != nil {
			return nil, err
		}
		result[file.AbsPath()] = dirchecksum.FileChecksumInfo{
			Path:           file.AbsPath(),
			SHA256checksum: checksum,
		}
	}
	return result, nil
}

// getChecksumsFromOnDiskFiles computes checksums for files on disk at the paths specified by the files.
// For files that don't exist, returns an entry with empty checksum.
func getChecksumsFromOnDiskFiles(files []string) (map[string]dirchecksum.FileChecksumInfo, error) {
	result := make(map[string]dirchecksum.FileChecksumInfo)
	for _, file := range files {
		bytes, err := os.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			// File doesn't exist - include with empty checksum
			result[file] = dirchecksum.FileChecksumInfo{Path: file}
			continue
		}
		if err != nil {
			return nil, err
		}
		checksum, err := computeSHA256Hash(bytes)
		if err != nil {
			return nil, err
		}
		result[file] = dirchecksum.FileChecksumInfo{
			Path:           file,
			SHA256checksum: checksum,
		}
	}
	return result, nil
}

// computeSHA256Hash computes the SHA256 hash of the given bytes and returns it as a hex string.
func computeSHA256Hash(data []byte) (string, error) {
	h := sha256.New()
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// getAllConjureGoFilesInOutputDir returns the absolute paths of all Conjure-generated files
// (files ending in .conjure.go or .conjure.json) within the specified output directory.
func getAllConjureGoFilesInOutputDir(outputDir string) ([]string, error) {
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
