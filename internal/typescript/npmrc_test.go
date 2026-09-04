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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveNpmConfigDirectTokenAndDistinctInstallRegistry(t *testing.T) {
	config, err := ResolveNpmConfig(NpmConfigOptions{
		PackageNames:    []string{"@example/api", "unscoped"},
		PublishRegistry: "https://registry.example.com/api/npm/release///",
		InstallRegistry: "https://registry.example.com/api/npm/all/",
		Token:           "secret-token",
	})
	require.NoError(t, err)

	assert.Equal(t, "https://registry.example.com/api/npm/release", config.PublishRegistry)
	assert.Equal(t, "registry=https://registry.example.com/api/npm/all/\n@example:registry=https://registry.example.com/api/npm/all/\n", config.InstallNpmrc)
	assert.Equal(t, "registry=https://registry.example.com/api/npm/release/\n@example:registry=https://registry.example.com/api/npm/release/\n//registry.example.com/api/npm/release/:_authToken=secret-token\n", config.PublishNpmrc)
	assert.NotContains(t, config.InstallNpmrc, "secret-token")
}

func TestResolveNpmConfigRejectsInvalidInputs(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		opts      NpmConfigOptions
		wantError string
	}{
		{
			name:      "missing publish registry",
			opts:      NpmConfigOptions{},
			wantError: "npm publish registry",
		},
		{
			name: "partial credentials",
			opts: NpmConfigOptions{
				PublishRegistry: "https://registry.example.com",
				Username:        "user",
			},
			wantError: "username and password must be specified together",
		},
		{
			name: "token and credentials",
			opts: NpmConfigOptions{
				PublishRegistry: "https://registry.example.com",
				Username:        "user",
				Password:        "password",
				Token:           "token",
			},
			wantError: "not both",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ResolveNpmConfig(testCase.opts)
			require.ErrorContains(t, err, testCase.wantError)
		})
	}
}

func TestValidateNpmConfigRejectsInvalidInputs(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		opts      NpmConfigOptions
		wantError string
	}{
		{
			name:      "missing publish registry",
			opts:      NpmConfigOptions{},
			wantError: "npm publish registry",
		},
		{
			name: "invalid install registry",
			opts: NpmConfigOptions{
				PublishRegistry: "https://registry.example.com",
				InstallRegistry: "not-a-url",
			},
			wantError: "npm install registry",
		},
		{
			name: "partial credentials",
			opts: NpmConfigOptions{
				PublishRegistry: "https://registry.example.com",
				Username:        "user",
			},
			wantError: "username and password must be specified together",
		},
		{
			name: "token and credentials",
			opts: NpmConfigOptions{
				PublishRegistry: "https://registry.example.com",
				Username:        "user",
				Password:        "password",
				Token:           "token",
			},
			wantError: "not both",
		},
		{
			name: "invalid package scope",
			opts: NpmConfigOptions{
				PublishRegistry: "https://registry.example.com",
				PackageNames:    []string{"@/bad-scope"},
			},
			wantError: "invalid scope",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.ErrorContains(t, ValidateNpmConfig(testCase.opts), testCase.wantError)
		})
	}
}
