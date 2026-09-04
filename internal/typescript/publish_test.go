// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package typescript

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishPackage_PublishesTarballWithoutCredentialArguments(t *testing.T) {
	binDir := t.TempDir()
	captureDir := t.TempDir()
	argsFile := filepath.Join(captureDir, "args")
	workingDirFile := filepath.Join(captureDir, "working-dir")
	userConfigCapture := filepath.Join(captureDir, "userconfig")
	fakeNpm := `#!/bin/sh
set -eu
printf '%s\n' "$*" > "$NPM_ARGS_FILE"
printf '%s\n' "$PWD" > "$NPM_WORKING_DIR_FILE"
cp "$NPM_CONFIG_USERCONFIG" "$NPM_USERCONFIG_CAPTURE"
printf '%s\n' 'published package'
`
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "npm"), []byte(fakeNpm), 0700))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NPM_ARGS_FILE", argsFile)
	t.Setenv("NPM_WORKING_DIR_FILE", workingDirFile)
	t.Setenv("NPM_USERCONFIG_CAPTURE", userConfigCapture)

	packagePath := filepath.Join(t.TempDir(), "api-1.2.3.tgz")
	require.NoError(t, os.WriteFile(packagePath, []byte("package"), 0600))
	npmrcPath := filepath.Join(t.TempDir(), "publish.npmrc")
	require.NoError(t, os.WriteFile(npmrcPath, []byte("//registry.example.com/:_authToken=super-secret\n"), 0600))

	var output bytes.Buffer
	err := PublishPackage(packagePath, "https://registry.example.com/npm/release", npmrcPath, &output)
	require.NoError(t, err)
	assert.Equal(t, "published package\n", output.String())

	args, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	assert.Equal(t, "publish "+packagePath+" --registry https://registry.example.com/npm/release/\n", string(args))
	assert.NotContains(t, string(args), "super-secret")
	assert.NotContains(t, output.String(), "super-secret")

	capturedConfig, err := os.ReadFile(userConfigCapture)
	require.NoError(t, err)
	assert.Contains(t, string(capturedConfig), "super-secret")
	workingDirBytes, err := os.ReadFile(workingDirFile)
	require.NoError(t, err)
	_, err = os.Stat(strings.TrimSpace(string(workingDirBytes)))
	assert.True(t, os.IsNotExist(err), "temporary npm publish working directory should be removed")
}
