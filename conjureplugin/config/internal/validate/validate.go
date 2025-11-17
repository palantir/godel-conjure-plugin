// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// IsSubdirectory returns true if potentialSubDir is a subdirectory of parent, false otherwise.
// This determination is made by normalizing both paths using filepath.Clean and using filepath.Rel to determine if one path
// is a subdirectory of the other. Returns false if filepath.Rel returns an error (for example, if one path is
// absolute and the other is relative). Does not resolve symlinks.
func IsSubdirectory(parent, potentialSubDir string) bool {
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
