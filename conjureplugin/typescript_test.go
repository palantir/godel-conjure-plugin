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

package conjureplugin

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palantir/distgo/pkg/git"
	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/palantir/godel-conjure-plugin/v7/ir-gen-cli-bundler/conjuretypescriptcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpmPackageVersion(t *testing.T) {
	t.Run("git scheme returns the git version unchanged", func(t *testing.T) {
		got, err := npmPackageVersion(NpmVersionSchemeGit, "0.593.0-23-gf643b56")
		require.NoError(t, err)
		assert.Equal(t, "0.593.0-23-gf643b56", got)
	})

	t.Run("empty scheme defaults to git", func(t *testing.T) {
		got, err := npmPackageVersion("", "1.2.3")
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", got)
	})

	t.Run("generator-major scheme prefixes generator major and a zero", func(t *testing.T) {
		got, err := npmPackageVersion(NpmVersionSchemeGeneratorMajor, "0.604.0")
		require.NoError(t, err)
		major, _, _ := strings.Cut(conjuretypescriptcli.Version, ".")
		assert.Equal(t, major+"00.604.0", got)
	})

	t.Run("unknown scheme is an error", func(t *testing.T) {
		_, err := npmPackageVersion("unknown", "1.2.3")
		require.Error(t, err)
	})
}

func TestPackageTypeScript(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := t.TempDir()
	npmUserConfigPath := filepath.Join(t.TempDir(), "install.npmrc")
	productDependencies := []byte(`[{"product-group":"com.palantir","product-name":"dependency"}]`)

	conjureProjectParams := ConjureProjectParams{
		{
			ProjectName: "api",
			IRProvider: staticTypeScriptIRProvider(
				`{"version":1,"extensions":{"recommended-product-dependencies":[{"product-group":"com.palantir","product-name":"dependency"}]}}`,
			),
			TypeScript: &TypeScriptParam{
				PackageName:                 "@palantir/api",
				NpmVersionScheme:            NpmVersionSchemeGeneratorMajor,
				FlavorizedAliases:           true,
				NodeCompatibleModules:       true,
				ReadonlyInterfaces:          true,
				GenerateThrowingServices:    true,
				GenerateNonThrowingServices: true,
			},
		},
		{ProjectName: "not-typescript", IRProvider: staticTypeScriptIRProvider(`{}`)},
	}
	packageOpts := PackageTypeScriptOptions{
		OutputDir:         outputDir,
		PackageVersion:    "9.8.7",
		NpmUserConfigPath: npmUserConfigPath,
	}
	var packageCalls int
	packager := func(irBytes []byte, params typescript.Params, packageOutputDir string, _ io.Writer) (string, error) {
		packageCalls++
		assert.JSONEq(t, `{"version":1,"extensions":{"recommended-product-dependencies":[{"product-group":"com.palantir","product-name":"dependency"}]}}`, string(irBytes))
		assert.Equal(t, typescript.Params{
			PackageName:                 "@palantir/api",
			Version:                     "9.8.7",
			ProductDependencies:         productDependencies,
			NpmUserConfigPath:           npmUserConfigPath,
			FlavorizedAliases:           true,
			NodeCompatibleModules:       true,
			ReadonlyInterfaces:          true,
			GenerateThrowingServices:    true,
			GenerateNonThrowingServices: true,
		}, params)
		assert.Equal(t, filepath.Join(outputDir, "api", "9.8.7", "npm"), packageOutputDir)
		packagePath := filepath.Join(packageOutputDir, "api-9.8.7.tgz")
		require.NoError(t, os.MkdirAll(packageOutputDir, 0755))
		require.NoError(t, os.WriteFile(packagePath, []byte("package"), 0600))
		return packagePath, nil
	}
	packages, err := packageTypeScript(conjureProjectParams, projectDir, packageOpts, io.Discard, io.Discard, packager, nil, "")
	require.NoError(t, err)
	assert.Equal(t, 1, packageCalls)
	expectedPackages := []TypeScriptPackage{{
		ProjectName: "api",
		PackageName: "@palantir/api",
		Version:     "9.8.7",
		Path:        filepath.Join(outputDir, "api", "9.8.7", "npm", "api-9.8.7.tgz"),
	}}
	require.Equal(t, expectedPackages, packages)
}

func TestPackageTypeScript_DefaultsToTemporaryOutputDir(t *testing.T) {
	projectDir := t.TempDir()
	conjureProjectParams := ConjureProjectParams{{
		ProjectName: "api",
		IRProvider:  staticTypeScriptIRProvider(`{"version":1}`),
		TypeScript:  &TypeScriptParam{PackageName: "@palantir/api"},
	}}
	var gotOutputDir string
	_, err := packageTypeScript(
		conjureProjectParams,
		projectDir,
		PackageTypeScriptOptions{PackageVersion: "1.2.3"},
		io.Discard,
		io.Discard,
		func(_ []byte, _ typescript.Params, packageOutputDir string, _ io.Writer) (string, error) {
			gotOutputDir = packageOutputDir
			return filepath.Join(packageOutputDir, "palantir-api-1.2.3.tgz"), nil
		},
		nil,
		"",
	)
	require.NoError(t, err)
	root := filepath.Dir(filepath.Dir(filepath.Dir(gotOutputDir)))
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	assert.False(t, strings.HasPrefix(gotOutputDir, projectDir), "default output dir %q should not be rooted under the project directory %q", gotOutputDir, projectDir)
	assert.Equal(t, filepath.Join(root, "api", "1.2.3", "npm"), gotOutputDir)
	info, err := os.Stat(root)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestPackageTypeScript_RejectsUnspecifiedGitVersion(t *testing.T) {
	params := ConjureProjectParams{{
		ProjectName: "api",
		IRProvider:  staticTypeScriptIRProvider(`{"version":1}`),
		TypeScript:  &TypeScriptParam{PackageName: "@palantir/api"},
	}}
	var packageCalls int
	packager := func(_ []byte, _ typescript.Params, _ string, _ io.Writer) (string, error) {
		packageCalls++
		return "", nil
	}
	versioner := func(string) (string, error) {
		return git.Unspecified, nil
	}

	_, err := packageTypeScriptWithVersioner(params, t.TempDir(), PackageTypeScriptOptions{}, io.Discard, io.Discard, packager, nil, "", versioner)
	require.EqualError(t, err, "unable to determine project version from Git")
	assert.Zero(t, packageCalls)
}

func TestProjectPackageOutputDir_RejectsUnsafeVersion(t *testing.T) {
	for _, version := range []string{"", ".", "..", "../outside", "nested/version"} {
		t.Run(version, func(t *testing.T) {
			_, err := projectPackageOutputDir(t.TempDir(), "api", version)
			require.ErrorContains(t, err, "invalid npm package version")
		})
	}
}

func TestPackageTypeScript_ProductDependencies(t *testing.T) {
	for _, tc := range []struct {
		name string
		ir   string
		want []byte
	}{
		{
			name: "extension embedded in IR",
			ir:   `{"version":1,"extensions":{"recommended-product-dependencies":[{"source":"ir"}]}}`,
			want: []byte(`[{"source":"ir"}]`),
		},
		{
			name: "explicitly empty extension embedded in IR",
			ir:   `{"version":1,"extensions":{"recommended-product-dependencies":[]}}`,
			want: []byte(`[]`),
		},
		{
			name: "no extension",
			ir:   `{"version":1,"extensions":{}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []byte
			conjureProjectParams := ConjureProjectParams{{
				ProjectName: "api",
				IRProvider:  staticTypeScriptIRProvider(tc.ir),
				TypeScript: &TypeScriptParam{
					PackageName: "@palantir/api",
				},
			}}
			packager := func(_ []byte, params typescript.Params, _ string, _ io.Writer) (string, error) {
				got = params.ProductDependencies
				return "api-1.2.3.tgz", nil
			}
			_, err := packageTypeScript(conjureProjectParams, t.TempDir(), PackageTypeScriptOptions{PackageVersion: "1.2.3"}, io.Discard, io.Discard, packager, nil, "")
			require.NoError(t, err)
			if tc.want == nil {
				assert.Nil(t, got)
			} else {
				assert.JSONEq(t, string(tc.want), string(got))
			}
		})
	}
}

func TestPackageTypeScript_ResolvesProductDependenciesUsingGitVersion(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		cliGroupID string
		wantGroup  string
	}{
		{
			name:      "project group",
			wantGroup: "com.palantir.project",
		},
		{
			name:       "CLI group overrides project group",
			cliGroupID: "com.palantir.cli",
			wantGroup:  "com.palantir.cli",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resolvedDependencies := []byte(`[{"product-group":"com.palantir","product-name":"api","minimum-version":"0.500.0","maximum-version":"0.x.x"}]`)
			extensionsProvider := func(irBytes []byte, groupID, project, version string) (map[string]any, error) {
				assert.JSONEq(t, `{"version":1}`, string(irBytes))
				assert.Equal(t, testCase.wantGroup, groupID)
				assert.Equal(t, "api", project)
				assert.Equal(t, "0.500.0", version)
				return map[string]any{
					"recommended-product-dependencies": []any{map[string]any{
						"product-group":   "com.palantir",
						"product-name":    "api",
						"minimum-version": "0.500.0",
						"maximum-version": "0.x.x",
					}},
				}, nil
			}

			var packagedIR []byte
			var packagedParams typescript.Params
			conjureProjectParams := ConjureProjectParams{{
				ProjectName: "api",
				GroupID:     "com.palantir.project",
				IRProvider:  staticTypeScriptIRProvider(`{"version":1}`),
				TypeScript: &TypeScriptParam{
					PackageName:      "@palantir/api",
					NpmVersionScheme: NpmVersionSchemeGeneratorMajor,
				},
			}}
			packager := func(irBytes []byte, params typescript.Params, _ string, _ io.Writer) (string, error) {
				packagedIR = irBytes
				packagedParams = params
				return "api-500.500.0.tgz", nil
			}
			versioner := func(string) (string, error) {
				return "0.500.0", nil
			}
			packages, err := packageTypeScriptWithVersioner(conjureProjectParams, t.TempDir(), PackageTypeScriptOptions{PackageVersion: "500.500.0"}, io.Discard, io.Discard, packager, extensionsProvider, testCase.cliGroupID, versioner)
			require.NoError(t, err)
			require.Len(t, packages, 1)
			assert.Equal(t, "500.500.0", packages[0].Version)
			assert.Equal(t, "500.500.0", packagedParams.Version)
			assert.JSONEq(t, string(resolvedDependencies), string(packagedParams.ProductDependencies))
			assert.JSONEq(t, `{"version":1,"extensions":{"recommended-product-dependencies":[{"product-group":"com.palantir","product-name":"api","minimum-version":"0.500.0","maximum-version":"0.x.x"}]}}`, string(packagedIR))
		})
	}
}

func TestPackageTypeScript_RejectsInvalidProductDependenciesInIR(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		ir        string
		wantError string
	}{
		{
			name:      "malformed IR JSON",
			ir:        `{`,
			wantError: `failed to parse Conjure IR as JSON`,
		},
		{
			name:      "product dependencies are not an array",
			ir:        `{"version":1,"extensions":{"recommended-product-dependencies":{"not":"an array"}}}`,
			wantError: `recommended-product-dependencies must be an array`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			params := ConjureProjectParams{{
				ProjectName: "api",
				IRProvider:  staticTypeScriptIRProvider(testCase.ir),
				TypeScript:  &TypeScriptParam{PackageName: "@palantir/api"},
			}}
			_, err := PackageTypeScript(params, t.TempDir(), PackageTypeScriptOptions{PackageVersion: "1.2.3"}, io.Discard, io.Discard, nil, "")
			require.ErrorContains(t, err, `failed to read product dependencies from IR for project "api"`)
			require.ErrorContains(t, err, testCase.wantError)
		})
	}
}

type staticTypeScriptIRProvider string

func (p staticTypeScriptIRProvider) IRBytes() ([]byte, error) { return []byte(p), nil }
func (p staticTypeScriptIRProvider) GeneratedFromYAML() bool  { return false }
