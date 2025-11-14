// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package v2

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/validate"
	"github.com/palantir/godel/v2/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const DefaultOutputDir = "internal/generated/conjure"

type ConjurePluginConfig struct {
	versionedconfig.ConfigWithVersion `yaml:",inline,omitempty"`
	// GroupID is the default group ID for all projects. Individual projects can override this.
	GroupID string `yaml:"group-id,omitempty"`
	// AllowConflictingOutputDirs downgrades output directory conflicts from errors to warnings.
	// Defaults to false (conflicts are errors).
	AllowConflictingOutputDirs bool                           `yaml:"allow-conflicting-output-dirs,omitempty"`
	ProjectConfigs             map[string]SingleConjureConfig `yaml:"projects"`
}

type SingleConjureConfig struct {
	// OutputDir specifies the base output directory for generated code.
	// Defaults to "internal/generated/conjure" if not specified.
	// By default, code is generated into {OutputDir}/{ProjectName}/ unless OmitTopLevelProjectDir is true.
	OutputDir string          `yaml:"output-dir,omitempty"`
	IRLocator IRLocatorConfig `yaml:"ir-locator"`
	// GroupID is the group ID for this project. If not specified, the top-level group-id is used.
	GroupID string `yaml:"group-id,omitempty"`
	// Publish specifies whether or not the IR specified by this project should be included in the publish operation.
	// If this value is not explicitly specified in configuration, it is treated as "true" for YAML sources of IR and
	// "false" for all other sources.
	Publish *bool `yaml:"publish,omitempty"`
	// Server indicates if we will generate server code. Currently this is behind a feature flag and is subject to change.
	Server bool `yaml:"server,omitempty"`
	// CLI indicates if we will generate cobra CLI bindings. Currently this is behind a feature flag and is subject to change.
	CLI bool `yaml:"cli,omitempty"`
	// AcceptFuncs indicates if we will generate lambda based visitor code.
	// Currently this is behind a feature flag and is subject to change.
	AcceptFuncs *bool `yaml:"accept-funcs,omitempty"`
	// Extensions contain metadata for consumption by assets of type `conjure-ir-extensions-provider`.
	Extensions map[string]any `yaml:"extensions,omitempty"`
	// OmitTopLevelProjectDir skips creating the {ProjectName} subdirectory.
	// When false (default), generates into {OutputDir}/{ProjectName}/.
	// When true, generates directly into {OutputDir}/.
	OmitTopLevelProjectDir bool `yaml:"omit-top-level-project-dir,omitempty"`
	// SkipDeleteGeneratedFiles skips cleanup of old generated files before regeneration.
	// When false (default), deletes all Conjure-generated files in the output directory tree before regenerating.
	// When true, preserves v1 behavior (no cleanup).
	SkipDeleteGeneratedFiles bool `yaml:"skip-delete-generated-files,omitempty"`
}

func (c *ConjurePluginConfig) Conflicts() map[string][]error {
	type Project struct {
		name   string
		config SingleConjureConfig
	}

	var projects []Project
	for project, config := range c.ProjectConfigs {
		projects = append(projects, Project{
			name:   project,
			config: config,
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].name < projects[j].name
	})

	r := make(map[string][]error)
	for i1, p1 := range projects {
		for i2, p2 := range projects {
			if i1 == i2 {
				continue
			}

			p1Dir := p1.config.ResolvedOutputDir(p1.name)
			p2Dir := p2.config.ResolvedOutputDir(p2.name)

			if p1Dir == p2Dir {
				// todo: better error messages
				r[p1.name] = append(r[p1.name], fmt.Errorf("project %q output directory %q is the same as project %q output directory", p1.name, p1Dir, p2.name))
			} else if validate.IsSubdirectory(p1Dir, p2Dir) {
				r[p1.name] = append(r[p1.name], fmt.Errorf("project %q output directory %q contains project %q output directory %q as a subdirectory", p1.name, p1Dir, p2.name, p2Dir))
			}
		}
	}

	// todo: figure out how to dedupe and normalize the errors for the easy testing

	// final := make(map[string][]error)
	// for project, result := range r {
	// 	slices.Sort(result)
	// 	result = slices.Compact(result)
	// 	var errs []error
	// 	for _, msg := range result {
	// 		errs = append(errs, stderrors.New(msg))
	// 	}
	// 	final[project] = errs
	// }

	return r
}

// ResolvedOutputDir returns the final output directory path where generated code will be written.
// It applies the following logic:
// 1. Uses OutputDir if specified, otherwise defaults to DefaultOutputDir ("internal/generated/conjure")
// 2. Appends the projectName subdirectory unless OmitTopLevelProjectDir is true
// 3. Normalizes the path with filepath.Clean
func (s *SingleConjureConfig) ResolvedOutputDir(projectName string) string {
	actualOutputDir := s.OutputDir
	if actualOutputDir == "" {
		actualOutputDir = DefaultOutputDir
	}
	if !s.OmitTopLevelProjectDir {
		actualOutputDir = filepath.Join(actualOutputDir, projectName)
	}
	return filepath.Clean(actualOutputDir)
}

type LocatorType string

const (
	LocatorTypeAuto   = LocatorType("auto")
	LocatorTypeRemote = LocatorType("remote")
	LocatorTypeYAML   = LocatorType("yaml")
	LocatorTypeIRFile = LocatorType("ir-file")
)

type IRLocatorConfig struct {
	Type    LocatorType `yaml:"type"`
	Locator string      `yaml:"locator"`
}

func (cfg *IRLocatorConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strInput string
	if err := unmarshal(&strInput); err == nil && strInput != "" {
		cfg.Type = LocatorTypeAuto
		cfg.Locator = strInput
		return nil
	}

	type irLocatorConfigAlias IRLocatorConfig
	var unmarshaledCfg irLocatorConfigAlias
	if err := unmarshal(&unmarshaledCfg); err != nil {
		return err
	}
	*cfg = IRLocatorConfig(unmarshaledCfg)
	return nil
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v2 configuration")
	}
	return cfgBytes, nil
}
