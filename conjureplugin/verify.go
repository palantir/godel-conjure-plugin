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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/palantir/conjure-go/v6/conjure"
	"github.com/palantir/godel/v2/pkg/dirchecksum"
	"github.com/pkg/errors"
)

// checksumRenderedFiles computes checksums for generated conjure files.
// Takes in-memory OutputFile objects, renders each one to bytes, and computes SHA256 checksums.
// Returns a map where keys are absolute file paths and values contain path + checksum.
func checksumRenderedFiles(files []*conjure.OutputFile) (dirchecksum.ChecksumSet, error) {
	set := dirchecksum.ChecksumSet{
		Checksums: map[string]dirchecksum.FileChecksumInfo{},
	}
	for _, file := range files {
		// Render the file content to output (this is the generated code).
		output, err := file.Render()
		if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to render file %s", file.AbsPath())
		}
		// Compute SHA256 hash of the content.
		checksum, err := computeSHA256Hash(output)
		if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to compute checksum for file %s", file.AbsPath())
		}
		set.Checksums[filepath.Clean(file.AbsPath())] = dirchecksum.FileChecksumInfo{
			Path:           filepath.Clean(file.AbsPath()),
			SHA256checksum: checksum,
		}
	}
	return set, nil
}

// checksumOnDiskFiles computes checksums for files on disk at the specified paths.
// Paths are normalized to absolute paths for consistent comparison.
// For files that don't exist, returns an entry with empty checksum.
func checksumOnDiskFiles(files []string) (dirchecksum.ChecksumSet, error) {
	set := dirchecksum.ChecksumSet{
		Checksums: map[string]dirchecksum.FileChecksumInfo{},
	}
	for _, file := range files {
		// Normalize to absolute path for consistent comparison.
		absPath, err := filepath.Abs(file)
		if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to get absolute path for %s", file)
		}

		// Read the file from disk.
		bytes, err := os.ReadFile(absPath)
		if errors.Is(err, fs.ErrNotExist) {
			// File doesn't exist - include entry with empty checksum.
			// The diff algorithm uses empty checksums to detect "missing" files.
			set.Checksums[absPath] = dirchecksum.FileChecksumInfo{Path: absPath}
			continue
		}
		if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to read file %s", absPath)
		}
		// Compute SHA256 hash of the file content.
		checksum, err := computeSHA256Hash(bytes)
		if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to compute checksum for file %s", absPath)
		}
		set.Checksums[absPath] = dirchecksum.FileChecksumInfo{
			Path:           absPath,
			SHA256checksum: checksum,
		}
	}
	return set, nil
}

// computeSHA256Hash computes the SHA256 hash of the given bytes and returns it as a hex string.
// Example output: "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
func computeSHA256Hash(data []byte) (string, error) {
	h := sha256.New()
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
