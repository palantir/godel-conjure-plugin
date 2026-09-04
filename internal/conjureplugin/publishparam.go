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
	"github.com/palantir/distgo/pkg/git"
	gitversioner "github.com/palantir/distgo/projectversioner/git"
	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/pkg/errors"
)

// PublishParam contains the fully resolved inputs for a publish operation.
type PublishParam struct {
	Version    string
	ConjureIR  []ConjureIRPublishParam
	TypeScript []TypeScriptPackageInput
}

// ConjureIRPublishParam contains the fully resolved inputs for publishing one project's Conjure IR.
// IR contains the final extension-enriched JSON as an immutable string.
type ConjureIRPublishParam struct {
	ProjectName string
	IR          string
	GroupID     string
}

// PublishParamOptions contains the inputs used to resolve a [PublishParam] from Conjure project parameters.
type PublishParamOptions struct {
	ProjectDir string
	// ExtensionsProvider enriches each selected project's IR. When nil, no extensions are added.
	ExtensionsProvider extensionsprovider.ExtensionsProvider
	GroupIDOverride    string
}

type publishProjectVersioner func(projectDir string) (string, error)

// NewPublishParam resolves the projects selected for publication into immutable, target-specific inputs.
func NewPublishParam(projects ConjureProjectParams, opts PublishParamOptions) (PublishParam, error) {
	return newPublishParam(projects, opts, gitversioner.New().ProjectVersion)
}

func newPublishParam(projects ConjureProjectParams, opts PublishParamOptions, versioner publishProjectVersioner) (PublishParam, error) {
	var selectedProjects ConjureProjectParams
	for _, project := range projects {
		if !project.Publish && !typeScriptPublishEnabled(project) {
			continue
		}
		selectedProjects = append(selectedProjects, project)
	}
	if len(selectedProjects) == 0 {
		return PublishParam{}, nil
	}

	version, err := versioner(opts.ProjectDir)
	if err != nil {
		return PublishParam{}, err
	}
	if version == git.Unspecified && anyTypeScriptPublishEnabled(selectedProjects) {
		return PublishParam{}, errors.New("unable to determine project version from Git")
	}
	if err := rejectDuplicateTypeScriptPackages(selectedProjects, version); err != nil {
		return PublishParam{}, err
	}

	param := PublishParam{Version: version}
	for _, project := range selectedProjects {
		irBytes, err := project.IRProvider.IRBytes()
		if err != nil {
			return PublishParam{}, err
		}

		groupID := project.GroupID
		if opts.GroupIDOverride != "" {
			groupID = opts.GroupIDOverride
		}
		if opts.ExtensionsProvider != nil {
			irBytes, err = addExtensionsToIRBytes(
				irBytes,
				opts.ExtensionsProvider,
				groupID,
				project.ProjectName,
				version,
			)
			if err != nil {
				return PublishParam{}, errors.WithStack(err)
			}
		}

		resolvedIR := string(irBytes)
		if project.Publish {
			param.ConjureIR = append(param.ConjureIR, ConjureIRPublishParam{
				ProjectName: project.ProjectName,
				IR:          resolvedIR,
				GroupID:     groupID,
			})
		}
		if typeScriptPublishEnabled(project) {
			packageVersion, err := npmPackageVersion(project.TypeScript.NpmVersionScheme, version)
			if err != nil {
				return PublishParam{}, errors.Wrapf(err, "failed to determine npm package version for project %q", project.ProjectName)
			}
			param.TypeScript = append(param.TypeScript, TypeScriptPackageInput{
				ProjectName:    project.ProjectName,
				IR:             resolvedIR,
				PackageVersion: packageVersion,
				Config:         *project.TypeScript,
			})
		}
	}
	return param, nil
}

// typeScriptPublishEnabled reports whether project's TypeScript client should be published as an npm package.
func typeScriptPublishEnabled(project ConjureProjectParam) bool {
	return project.TypeScript != nil && project.TypeScript.Publish
}

func anyTypeScriptPublishEnabled(projects ConjureProjectParams) bool {
	for _, project := range projects {
		if typeScriptPublishEnabled(project) {
			return true
		}
	}
	return false
}

// rejectDuplicateTypeScriptPackages returns an error if two projects would resolve to the same npm package name and version.
func rejectDuplicateTypeScriptPackages(projects ConjureProjectParams, version string) error {
	firstProjectForPackage := make(map[string]string, len(projects))
	for _, project := range projects {
		if !typeScriptPublishEnabled(project) {
			continue
		}
		packageVersion, err := npmPackageVersion(project.TypeScript.NpmVersionScheme, version)
		if err != nil {
			return errors.Wrapf(err, "failed to determine npm package version for project %q", project.ProjectName)
		}
		packageAndVersion := project.TypeScript.PackageName + "@" + packageVersion
		if firstProject, ok := firstProjectForPackage[packageAndVersion]; ok {
			return errors.Errorf(
				"projects %q and %q both resolve to npm package %s@%s",
				firstProject, project.ProjectName, project.TypeScript.PackageName, packageVersion,
			)
		}
		firstProjectForPackage[packageAndVersion] = project.ProjectName
	}
	return nil
}
