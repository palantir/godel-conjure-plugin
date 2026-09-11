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
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareTypeScriptPublishPackagesEachInputOnce(t *testing.T) {
	inputs := []TypeScriptPackageInput{
		{ProjectName: "api", IR: `{}`, PackageVersion: "1.2.3", Config: TypeScriptParam{PackageName: "@palantir/api"}},
		{ProjectName: "other", IR: `{}`, PackageVersion: "1.2.3", Config: TypeScriptParam{PackageName: "unscoped"}},
	}
	var installNpmrc, packageOutputRoot string
	var packageCalls int

	packager := func(irBytes []byte, gotParams typescript.Params, outputDir string, _ io.Writer) (string, error) {
		packageCalls++
		assert.JSONEq(t, `{}`, string(irBytes))
		assert.Equal(t, "1.2.3", gotParams.Version)
		installNpmrc = gotParams.NpmUserConfigPath
		contents, err := os.ReadFile(installNpmrc)
		require.NoError(t, err)
		assert.Equal(t, "install config\n", string(contents))
		info, err := os.Stat(installNpmrc)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		packageOutputRoot = filepath.Dir(filepath.Dir(filepath.Dir(outputDir)))
		var packageFile string
		switch gotParams.PackageName {
		case "@palantir/api":
			assert.Equal(t, filepath.Join(packageOutputRoot, "api", "1.2.3", "npm"), outputDir)
			packageFile = "api-1.2.3.tgz"
		case "unscoped":
			assert.Equal(t, filepath.Join(packageOutputRoot, "other", "1.2.3", "npm"), outputDir)
			packageFile = "other-1.2.3.tgz"
		default:
			t.Fatalf("unexpected package name %q", gotParams.PackageName)
		}
		require.NoError(t, os.MkdirAll(outputDir, 0755))
		packagePath := filepath.Join(outputDir, packageFile)
		require.NoError(t, os.WriteFile(packagePath, []byte("package"), 0600))
		return packagePath, nil
	}

	resolveNPMConfig := func(opts typescript.NpmConfigOptions) (typescript.NpmConfig, error) {
		assert.Equal(t, []string{"@palantir/api", "unscoped"}, opts.PackageNames)
		assert.Equal(t, "https://publish.example.com/", opts.PublishRegistry)
		assert.Equal(t, "https://install.example.com/", opts.InstallRegistry)
		assert.Equal(t, "user", opts.Username)
		assert.Equal(t, "password", opts.Password)
		return typescript.NpmConfig{
			PublishRegistry: "https://publish.example.com",
			InstallNpmrc:    "install config\n",
			PublishNpmrc:    "publish config\n",
		}, nil
	}

	prepared, err := prepareTypeScriptPublish(inputs, PublishTypeScriptOptions{
		PublishRegistry: "https://publish.example.com/",
		InstallRegistry: "https://install.example.com/",
		NpmUsername:     "user",
		NpmPassword:     "password",
	}, io.Discard, io.Discard, packager, resolveNPMConfig)
	require.NoError(t, err)
	assert.Equal(t, 2, packageCalls)
	assert.Equal(t, "https://publish.example.com", prepared.publishRegistry)
	require.Len(t, prepared.packages, 2)
	assert.Equal(t, "api", prepared.packages[0].ProjectName)
	assert.Equal(t, "other", prepared.packages[1].ProjectName)
	publishNpmrcContents, err := os.ReadFile(prepared.publishNpmrc)
	require.NoError(t, err)
	assert.Equal(t, "publish config\n", string(publishNpmrcContents))

	prepared.Cleanup()
	_, err = os.Stat(packageOutputRoot)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(installNpmrc)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(prepared.publishNpmrc)
	assert.True(t, os.IsNotExist(err))
}

func TestPrepareTypeScriptPublishSkipsWhenNoTypeScriptProjects(t *testing.T) {
	prepared, err := prepareTypeScriptPublish(nil, PublishTypeScriptOptions{}, io.Discard, io.Discard,
		func([]byte, typescript.Params, string, io.Writer) (string, error) {
			t.Fatal("packager must not run")
			return "", nil
		},
		func(typescript.NpmConfigOptions) (typescript.NpmConfig, error) {
			t.Fatal("npm config resolver must not run")
			return typescript.NpmConfig{}, nil
		})
	require.NoError(t, err)
	assert.Equal(t, PreparedTypeScriptPublish{}, prepared)
	prepared.Cleanup()
}

func TestPrepareTypeScriptPublishCleansUpOnPackagingFailure(t *testing.T) {
	inputs := []TypeScriptPackageInput{
		{ProjectName: "api", IR: `{}`, PackageVersion: "1.2.3", Config: TypeScriptParam{PackageName: "@palantir/api"}},
	}
	var installNpmrc string
	_, err := prepareTypeScriptPublish(inputs, PublishTypeScriptOptions{PublishRegistry: "https://publish.example.com/"}, io.Discard, io.Discard,
		func(_ []byte, gotParams typescript.Params, _ string, _ io.Writer) (string, error) {
			installNpmrc = gotParams.NpmUserConfigPath
			return "", errors.New("build failed")
		},
		func(typescript.NpmConfigOptions) (typescript.NpmConfig, error) {
			return typescript.NpmConfig{
				PublishRegistry: "https://publish.example.com",
				InstallNpmrc:    "install config\n",
				PublishNpmrc:    "publish config\n",
			}, nil
		})
	require.Error(t, err)
	_, statErr := os.Stat(installNpmrc)
	assert.True(t, os.IsNotExist(statErr), "temporary npm configuration should be cleaned up on packaging failure")
}

func TestPublishPreparedTypeScriptPublishesEachPackage(t *testing.T) {
	packagePath := filepath.Join(t.TempDir(), "api-1.2.3.tgz")
	require.NoError(t, os.WriteFile(packagePath, []byte("package"), 0600))
	npmrcPath := filepath.Join(t.TempDir(), "publish.npmrc")
	require.NoError(t, os.WriteFile(npmrcPath, []byte("publish config\n"), 0600))

	prepared := PreparedTypeScriptPublish{
		packages:        []TypeScriptPackage{{ProjectName: "api", PackageName: "@palantir/api", Version: "1.2.3", Path: packagePath}},
		publishRegistry: "https://publish.example.com",
		publishNpmrc:    npmrcPath,
	}

	var published []string
	var output bytes.Buffer
	err := publishPreparedTypeScript(prepared, false, &output,
		func(gotPackagePath, registry, npmUserConfigPath string, _ io.Writer) error {
			assert.Equal(t, packagePath, gotPackagePath)
			assert.Equal(t, "https://publish.example.com", registry)
			assert.Equal(t, npmrcPath, npmUserConfigPath)
			published = append(published, gotPackagePath)
			return nil
		})
	require.NoError(t, err)
	assert.Equal(t, []string{packagePath}, published)
	assert.Contains(t, output.String(), packagePath)
}

func TestPublishPreparedTypeScriptDryRunSkipsPublishing(t *testing.T) {
	prepared := PreparedTypeScriptPublish{
		packages:        []TypeScriptPackage{{ProjectName: "api", PackageName: "@palantir/api", Version: "1.2.3", Path: "api.tgz"}},
		publishRegistry: "https://publish.example.com",
	}
	var output bytes.Buffer
	err := publishPreparedTypeScript(prepared, true, &output,
		func(string, string, string, io.Writer) error {
			t.Fatal("publisher must not run during a dry run")
			return nil
		})
	require.NoError(t, err)
	assert.Contains(t, output.String(), "[DRY RUN] Publishing npm package")
}
