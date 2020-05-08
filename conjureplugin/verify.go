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

	"github.com/palantir/conjure-go/v5/conjure"
	"github.com/palantir/conjure-go/v5/conjure-api/conjure/spec"
	"github.com/palantir/godel/v2/pkg/dirchecksum"
	"github.com/pkg/errors"
)

// diffOnDisk generates the conjure files in memory and compares checksums to on-disk files.
func diffOnDisk(conjureDefinition spec.ConjureDefinition, projectDir string, outputConf conjure.OutputConfiguration) (dirchecksum.ChecksumsDiff, error) {
	files, err := conjure.GenerateOutputFiles(conjureDefinition, outputConf)
	if err != nil {
		return dirchecksum.ChecksumsDiff{}, errors.Wrap(err, "conjure failed")
	}
	originalChecksums, err := checksumOnDiskFiles(files, projectDir)
	if err != nil {
		return dirchecksum.ChecksumsDiff{}, errors.Wrap(err, "failed to compute on-disk checksums")
	}
	newChecksums, err := checksumRenderedFiles(files, projectDir)
	if err != nil {
		return dirchecksum.ChecksumsDiff{}, errors.Wrap(err, "failed to compute generated checksums")
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

func checksumOnDiskFiles(files []*conjure.OutputFile, projectDir string) (dirchecksum.ChecksumSet, error) {
	set := dirchecksum.ChecksumSet{
		RootDir:   projectDir,
		Checksums: map[string]dirchecksum.FileChecksumInfo{},
	}
	for _, file := range files {
		relPath, err := filepath.Rel(projectDir, file.AbsPath())
		if err != nil {
			return dirchecksum.ChecksumSet{}, err
		}

		f, err := os.Open(file.AbsPath())
		if os.IsNotExist(err) {
			// skip nonexistent files
			continue
		} else if err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to open file for checksum %s", file.AbsPath())
		}
		defer func() {
			// file is opened for reading only, so safe to ignore errors on close
			_ = f.Close()
		}()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return dirchecksum.ChecksumSet{}, errors.Wrapf(err, "failed to checksum on-disk content for %s", file.AbsPath())
		}
		set.Checksums[relPath] = dirchecksum.FileChecksumInfo{
			Path:           relPath,
			SHA256checksum: fmt.Sprintf("%x", h.Sum(nil)),
		}
	}
	return set, nil
}
