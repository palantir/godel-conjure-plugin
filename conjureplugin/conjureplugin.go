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

		// Generate the conjure output files in memory (not written to disk yet).
		files, err := conjure.GenerateOutputFiles(conjureDef, outputConf)
		if err != nil {
			return errors.Wrap(err, "failed to generate conjure output files")
		}

		// Compute checksums for the files we're about to create/update.
		// These are computed from the in-memory generated content.
		checksumsOfFilesToBeCreated, err := getChecksumsFromConjureGoFiles(files)
		if err != nil {
			return err
		}

		// Find all existing conjure-generated files in the output directory
		// (files ending in .conjure.go or .conjure.json).
		allConjureGoFiles, err := getAllConjureGoFilesInOutputDir(outputConf.OutputDir)
		if err != nil {
			return err
		}

		// Compute checksums for all files currently on disk.
		// Files that don't exist get an empty checksum.
		onDiskChecksums, err := getChecksumsFromOnDiskFiles(allConjureGoFiles)
		if err != nil {
			return err
		}

		// Compare expected vs actual checksums to find differences.
		// The diff will contain entries like:
		//   "checksum changed..."	- file exists but content differs
		//   "missing"				- file should exist but doesn't
		//   "extra"				- file exists but shouldn't (stale file to delete)
		diff := (&dirchecksum.ChecksumSet{Checksums: checksumsOfFilesToBeCreated}).Diff(dirchecksum.ChecksumSet{Checksums: onDiskChecksums})
		if currParam.SkipDeleteGeneratedFiles {
			// When configured to skip deletion, filter out "extra" files from the diff.
			// This means we won't report them in verify mode or delete them in write mode.
			for k, v := range diff.Diffs {
				if v == "extra" {
					delete(diff.Diffs, k)
				}
			}
		}

		if verify {
			// Verify mode: report any differences but don't modify files.
			if len(diff.Diffs) > 0 {
				verifyFailedFn(k, diff.String())
			}
		} else {
			// Write mode: apply the changes to disk.

			// Categorize files by what action is needed.
			requiresUpdating := make(map[string]bool)	// Files that need to be written (changed or missing)
			var requiresDeleting []string				// Files that need to be deleted (extra/stale)
			for k, v := range diff.Diffs {
				if v == "extra" {
					requiresDeleting = append(requiresDeleting, k)
				} else {
					// "changed" or "missing" both require writing
					requiresUpdating[k] = true
				}
			}

			// Delete stale files first.
			for _, f := range requiresDeleting {
				if err := os.Remove(f); err != nil {
					return err
				}
			}

			// Write only the files that need updating (optimization: skip unchanged files).
			for _, file := range files {
				if requiresUpdating[file.AbsPath()] {
					fmt.Printf("file.AbsPath(): %v\n", file.AbsPath())
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
// Takes in-memory OutputFile objects, renders each one to bytes, and computes SHA256 checksums.
// Returns a map where keys are absolute file paths and values contain path + checksum.
func getChecksumsFromConjureGoFiles(files []*conjure.OutputFile) (map[string]dirchecksum.FileChecksumInfo, error) {
	result := make(map[string]dirchecksum.FileChecksumInfo)
	for _, file := range files {
		// Render the file content to bytes (this is the generated code).
		bytes, err := file.Render()
		if err != nil {
			return nil, err
		}
		// Compute SHA256 hash of the content.
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

// getChecksumsFromOnDiskFiles computes checksums for files on disk at the specified paths.
// For files that don't exist, returns an entry with empty checksum (needed for proper diff calculation).
// Returns a map where keys are file paths and values contain path + checksum (or just path if missing).
func getChecksumsFromOnDiskFiles(files []string) (map[string]dirchecksum.FileChecksumInfo, error) {
	result := make(map[string]dirchecksum.FileChecksumInfo)
	for _, file := range files {
		// Read the file from disk.
		bytes, err := os.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			// File doesn't exist - include entry with empty checksum.
			// The diff algorithm uses empty checksums to detect "missing" files.
			result[file] = dirchecksum.FileChecksumInfo{Path: file}
			continue
		}
		if err != nil {
			return nil, err
		}
		// Compute SHA256 hash of the file content.
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
// Example output: "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
func computeSHA256Hash(data []byte) (string, error) {
	h := sha256.New()
	// Write() on a hash.Hash never returns an error, but we check anyway for safety.
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	// Format the hash as a lowercase hexadecimal string.
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// getAllConjureGoFilesInOutputDir returns the absolute paths of all Conjure-generated files
// (files ending in .conjure.go or .conjure.json) within the specified output directory.
// Returns an empty slice if the directory doesn't exist (not an error - directory may not exist yet).
// This is used to find all existing generated files so we can detect stale ones that need deletion.
func getAllConjureGoFilesInOutputDir(outputDir string) ([]string, error) {
	// Check if the output directory exists.
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		// Directory doesn't exist - not an error, just means no files to find.
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to stat output directory %s", outputDir)
	}

	// Match files ending in .conjure.go or .conjure.json.
	// These are the suffixes we use for all conjure-generated code.
	include := matcher.Name(`.*\.conjure\.(go|json)$`)

	// List all matching files (returns paths relative to outputDir).
	relPaths, err := matcher.ListFiles(outputDir, include, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list files in output directory %s", outputDir)
	}

	// Convert relative paths to absolute paths, filtering out directories.
	// matcher.ListFiles can return both files and directories that match the pattern,
	// so we need to filter to only actual files.
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
