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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackage_RunsNpmInSingleTempDir(t *testing.T) {
	callsFile, _ := setupFakeNpmExecutable(t)
	outputDir := t.TempDir()

	packagePath, err := Package(
		[]byte(`{"version":1,"errors":[],"types":[],"services":[],"extensions":{}}`),
		Params{
			PackageName:              "@palantir/test-api",
			Version:                  "1.2.3",
			GenerateThrowingServices: true,
		},
		outputDir,
		os.Stdout,
	)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "palantir-test-api-1.2.3.tgz"), packagePath)
	assert.FileExists(t, packagePath)

	calls := readNpmCalls(t, callsFile)
	require.Len(t, calls, 3)
	assert.Equal(t, "install --no-package-lock --no-production", calls[0].args)
	assert.Equal(t, "run-script build", calls[1].args)
	assert.Equal(t, "pack --json", calls[2].args)
	assert.Equal(t, calls[0].dir, calls[1].dir)
	assert.Equal(t, calls[1].dir, calls[2].dir)
	_, err = os.Stat(calls[0].dir)
	assert.True(t, os.IsNotExist(err), "temporary build directory should be removed")
}

func TestPackage_EmbedsProductDependenciesInManifest(t *testing.T) {
	_, packageJSONCapturePath := setupFakeNpmExecutable(t)
	productDependencies := []byte(`[
  {
    "product-group": "com.palantir.example",
    "product-name": "service",
    "minimum-version": "1.0.0",
    "recommended-version": "1.2.0",
    "maximum-version": "2.x.x",
    "optional": true
  }
]`)

	_, err := Package(
		[]byte(`{"version":1,"errors":[],"types":[],"services":[],"extensions":{}}`),
		Params{
			PackageName:                 "@palantir/test-api",
			Version:                     "1.2.3",
			ProductDependencies:         productDependencies,
			GenerateThrowingServices:    true,
			GenerateNonThrowingServices: false,
		},
		t.TempDir(),
		os.Stdout,
	)
	require.NoError(t, err)

	packageJSONBytes, err := os.ReadFile(packageJSONCapturePath)
	require.NoError(t, err)
	var packageJSON struct {
		SLS struct {
			Dependencies map[string]struct {
				MinVersion         string `json:"minVersion"`
				RecommendedVersion string `json:"recommendedVersion"`
				MaxVersion         string `json:"maxVersion"`
				Optional           bool   `json:"optional"`
			} `json:"dependencies"`
		} `json:"sls"`
	}
	require.NoError(t, json.Unmarshal(packageJSONBytes, &packageJSON))
	assert.Equal(t, "1.0.0", packageJSON.SLS.Dependencies["com.palantir.example:service"].MinVersion)
	assert.Equal(t, "1.2.0", packageJSON.SLS.Dependencies["com.palantir.example:service"].RecommendedVersion)
	assert.Equal(t, "2.x.x", packageJSON.SLS.Dependencies["com.palantir.example:service"].MaxVersion)
	assert.True(t, packageJSON.SLS.Dependencies["com.palantir.example:service"].Optional)
}

func TestPackage_RejectsNpmrcInTarball(t *testing.T) {
	setupFakeNpmExecutable(t)
	t.Setenv("NPM_PACK_JSON", `[{"filename":"palantir-test-api-1.2.3.tgz","files":[{"path":".npmrc"}]}]`)
	_, err := Package(
		[]byte(`{"version":1,"errors":[],"types":[],"services":[],"extensions":{}}`),
		Params{PackageName: "@palantir/test-api", Version: "1.2.3", GenerateThrowingServices: true},
		t.TempDir(),
		os.Stdout,
	)
	require.EqualError(t, err, "npm pack attempted to include .npmrc in the package")
}

func TestPackage_UsesExternalNpmUserConfig(t *testing.T) {
	setupFakeNpmExecutable(t)
	npmrcPath := filepath.Join(t.TempDir(), "install.npmrc")
	require.NoError(t, os.WriteFile(npmrcPath, []byte("//registry.example.com/:_authToken=secret\n"), 0600))
	userConfigPathCapture := filepath.Join(t.TempDir(), "userconfig-path")
	userConfigContentsCapture := filepath.Join(t.TempDir(), "userconfig-contents")
	t.Setenv("NPM_USERCONFIG_PATH_CAPTURE_FILE", userConfigPathCapture)
	t.Setenv("NPM_USERCONFIG_CONTENTS_CAPTURE_FILE", userConfigContentsCapture)
	t.Setenv("NPM_CONFIG_USERCONFIG", filepath.Join(t.TempDir(), "ambient-userconfig"))

	_, err := Package(
		[]byte(`{"version":1,"errors":[],"types":[],"services":[],"extensions":{}}`),
		Params{
			PackageName:              "@palantir/test-api",
			Version:                  "1.2.3",
			NpmUserConfigPath:        npmrcPath,
			GenerateThrowingServices: true,
		},
		t.TempDir(),
		os.Stdout,
	)
	require.NoError(t, err)
	pathContents, err := os.ReadFile(userConfigPathCapture)
	require.NoError(t, err)
	assert.Equal(t, npmrcPath+"\n", string(pathContents))
	configContents, err := os.ReadFile(userConfigContentsCapture)
	require.NoError(t, err)
	assert.Equal(t, "//registry.example.com/:_authToken=secret\n", string(configContents))
}

type npmCall struct {
	dir  string
	args string
}

func setupFakeNpmExecutable(t *testing.T) (string, string) {
	t.Helper()
	binDir := t.TempDir()
	callsFile := filepath.Join(t.TempDir(), "calls")
	packageJSONCapturePath := filepath.Join(t.TempDir(), "package.json")
	script := `#!/bin/sh
set -eu
printf '%s|%s\n' "$PWD" "$*" >> "$NPM_CALLS_FILE"
cp package.json "$PACKAGE_JSON_CAPTURE_FILE"
if [ -n "${NPM_USERCONFIG_PATH_CAPTURE_FILE:-}" ]; then
  printf '%s\n' "$NPM_CONFIG_USERCONFIG" > "$NPM_USERCONFIG_PATH_CAPTURE_FILE"
  cp "$NPM_CONFIG_USERCONFIG" "$NPM_USERCONFIG_CONTENTS_CAPTURE_FILE"
fi
if [ "${1:-}" = "pack" ]; then
  touch "palantir-test-api-1.2.3.tgz"
  printf '%s\n' "$NPM_PACK_JSON"
fi
`
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "npm"), []byte(script), 0700))
	t.Setenv("NPM_CALLS_FILE", callsFile)
	t.Setenv("PACKAGE_JSON_CAPTURE_FILE", packageJSONCapturePath)
	t.Setenv("NPM_PACK_JSON", `[{"filename":"palantir-test-api-1.2.3.tgz","files":[{"path":"package.json"},{"path":"index.js"}]}]`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return callsFile, packageJSONCapturePath
}

func readNpmCalls(t *testing.T, path string) []npmCall {
	t.Helper()
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	var calls []npmCall
	for line := range strings.SplitSeq(strings.TrimSpace(string(contents)), "\n") {
		parts := strings.SplitN(line, "|", 2)
		require.Len(t, parts, 2)
		calls = append(calls, npmCall{dir: parts[0], args: parts[1]})
	}
	return calls
}
