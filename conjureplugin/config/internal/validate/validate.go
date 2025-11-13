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

package validate

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// GetConflictingOutputDirs checks for conflicts in output directories:
// 1. Multiple projects using the same output directory
// 2. Parent-child directory relationships between output directories
// Returns a slice of errors describing any conflicts found.
func GetConflictingOutputDirs(outputDirToProjects map[string][]string) []error {
	var errs []error

	sortedOutputDirs := slices.Sorted(maps.Keys(outputDirToProjects))
	for _, outputDir := range sortedOutputDirs {
		projects := outputDirToProjects[outputDir]
		if len(projects) <= 1 {
			continue
		}
		errs = append(errs, errors.Errorf("Projects %v are configured with the same outputDir %q, which may cause conflicts when generating Conjure output", projects, outputDir))
	}

	for i, dir1 := range sortedOutputDirs {
		for _, dir2 := range sortedOutputDirs[i+1:] {
			var parentDir string
			var subdir string
			if isSubdirectory(dir1, dir2) {
				parentDir = dir1
				subdir = dir2
			} else if isSubdirectory(dir2, dir1) {
				parentDir = dir2
				subdir = dir1
			} else {
				// no subdirectory issues
				continue
			}
			errs = append(errs,
				errors.Errorf(
					"Projects %v are configured with outputDir %q, which is a subdirectory of the outputDir %q configured for projects %v, which may cause conflicts when generating Conjure output",
					outputDirToProjects[subdir],
					subdir,
					parentDir,
					outputDirToProjects[parentDir],
				),
			)
		}
	}

	return errs
}

// isSubdirectory returns true if potentialSubDir is a subdirectory of parent, false otherwise.
// This determination is made by normalizing both paths using filepath.Clean and using filepath.Rel to determine if one path
// is a subdirectory of the other. Returns false if filepath.Rel returns an error (for example, if one path is
// absolute and the other is relative). Does not resolve symlinks.
func isSubdirectory(parent, potentialSubDir string) bool {
	parent = filepath.Clean(parent)
	potentialSubDir = filepath.Clean(potentialSubDir)
	rel, err := filepath.Rel(parent, potentialSubDir)
	return err == nil && !strings.HasPrefix(rel, "..") && rel != "."
}

// ValidateProjectName validates that a project name is safe to use as part of a file path.
// Returns an error if:
// - The name contains forward slashes (/) or backslashes (\)
// - The name is "." or ".."
func ValidateProjectName(projectName string) error {
	if strings.Contains(projectName, "/") || strings.Contains(projectName, "\\") {
		return errors.Errorf("project name %q cannot contain path separators (/ or \\)", projectName)
	}
	if projectName == "." || projectName == ".." {
		return errors.Errorf("project name cannot be %q", projectName)
	}
	return nil
}
