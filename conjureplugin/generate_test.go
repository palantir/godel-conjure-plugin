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

package conjureplugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllGeneratedFiles(t *testing.T) {
	t.Run("finds conjure.go files", func(t *testing.T) {
		tmpDir := t.TempDir()

		createFile(t, filepath.Join(tmpDir, "aliases.conjure.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "structs.conjure.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "regular.go"), "package test")
		createFile(t, filepath.Join(tmpDir, "conjure.go"), "package test")

		files, err := getAllGeneratedFiles(tmpDir)
		require.NoError(t, err)
		require.Len(t, files, 2)

		// Convert to basenames for easier assertions
		basenames := make([]string, len(files))
		for i, f := range files {
			basenames[i] = filepath.Base(f)
		}
		assert.Contains(t, basenames, "aliases.conjure.go")
		assert.Contains(t, basenames, "structs.conjure.go")
		assert.NotContains(t, basenames, "regular.go")
		assert.NotContains(t, basenames, "conjure.go")
	})

	t.Run("finds conjure.json files", func(t *testing.T) {
		tmpDir := t.TempDir()

		createFile(t, filepath.Join(tmpDir, "extensions.conjure.json"), "{}")
		createFile(t, filepath.Join(tmpDir, "config.json"), "{}")

		files, err := getAllGeneratedFiles(tmpDir)
		require.NoError(t, err)
		require.Len(t, files, 1)

		assert.Equal(t, "extensions.conjure.json", filepath.Base(files[0]))
	})

	t.Run("finds files in subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		subDir1 := filepath.Join(tmpDir, "pkg1")
		subDir2 := filepath.Join(tmpDir, "pkg2", "nested")
		require.NoError(t, os.MkdirAll(subDir1, 0755))
		require.NoError(t, os.MkdirAll(subDir2, 0755))

		createFile(t, filepath.Join(tmpDir, "top.conjure.go"), "package test")
		createFile(t, filepath.Join(subDir1, "pkg1.conjure.go"), "package pkg1")
		createFile(t, filepath.Join(subDir2, "nested.conjure.go"), "package nested")

		files, err := getAllGeneratedFiles(tmpDir)
		require.NoError(t, err)
		assert.Len(t, files, 3)
	})

	t.Run("handles missing directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		files, err := getAllGeneratedFiles(filepath.Join(tmpDir, "does-not-exist"))
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("returns absolute paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		createFile(t, filepath.Join(tmpDir, "test.conjure.go"), "package test")

		files, err := getAllGeneratedFiles(tmpDir)
		require.NoError(t, err)
		require.Len(t, files, 1)

		assert.True(t, filepath.IsAbs(files[0]), "expected absolute path, got: %s", files[0])
	})
}

func createFile(t *testing.T, path string, content string) {
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}
