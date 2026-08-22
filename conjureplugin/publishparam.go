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
	gitversioner "github.com/palantir/distgo/projectversioner/git"
	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/pkg/errors"
)

// PublishParam contains the fully resolved inputs for a publish operation.
type PublishParam struct {
	Version   string
	ConjureIR []ConjureIRPublishParam
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
		if !project.Publish {
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

	param := PublishParam{
		Version:   version,
		ConjureIR: make([]ConjureIRPublishParam, 0, len(selectedProjects)),
	}
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

		param.ConjureIR = append(param.ConjureIR, ConjureIRPublishParam{
			ProjectName: project.ProjectName,
			IR:          string(irBytes),
			GroupID:     groupID,
		})
	}
	return param, nil
}
