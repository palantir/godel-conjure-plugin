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
	"github.com/pkg/errors"
)

// NpmPublishProject identifies one configured TypeScript project to the npm publishing planner.
type NpmPublishProject struct {
	ProjectIndex      int
	ProjectName       string
	PackageName       string
	PublisherProvider string
}

// NpmPublishPlanConfig contains operation-specific npm registry and credential inputs.
type NpmPublishPlanConfig struct {
	PublishRegistry string
	InstallRegistry string
	Username        string
	Password        string
	Token           string
	NpmrcFile       string
}

// NpmPublishGroup contains projects that can share install and publish npm configuration.
type NpmPublishGroup struct {
	ProjectIndexes   []int
	InstallNpmrcPath string
	PublishNpmrcPath string
}

// NpmPublishPlan contains prepared npm configuration for every provider-defined project group.
type NpmPublishPlan struct {
	PublishRegistry string
	Groups          []NpmPublishGroup
	cleanups        []func() error
}

type npmPublishGroupKey struct {
	provider string
	group    string
}

type pendingNpmPublishGroup struct {
	providerName   string
	publisher      npmPublisher
	usesNpmrcFile  bool
	projectIndexes []int
	packageNames   []string
}

// PrepareNpmPublishPlan groups projects according to their configured providers and prepares their npm configuration.
func PrepareNpmPublishPlan(projects []NpmPublishProject, cfg NpmPublishPlanConfig) (*NpmPublishPlan, error) {
	publishRegistry, _, err := normalizeRegistryURL(cfg.PublishRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "invalid npm publish registry")
	}

	publishers := make(map[string]npmPublisherDefinition)
	groupIndexes := make(map[npmPublishGroupKey]int)
	var groups []pendingNpmPublishGroup
	var hasNpmrcProvider, hasGeneratedProvider bool
	for _, project := range projects {
		providerName := project.PublisherProvider
		if providerName == "" {
			providerName = DefaultNpmPublisherProvider
		}
		publisherDefinition, ok := publishers[providerName]
		if !ok {
			publisherDefinition, err = npmPublisherFor(providerName)
			if err != nil {
				return nil, errors.Errorf(
					"TypeScript project %q has unsupported npm publisher provider %q",
					project.ProjectName, providerName)
			}
			publishers[providerName] = publisherDefinition
		}
		publisher := publisherDefinition.publisher
		providerGroup, err := publisher.groupKey(project.PackageName)
		if err != nil {
			return nil, err
		}
		key := npmPublishGroupKey{provider: providerName, group: providerGroup}
		groupIndex, ok := groupIndexes[key]
		if !ok {
			groupIndex = len(groups)
			groupIndexes[key] = groupIndex
			groups = append(groups, pendingNpmPublishGroup{
				providerName:  providerName,
				publisher:     publisher,
				usesNpmrcFile: publisherDefinition.usesNpmrcFile,
			})
		}
		groups[groupIndex].projectIndexes = append(groups[groupIndex].projectIndexes, project.ProjectIndex)
		groups[groupIndex].packageNames = append(groups[groupIndex].packageNames, project.PackageName)
		if publisherDefinition.usesNpmrcFile {
			hasNpmrcProvider = true
		} else {
			hasGeneratedProvider = true
		}
	}

	if hasNpmrcProvider && cfg.NpmrcFile == "" {
		return nil, errors.New("npmrc publisher provider requires an npmrc file")
	}
	if !hasNpmrcProvider && cfg.NpmrcFile != "" {
		return nil, errors.New("npmrc file was specified, but no TypeScript project uses the npmrc publisher provider")
	}
	if hasNpmrcProvider && !hasGeneratedProvider {
		if cfg.Username != "" || cfg.Password != "" || cfg.Token != "" {
			return nil, errors.New("npm credentials cannot be specified when all TypeScript projects use the npmrc publisher provider")
		}
		if cfg.InstallRegistry != "" {
			return nil, errors.New("npm install registry cannot be specified when all TypeScript projects use the npmrc publisher provider; the file's own configuration is used for install")
		}
	}

	plan := &NpmPublishPlan{PublishRegistry: publishRegistry}
	for _, group := range groups {
		files, err := group.publisher.prepare(npmConfigRequest{
			packageNames:    group.packageNames,
			publishRegistry: publishRegistry,
			installRegistry: cfg.InstallRegistry,
			username:        cfg.Username,
			password:        cfg.Password,
			token:           cfg.Token,
			npmrcFile:       cfg.NpmrcFile,
		})
		if err != nil {
			_ = plan.Close()
			if group.usesNpmrcFile {
				return nil, err
			}
			return nil, errors.Wrapf(err, "failed to configure npm registries for %s publisher", group.providerName)
		}
		if files.close != nil {
			plan.cleanups = append(plan.cleanups, files.close)
		}
		plan.Groups = append(plan.Groups, NpmPublishGroup{
			ProjectIndexes:   group.projectIndexes,
			InstallNpmrcPath: files.installPath,
			PublishNpmrcPath: files.publishPath,
		})
	}
	return plan, nil
}

// Close removes all generated npm configuration. Caller-owned npmrc files are left untouched.
func (p *NpmPublishPlan) Close() error {
	var result error
	for i := len(p.cleanups) - 1; i >= 0; i-- {
		if err := p.cleanups[i](); result == nil && err != nil {
			result = err
		}
	}
	p.cleanups = nil
	return result
}
