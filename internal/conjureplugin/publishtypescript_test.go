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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishTypeScriptPublishesExactPackagedTarball(t *testing.T) {
	params := ConjureProjectParams{
		{
			ProjectName: "api",
			IRProvider:  staticTypeScriptIRProvider(`{"version":1}`),
			TypeScript:  &TypeScriptParam{PackageName: "@palantir/api"},
		},
		{
			ProjectName: "other",
			IRProvider:  staticTypeScriptIRProvider(`{"version":1}`),
			TypeScript:  &TypeScriptParam{PackageName: "@palantir/other"},
		},
	}
	packagePath := filepath.Join(t.TempDir(), "palantir-api-1.2.3.tgz")
	require.NoError(t, os.WriteFile(packagePath, []byte("exact tarball"), 0600))
	extensionsProvider := extensionsprovider.ExtensionsProvider(func([]byte, string, string, string) (map[string]any, error) {
		return nil, nil
	})

	var installNpmrcPath string
	var publishNpmrcPath string
	var output bytes.Buffer
	err := publishTypeScript(params, t.TempDir(), PublishTypeScriptOptions{
		PublishRegistry: "https://registry.example.com/api/npm/internal-release/",
		InstallRegistry: "https://registry.example.com/api/npm/all-npm/",
		NpmToken:        "super-secret-token",
		OutputDir:       "custom-output",
		PackageVersion:  "1.2.3",
	}, &output, io.Discard, extensionsProvider, "com.palantir.cli",
		func(gotParams ConjureProjectParams, _ string, opts PackageTypeScriptOptions, _ io.Writer, _ io.Writer,
			gotExtensionsProvider extensionsprovider.ExtensionsProvider, cliGroupID string) ([]TypeScriptPackage, error) {
			assert.Equal(t, params, gotParams)
			assert.Equal(t, "custom-output", opts.OutputDir)
			assert.Equal(t, "1.2.3", opts.PackageVersion)
			assert.NotNil(t, gotExtensionsProvider)
			assert.Equal(t, "com.palantir.cli", cliGroupID)
			installNpmrcPath = opts.NpmUserConfigPath
			contents, err := os.ReadFile(installNpmrcPath)
			require.NoError(t, err)
			assert.Contains(t, string(contents), "registry=https://registry.example.com/api/npm/all-npm/")
			assert.Contains(t, string(contents), "@palantir:registry=https://registry.example.com/api/npm/all-npm/")
			// The install registry is distinct from the publish registry, so it must not receive the publish token:
			// doing so would risk disclosing it to an unrelated registry.
			assert.NotContains(t, string(contents), "super-secret-token")
			info, err := os.Stat(installNpmrcPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
			return []TypeScriptPackage{{ProjectName: "api", PackageName: "@palantir/api", Version: "1.2.3", Path: packagePath}}, nil
		},
		func(gotPackagePath, registry, npmUserConfigPath string, _ io.Writer) error {
			assert.Equal(t, packagePath, gotPackagePath)
			assert.Equal(t, "https://registry.example.com/api/npm/internal-release", registry)
			publishNpmrcPath = npmUserConfigPath
			contents, err := os.ReadFile(npmUserConfigPath)
			require.NoError(t, err)
			assert.Contains(t, string(contents), "registry=https://registry.example.com/api/npm/internal-release/")
			assert.Contains(t, string(contents), "@palantir:registry=https://registry.example.com/api/npm/internal-release/")
			assert.Contains(t, string(contents), "//registry.example.com/api/npm/internal-release/:_authToken=super-secret-token")
			return nil
		})
	require.NoError(t, err)
	assert.NotContains(t, output.String(), "super-secret-token")
	assert.Contains(t, output.String(), packagePath)
	_, err = os.Stat(installNpmrcPath)
	assert.True(t, os.IsNotExist(err), "temporary install npmrc should be removed")
	_, err = os.Stat(publishNpmrcPath)
	assert.True(t, os.IsNotExist(err), "temporary publish npmrc should be removed")
}

func TestPublishTypeScriptSkipsProjectsWithoutTypeScript(t *testing.T) {
	err := publishTypeScript(ConjureProjectParams{{ProjectName: "go-only"}}, t.TempDir(), PublishTypeScriptOptions{}, io.Discard, io.Discard,
		nil, "",
		func(ConjureProjectParams, string, PackageTypeScriptOptions, io.Writer, io.Writer, extensionsprovider.ExtensionsProvider, string) ([]TypeScriptPackage, error) {
			t.Fatal("package task must not run")
			return nil, nil
		},
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run")
			return nil
		})
	require.NoError(t, err)
}

func TestNpmPublishProjectsUsesTypeScriptOptInIndependentlyOfIRPublish(t *testing.T) {
	params := ConjureProjectParams{
		{
			ProjectName: "typescript",
			Publish:     false,
			TypeScript:  &TypeScriptParam{PackageName: "@palantir/typescript"},
		},
		{
			ProjectName: "ir-only",
			Publish:     true,
		},
	}

	assert.Equal(t, []typescript.NpmPublishProject{{
		ProjectIndex: 0,
		ProjectName:  "typescript",
		PackageName:  "@palantir/typescript",
	}}, npmPublishProjects(params))
}

func TestPublishTypeScriptRequiresPublishRegistry(t *testing.T) {
	err := publishTypeScript(ConjureProjectParams{{
		ProjectName: "api",
		TypeScript: &TypeScriptParam{
			PackageName:          "@palantir/api",
			NpmPublisherProvider: typescript.NpmPublisherProviderNpmrc,
		},
	}}, t.TempDir(), PublishTypeScriptOptions{}, io.Discard, io.Discard, nil, "",
		func(ConjureProjectParams, string, PackageTypeScriptOptions, io.Writer, io.Writer, extensionsprovider.ExtensionsProvider, string) ([]TypeScriptPackage, error) {
			t.Fatal("package task must not run")
			return nil, nil
		},
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run")
			return nil
		})
	require.ErrorContains(t, err, "npm publish registry")
}

func TestPublishTypeScriptUsesNpmrcFileVerbatim(t *testing.T) {
	npmrcFile := filepath.Join(t.TempDir(), "external.npmrc")
	require.NoError(t, os.WriteFile(npmrcFile, []byte("registry=https://registry.example.com/api/npm/internal-release/\n"), 0600))
	packagePath := filepath.Join(t.TempDir(), "palantir-api-1.2.3.tgz")
	require.NoError(t, os.WriteFile(packagePath, []byte("tarball"), 0600))

	var gotInstallPath, gotPublishPath string
	err := publishTypeScript(ConjureProjectParams{{
		ProjectName: "api",
		TypeScript: &TypeScriptParam{
			PackageName:          "@palantir/api",
			NpmPublisherProvider: typescript.NpmPublisherProviderNpmrc,
		},
	}}, t.TempDir(), PublishTypeScriptOptions{
		PublishRegistry: "https://registry.example.com/api/npm/internal-release/",
		NpmrcFile:       npmrcFile,
		PackageVersion:  "1.2.3",
	}, io.Discard, io.Discard, nil, "",
		func(_ ConjureProjectParams, _ string, opts PackageTypeScriptOptions, _ io.Writer, _ io.Writer, _ extensionsprovider.ExtensionsProvider, _ string) ([]TypeScriptPackage, error) {
			gotInstallPath = opts.NpmUserConfigPath
			return []TypeScriptPackage{{ProjectName: "api", PackageName: "@palantir/api", Version: "1.2.3", Path: packagePath}}, nil
		},
		func(gotPackagePath, registry, npmUserConfigPath string, _ io.Writer) error {
			assert.Equal(t, packagePath, gotPackagePath)
			assert.Equal(t, "https://registry.example.com/api/npm/internal-release", registry)
			gotPublishPath = npmUserConfigPath
			return nil
		})
	require.NoError(t, err)

	absNpmrcFile, err := filepath.Abs(npmrcFile)
	require.NoError(t, err)
	assert.Equal(t, absNpmrcFile, gotInstallPath)
	assert.Equal(t, absNpmrcFile, gotPublishPath)
	// The supplied file must still exist afterward: unlike a generated npmrc, this plugin does not own its lifecycle.
	assert.FileExists(t, npmrcFile)
}

func TestPublishTypeScriptRejectsNpmrcFileWithCredentials(t *testing.T) {
	npmrcFile := filepath.Join(t.TempDir(), "external.npmrc")
	require.NoError(t, os.WriteFile(npmrcFile, []byte("registry=https://registry.example.com/\n"), 0600))

	err := publishTypeScript(ConjureProjectParams{{
		ProjectName: "api",
		TypeScript: &TypeScriptParam{
			PackageName:          "@palantir/api",
			NpmPublisherProvider: typescript.NpmPublisherProviderNpmrc,
		},
	}}, t.TempDir(), PublishTypeScriptOptions{
		PublishRegistry: "https://registry.example.com",
		NpmrcFile:       npmrcFile,
		NpmToken:        "token",
	}, io.Discard, io.Discard, nil, "",
		func(ConjureProjectParams, string, PackageTypeScriptOptions, io.Writer, io.Writer, extensionsprovider.ExtensionsProvider, string) ([]TypeScriptPackage, error) {
			t.Fatal("package task must not run")
			return nil, nil
		},
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run")
			return nil
		})
	require.ErrorContains(t, err, "npm credentials cannot be specified")
}

func TestPublishTypeScriptRejectsNpmrcFileWithInstallRegistry(t *testing.T) {
	npmrcFile := filepath.Join(t.TempDir(), "external.npmrc")
	require.NoError(t, os.WriteFile(npmrcFile, []byte("registry=https://registry.example.com/\n"), 0600))

	err := publishTypeScript(ConjureProjectParams{{
		ProjectName: "api",
		TypeScript: &TypeScriptParam{
			PackageName:          "@palantir/api",
			NpmPublisherProvider: typescript.NpmPublisherProviderNpmrc,
		},
	}}, t.TempDir(), PublishTypeScriptOptions{
		PublishRegistry: "https://registry.example.com",
		InstallRegistry: "https://read.example.com",
		NpmrcFile:       npmrcFile,
	}, io.Discard, io.Discard, nil, "",
		func(ConjureProjectParams, string, PackageTypeScriptOptions, io.Writer, io.Writer, extensionsprovider.ExtensionsProvider, string) ([]TypeScriptPackage, error) {
			t.Fatal("package task must not run")
			return nil, nil
		},
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run")
			return nil
		})
	require.ErrorContains(t, err, "npm install registry cannot be specified")
}

func TestPublishTypeScriptRejectsMissingNpmrcFile(t *testing.T) {
	err := publishTypeScript(ConjureProjectParams{{
		ProjectName: "api",
		TypeScript: &TypeScriptParam{
			PackageName:          "@palantir/api",
			NpmPublisherProvider: typescript.NpmPublisherProviderNpmrc,
		},
	}}, t.TempDir(), PublishTypeScriptOptions{
		PublishRegistry: "https://registry.example.com",
		NpmrcFile:       filepath.Join(t.TempDir(), "does-not-exist.npmrc"),
	}, io.Discard, io.Discard, nil, "",
		func(ConjureProjectParams, string, PackageTypeScriptOptions, io.Writer, io.Writer, extensionsprovider.ExtensionsProvider, string) ([]TypeScriptPackage, error) {
			t.Fatal("package task must not run")
			return nil, nil
		},
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run")
			return nil
		})
	require.ErrorContains(t, err, "npmrc file does not exist")
}

func TestPublishTypeScriptDryRunPackagesWithoutPublishing(t *testing.T) {
	packagePath := filepath.Join(t.TempDir(), "palantir-api-1.2.3.tgz")
	require.NoError(t, os.WriteFile(packagePath, []byte("tarball"), 0600))
	var output bytes.Buffer
	err := publishTypeScript(ConjureProjectParams{{
		ProjectName: "api",
		TypeScript:  &TypeScriptParam{PackageName: "@palantir/api"},
	}}, t.TempDir(), PublishTypeScriptOptions{
		PublishRegistry: "https://registry.example.com",
		NpmToken:        "token",
		DryRun:          true,
	}, &output, io.Discard, nil, "",
		func(ConjureProjectParams, string, PackageTypeScriptOptions, io.Writer, io.Writer, extensionsprovider.ExtensionsProvider, string) ([]TypeScriptPackage, error) {
			return []TypeScriptPackage{{ProjectName: "api", Path: packagePath}}, nil
		},
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run during a dry run")
			return nil
		})
	require.NoError(t, err)
	assert.Contains(t, output.String(), "[DRY RUN] Publishing npm package "+packagePath)
}
