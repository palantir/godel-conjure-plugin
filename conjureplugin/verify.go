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
	"os"
	"path/filepath"

	"github.com/palantir/conjure-go/v6/conjure"
	"github.com/palantir/godel/v2/pkg/dirchecksum"
	"github.com/pkg/errors"
)

func diffOnDisk(projectDir string, files []*conjure.OutputFile, outputDir string, skipDeleteGeneratedFiles bool) (dirchecksum.ChecksumsDiff, error) {
	newChecksums, err := checksumRenderedFiles(files, projectDir)
	if err != nil {
		return dirchecksum.ChecksumsDiff{}, errors.Wrap(err, "failed to compute generated checksums")
	}
	originalChecksums, err := checksumOnDiskFiles(files, projectDir, outputDir, skipDeleteGeneratedFiles)
	if err != nil {
		return dirchecksum.ChecksumsDiff{}, errors.Wrap(err, "failed to compute on-disk checksums")
	}

	return originalChecksums.Diff(newChecksums), nil
}

func checksumRenderedFiles(files []*conjure.OutputFile, projectDir string) (dirchecksum.ChecksumSet, error) {
	set := dirchecksum.ChecksumSet{
		RootDir:   projectDir,
		Checksums: map[string]dirchecksum.FileChecksumInfo{},
	}
	for _, file := range files {
		relPath, err := filepath.Rel(projectDir, file.AbsPath())
		if err != nil {
			return dirchecksum.ChecksumSet{}, err
		}
		output, err := file.Render()
		if err != nil {
			return dirchecksum.ChecksumSet{}, err
		}
		h := sha256.New()
		_, err = h.Write(output)
		if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to checksum generated content for %s", file.AbsPath())
		}
		set.Checksums[relPath] = dirchecksum.FileChecksumInfo{
			Path:           relPath,
			IsDir:          false,
			SHA256checksum: fmt.Sprintf("%x", h.Sum(nil)),
		}
	}
	return set, nil
}

func checksumOnDiskFiles(files []*conjure.OutputFile, projectDir string, outputDir string, skipDeleteGeneratedFiles bool) (dirchecksum.ChecksumSet, error) {
	set := dirchecksum.ChecksumSet{
		RootDir:   projectDir,
		Checksums: map[string]dirchecksum.FileChecksumInfo{},
	}

	// Build a set of files that WILL be generated (for quick lookup)
	willBeGenerated := make(map[string]bool)
	for _, file := range files {
		willBeGenerated[file.AbsPath()] = true
	}

	// Determine which files to checksum on disk
	var filesToChecksum []string

	if skipDeleteGeneratedFiles {
		// OLD BEHAVIOR: Only checksum files that will be generated
		for _, file := range files {
			filesToChecksum = append(filesToChecksum, file.AbsPath())
		}
	} else {
		// NEW BEHAVIOR: Checksum ALL existing generated files in outputDir
		allExistingFiles, err := getAllGeneratedFiles(outputDir)
		if err != nil {
			return dirchecksum.ChecksumSet{}, err
		}
		filesToChecksum = allExistingFiles
	}

	// Compute checksums for the determined set of files
	for _, absPath := range filesToChecksum {
		relPath, err := filepath.Rel(projectDir, absPath)
		if err != nil {
			return dirchecksum.ChecksumSet{}, err
		}

		f, err := os.Open(absPath)
		if os.IsNotExist(err) {
			// File doesn't exist on disk - skip it
			continue
		} else if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to open file for checksum %s", absPath)
		}
		defer func() {
			// file is opened for reading only, so safe to ignore errors on close
			_ = f.Close()
		}()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to checksum on-disk content for %s", absPath)
		}

		set.Checksums[relPath] = dirchecksum.FileChecksumInfo{
			Path:           relPath,
			SHA256checksum: fmt.Sprintf("%x", h.Sum(nil)),
		}
	}

	return set, nil
}
