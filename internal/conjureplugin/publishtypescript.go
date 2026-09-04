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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/palantir/distgo/distgo"
	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/pkg/errors"
)

// PublishTypeScriptOptions configures TypeScript npm publishing.
type PublishTypeScriptOptions struct {
	PublishRegistry string
	InstallRegistry string
	NpmUsername     string
	NpmPassword     string
	NpmToken        string
}

// PreparedTypeScriptPublish holds npm packages to publish.
type PreparedTypeScriptPublish struct {
	packages        []TypeScriptPackage
	publishRegistry string
	publishNpmrc    string
	cleanup         func()
}

// Cleanup removes the temporary npm configuration and package output.
func (p PreparedTypeScriptPublish) Cleanup() {
	if p.cleanup != nil {
		p.cleanup()
	}
}

// PrepareTypeScriptPublish resolves npm registry auth and packages every TypeScript client into an npm tarball, without publishing anything.
func PrepareTypeScriptPublish(inputs []TypeScriptPackageInput, opts PublishTypeScriptOptions, stdout, stderr io.Writer) (PreparedTypeScriptPublish, error) {
	return prepareTypeScriptPublish(inputs, opts, stdout, stderr, typescript.Package, typescript.ResolveNpmConfig)
}

// PublishPreparedTypeScript publishes prepared npm packages.
func PublishPreparedTypeScript(prepared PreparedTypeScriptPublish, dryRun bool, stdout io.Writer) error {
	return publishPreparedTypeScript(prepared, dryRun, stdout, typescript.PublishPackage)
}

type npmConfigResolver func(typescript.NpmConfigOptions) (typescript.NpmConfig, error)
type typeScriptPackagePublisher func(packagePath, registry, npmUserConfigPath string, stdout io.Writer) error

func prepareTypeScriptPublish(
	inputs []TypeScriptPackageInput,
	opts PublishTypeScriptOptions,
	stdout, stderr io.Writer,
	packager typeScriptPackager,
	resolveNpmConfig npmConfigResolver,
) (PreparedTypeScriptPublish, error) {
	packageNames := typeScriptPackageNames(inputs)
	if len(packageNames) == 0 {
		return PreparedTypeScriptPublish{}, nil
	}

	npmConfig, err := resolveNpmConfig(typescript.NpmConfigOptions{
		PackageNames:    packageNames,
		PublishRegistry: opts.PublishRegistry,
		InstallRegistry: opts.InstallRegistry,
		Username:        opts.NpmUsername,
		Password:        opts.NpmPassword,
		Token:           opts.NpmToken,
	})
	if err != nil {
		return PreparedTypeScriptPublish{}, err
	}

	installNpmrcPath, publishNpmrcPath, cleanupNpmrc, err := writeNpmrcConfig(npmConfig)
	if err != nil {
		return PreparedTypeScriptPublish{}, err
	}

	outputDir, err := os.MkdirTemp("", "conjure-typescript-publish-")
	if err != nil {
		cleanupNpmrc()
		return PreparedTypeScriptPublish{}, errors.Wrap(err, "failed to create temporary TypeScript package directory")
	}
	cleanup := func() {
		cleanupNpmrc()
		_ = os.RemoveAll(outputDir)
	}

	packages, err := packageTypeScriptInputs(inputs, outputDir, installNpmrcPath, stdout, stderr, packager)
	if err != nil {
		cleanup()
		return PreparedTypeScriptPublish{}, err
	}

	return PreparedTypeScriptPublish{
		packages:        packages,
		publishRegistry: npmConfig.PublishRegistry,
		publishNpmrc:    publishNpmrcPath,
		cleanup:         cleanup,
	}, nil
}

func publishPreparedTypeScript(
	prepared PreparedTypeScriptPublish,
	dryRun bool,
	stdout io.Writer,
	publishPackage typeScriptPackagePublisher,
) error {
	for _, pkg := range prepared.packages {
		distgo.PrintlnOrDryRunPrintln(stdout, fmt.Sprintf("Publishing npm package %s to %s\n", pkg.Path, prepared.publishRegistry), dryRun)
		if dryRun {
			continue
		}
		if err := publishPackage(pkg.Path, prepared.publishRegistry, prepared.publishNpmrc, stdout); err != nil {
			return errors.Wrapf(err, "failed to publish TypeScript client for project %q", pkg.ProjectName)
		}
	}
	return nil
}

func typeScriptPackageNames(inputs []TypeScriptPackageInput) []string {
	var result []string
	for _, input := range inputs {
		result = append(result, input.Config.PackageName)
	}
	return result
}

func writeNpmrcConfig(config typescript.NpmConfig) (string, string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "conjure-typescript-npm-config-")
	if err != nil {
		return "", "", nil, errors.Wrap(err, "failed to create temporary npm configuration directory")
	}
	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}
	installPath := filepath.Join(tmpDir, "install.npmrc")
	publishPath := filepath.Join(tmpDir, "publish.npmrc")
	if err := os.WriteFile(installPath, []byte(config.InstallNpmrc), 0600); err != nil {
		cleanup()
		return "", "", nil, errors.Wrap(err, "failed to write temporary install npmrc")
	}
	if err := os.WriteFile(publishPath, []byte(config.PublishNpmrc), 0600); err != nil {
		cleanup()
		return "", "", nil, errors.Wrap(err, "failed to write temporary publish npmrc")
	}
	return installPath, publishPath, cleanup, nil
}
