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
	var verifyFailedIndex []int
	verifyFailedErrors := make(map[int]string)
	verifyFailedFn := func(name int, errStr string) {
		verifyFailedIndex = append(verifyFailedIndex, name)
		verifyFailedErrors[name] = errStr
	}

	k := 0
	for _, currParam := range params.OrderedParams() {
		// Convert output directory to absolute path for consistent path comparisons.
		outputDir := filepath.Join(projectDir, currParam.OutputDir)
		absOutputDir, err := filepath.Abs(outputDir)
		if err != nil {
			return errors.Wrapf(err, "failed to get absolute path for output directory %s", outputDir)
		}

		conjureDef, err := conjureDefinitionFromParam(currParam)
		if err != nil {
			return err
		}

		outputConf := conjure.OutputConfiguration{
			OutputDir:            absOutputDir,
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
		// Checksums are keyed by absolute paths (using file.AbsPath()) for consistent diff comparison.
		renderedChecksums, err := checksumRenderedFiles(files)
		if err != nil {
			return errors.Wrap(err, "failed to compute checksums for generated files")
		}

		// Find all existing conjure-generated files in the output directory
		// (files ending in .conjure.go or .conjure.json).
		// Since absOutputDir is absolute, the returned paths will also be absolute.
		allConjureGoFiles, err := getAllConjureGoFilesInOutputDir(absOutputDir)
		if err != nil {
			return errors.Wrapf(err, "failed to list existing conjure files in %s", absOutputDir)
		}

		// Compute checksums for the existing conjure-generated files on disk.
		// Checksums are keyed by absolute paths (via filepath.Abs) for consistent diff comparison.
		onDiskChecksums, err := checksumOnDiskFiles(allConjureGoFiles)
		if err != nil {
			return errors.Wrap(err, "failed to compute checksums for on-disk files")
		}

		// Compare expected vs actual checksums to find differences.
		// The diff will contain entries like:
		//   "checksum changed..."	- file exists but content differs
		//   "missing"				- file should exist but doesn't
		//   "extra"				- file exists but shouldn't (stale file to delete)
		diff := renderedChecksums.Diff(onDiskChecksums)
		if currParam.SkipDeleteGeneratedFiles {
			// When configured to skip deletion, filter out "extra" files from the diff.
			// This means we won't report them in verify mode or delete them in write mode.
			for path, msg := range diff.Diffs {
				if msg == "extra" {
					delete(diff.Diffs, path)
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
			requiresWriting := make(map[string]bool) // Files that need to be written
			var requiresDeleting []string            // Stale files to delete
			for path, diffType := range diff.Diffs {
				switch diffType {
				case "extra":
					// Stale file that should no longer exist
					requiresDeleting = append(requiresDeleting, path)
				default:
					// All other cases require writing: "missing", "checksum changed...", etc.
					requiresWriting[filepath.Clean(path)] = true
				}
			}

			// Delete stale files first.
			for _, f := range requiresDeleting {
				if err := os.Remove(f); err != nil {
					return errors.Wrapf(err, "failed to delete stale file %s", f)
				}
			}

			// Write only the files that need updating (optimization: skip unchanged files).
			for _, file := range files {
				if requiresWriting[filepath.Clean(file.AbsPath())] {
					if err := file.Write(); err != nil {
						return errors.Wrapf(err, "failed to write file %s", file.AbsPath())
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

// getAllConjureGoFilesInOutputDir returns paths of all Conjure-generated files
// (files ending in .conjure.go or .conjure.json) within the specified output directory.
// Returns paths with outputDir prepended (absolute if outputDir is absolute, relative if relative).
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
	pathsRelativeToOutputDir, err := matcher.ListFiles(outputDir, include, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list files in output directory %s", outputDir)
	}

	// Convert relative paths to paths with outputDir prepended, filtering out directories.
	// matcher.ListFiles can return both files and directories that match the pattern,
	// so we need to filter to only actual files.
	var pathsWithOutputDir []string
	for _, relPath := range pathsRelativeToOutputDir {
		fullPath := filepath.Join(outputDir, relPath)
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to stat %s", fullPath)
		}
		if !fileInfo.IsDir() {
			pathsWithOutputDir = append(pathsWithOutputDir, fullPath)
		}
	}

	return pathsWithOutputDir, nil
}
