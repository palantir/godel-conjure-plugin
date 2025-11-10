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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteGeneratedFiles(t *testing.T) {
	t.Run("deletes conjure.go files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		createFile(t, filepath.Join(tmpDir, "aliases.conjure.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "structs.conjure.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "regular.go"), "package test")

		// Run delete
		err := deleteGeneratedFiles(tmpDir)
		require.NoError(t, err)

		// Verify conjure files are deleted
		assertFileNotExists(t, filepath.Join(tmpDir, "aliases.conjure.go"))
		assertFileNotExists(t, filepath.Join(tmpDir, "structs.conjure.go"))

		// Verify regular file still exists
		assertFileExists(t, filepath.Join(tmpDir, "regular.go"))
	})

	t.Run("deletes conjure.json files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		createFile(t, filepath.Join(tmpDir, "extensions.conjure.json"), "{}")
		createFile(t, filepath.Join(tmpDir, "config.json"), "{}")

		// Run delete
		err := deleteGeneratedFiles(tmpDir)
		require.NoError(t, err)

		// Verify conjure json is deleted
		assertFileNotExists(t, filepath.Join(tmpDir, "extensions.conjure.json"))

		// Verify regular json still exists
		assertFileExists(t, filepath.Join(tmpDir, "config.json"))
	})

	t.Run("deletes files in subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create nested directory structure
		subDir1 := filepath.Join(tmpDir, "pkg1")
		subDir2 := filepath.Join(tmpDir, "pkg2", "nested")
		require.NoError(t, os.MkdirAll(subDir1, 0755))
		require.NoError(t, os.MkdirAll(subDir2, 0755))

		// Create test files
		createFile(t, filepath.Join(tmpDir, "top.conjure.go"), "package test")
		createFile(t, filepath.Join(subDir1, "pkg1.conjure.go"), "package pkg1")
		createFile(t, filepath.Join(subDir2, "nested.conjure.go"), "package nested")
		createFile(t, filepath.Join(subDir1, "regular.go"), "package pkg1")

		// Run delete
		err := deleteGeneratedFiles(tmpDir)
		require.NoError(t, err)

		// Verify all conjure files are deleted
		assertFileNotExists(t, filepath.Join(tmpDir, "top.conjure.go"))
		assertFileNotExists(t, filepath.Join(subDir1, "pkg1.conjure.go"))
		assertFileNotExists(t, filepath.Join(subDir2, "nested.conjure.go"))

		// Verify regular file still exists
		assertFileExists(t, filepath.Join(subDir1, "regular.go"))
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist")

		// Should not error on non-existent directory
		err := deleteGeneratedFiles(nonExistent)
		require.NoError(t, err)
	})

	t.Run("handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Should not error on empty directory
		err := deleteGeneratedFiles(tmpDir)
		require.NoError(t, err)
	})

	t.Run("only deletes exact pattern matches", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create files with similar but not exact patterns
		createFile(t, filepath.Join(tmpDir, "file.conjure.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "file.conjure.go.bak"), "package test")
		createFile(t, filepath.Join(tmpDir, "conjure.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "file_conjure.go"), "package test")

		// Run delete
		err := deleteGeneratedFiles(tmpDir)
		require.NoError(t, err)

		// Verify only exact match is deleted
		assertFileNotExists(t, filepath.Join(tmpDir, "file.conjure.go"))
		assertFileExists(t, filepath.Join(tmpDir, "file.conjure.go.bak"))
		assertFileExists(t, filepath.Join(tmpDir, "conjure.go"))
		assertFileExists(t, filepath.Join(tmpDir, "file_conjure.go"))
	})
}

// Helper functions

func createFile(t *testing.T, path string, content string) {
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

func assertFileExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.NoError(t, err, "expected file to exist: %s", path)
}

func assertFileNotExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "expected file to not exist: %s", path)
}
