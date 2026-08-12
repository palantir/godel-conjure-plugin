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

	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/pkg/errors"
)

// PublishTypeScriptOptions configures TypeScript packaging and publication. Registry destinations and credentials are
// operation inputs rather than project configuration so release environments can select their own repositories.
type PublishTypeScriptOptions struct {
	PublishRegistry string
	InstallRegistry string
	NpmUsername     string
	NpmPassword     string
	NpmToken        string
	// NpmrcFile supplies npm configuration for projects configured with the "npmrc" publisher provider. The file is
	// used verbatim for both installing dependencies and publishing, and remains owned by the caller.
	NpmrcFile      string
	OutputDir      string
	PackageVersion string
	DryRun         bool
}

// PublishTypeScript packages every configured TypeScript client using PackageTypeScript and publishes each literal
// tarball returned by that operation.
func PublishTypeScript(
	params ConjureProjectParams,
	projectDir string,
	opts PublishTypeScriptOptions,
	stdout io.Writer,
	stderr io.Writer,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string,
) error {
	return publishTypeScript(params, projectDir, opts, stdout, stderr, extensionsProvider, cliGroupID,
		PackageTypeScript, typescript.PublishPackage)
}

type packageTypeScriptTask func(
	params ConjureProjectParams,
	projectDir string,
	opts PackageTypeScriptOptions,
	stdout io.Writer,
	stderr io.Writer,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string,
) ([]TypeScriptPackage, error)

type typeScriptPackagePublisher func(packagePath, registry, npmUserConfigPath string, stdout io.Writer) error

type typeScriptPublication struct {
	pkg              TypeScriptPackage
	publishNpmrcPath string
}

func publishTypeScript(
	params ConjureProjectParams,
	projectDir string,
	opts PublishTypeScriptOptions,
	stdout io.Writer,
	stderr io.Writer,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string,
	packageTask packageTypeScriptTask,
	publisher typeScriptPackagePublisher,
) (rErr error) {
	projects := npmPublishProjects(params)
	if len(projects) == 0 {
		return nil
	}

	plan, err := typescript.PrepareNpmPublishPlan(projects, typescript.NpmPublishPlanConfig{
		PublishRegistry: opts.PublishRegistry,
		InstallRegistry: opts.InstallRegistry,
		Username:        opts.NpmUsername,
		Password:        opts.NpmPassword,
		Token:           opts.NpmToken,
		NpmrcFile:       opts.NpmrcFile,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := plan.Close(); rErr == nil && err != nil {
			rErr = err
		}
	}()

	var publications []typeScriptPublication
	for _, group := range plan.Groups {
		groupParams := make(ConjureProjectParams, 0, len(group.ProjectIndexes))
		for _, projectIndex := range group.ProjectIndexes {
			groupParams = append(groupParams, params[projectIndex])
		}
		packages, packageErr := packageTask(groupParams, projectDir, PackageTypeScriptOptions{
			OutputDir:         opts.OutputDir,
			PackageVersion:    opts.PackageVersion,
			NpmUserConfigPath: group.InstallNpmrcPath,
		}, stdout, stderr, extensionsProvider, cliGroupID)
		if packageErr != nil {
			return packageErr
		}
		for _, pkg := range packages {
			publications = append(publications, typeScriptPublication{pkg: pkg, publishNpmrcPath: group.PublishNpmrcPath})
		}
	}

	for _, publication := range publications {
		if opts.DryRun {
			_, _ = fmt.Fprintf(stdout, "[DRY RUN] Publishing npm package %s to %s\n", publication.pkg.Path, plan.PublishRegistry)
			continue
		}
		_, _ = fmt.Fprintf(stdout, "Publishing npm package %s to %s\n", publication.pkg.Path, plan.PublishRegistry)
		if err := publisher(publication.pkg.Path, plan.PublishRegistry, publication.publishNpmrcPath, stdout); err != nil {
			return errors.Wrapf(err, "failed to publish TypeScript client for project %q", publication.pkg.ProjectName)
		}
	}
	return nil
}

func npmPublishProjects(params ConjureProjectParams) []typescript.NpmPublishProject {
	var projects []typescript.NpmPublishProject
	for index, param := range params {
		// A TypeScript block independently opts the project into npm publication. Publish controls IR publication only.
		if param.TypeScript == nil {
			continue
		}
		projects = append(projects, typescript.NpmPublishProject{
			ProjectIndex:      index,
			ProjectName:       param.ProjectName,
			PackageName:       param.TypeScript.PackageName,
			PublisherProvider: param.TypeScript.NpmPublisherProvider,
		})
	}
	return projects
}
