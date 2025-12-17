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
	AllowConflictingOutputDirs bool `yaml:"allow-conflicting-output-dirs,omitempty"`
	// CGRModuleVersion specifies which module version of conjure-go-runtime to use in generated code.
	// Defaults to 2 if not specified.
	CGRModuleVersion int `yaml:"cgr-module-version,omitempty"`
	// WGSModuleVersion specifies which module version of witchcraft-go-server to use in generated code.
	// Defaults to 2 if not specified.
	WGSModuleVersion int                   `yaml:"wgs-module-version,omitempty"`
	ProjectConfigs   ConjureProjectConfigs `yaml:"projects"`
}

// ConjureProjectConfigs is a type defined to support an ordered map. Its serialized form is a
// map[string]SingleConjureConfig.
//
// Implements MarshalYAML and UnmarshalYAML in a manner that supports writing and reading a map that maintains ordering.
type ConjureProjectConfigs []NamedConjureProjectConfig

type NamedConjureProjectConfig struct {
	Name   string
	Config SingleConjureConfig
}

func (c ConjureProjectConfigs) MarshalYAML() (interface{}, error) {
	var mapSlice yaml.MapSlice
	for _, project := range c {
		mapSlice = append(mapSlice, yaml.MapItem{
			Key:   project.Name,
			Value: project.Config,
		})
	}
	return mapSlice, nil
}

func (c *ConjureProjectConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if c == nil {
		return errors.Errorf("cannot unmarshal into nil ConjureProjectConfigs")
	}

	// unmarshal projects into map so that full type-based unmarshal is performed
	var projects map[string]SingleConjureConfig
	if err := unmarshal(&projects); err != nil {
		return err
	}

	// unmarshal into MapSlice to get ordered keys
	var mapSlice yaml.MapSlice
	if err := unmarshal(&mapSlice); err != nil {
		return err
	}

	var configs []NamedConjureProjectConfig
	for _, mapItem := range mapSlice {
		key := mapItem.Key.(string)
		configs = append(configs, NamedConjureProjectConfig{
			Name:   key,
			Config: projects[key],
		})
	}

	*c = configs
	return nil
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
	// SkipBackCompat indicates if backcompat operations should be skipped for this project.
	// Defaults to false (backcompat operations will run). Only valid for projects that generate IR from YAML: config validation will fail if this is set to true for projects that do not generate IR from YAML.
	SkipBackCompat bool `yaml:"skip-backcompat,omitempty"`
}

// OutputDirConflicts detects output directory conflicts between projects.
// Returns a map from project name to a list of errors describing conflicts with other projects.
//
// Two types of conflicts are detected:
//   - Identical output directories: Two projects writing to the exact same directory
//   - Nested output directories: One project's output directory is a subdirectory of another's
//
// An empty map indicates no conflicts were found.
func (c *ConjurePluginConfig) OutputDirConflicts() map[string][]error {
	result := make(map[string][]error)
	for i1, p1 := range c.ProjectConfigs {
		for i2, p2 := range c.ProjectConfigs {
			if i1 == i2 {
				continue
			}

			p1Dir := p1.Config.ResolvedOutputDir(p1.Name)
			p2Dir := p2.Config.ResolvedOutputDir(p2.Name)

			if p1Dir == p2Dir {
				result[p1.Name] = append(result[p1.Name], fmt.Errorf("project %q and %q have the same output directory %q", p1.Name, p2.Name, p1Dir))
			} else if validate.IsSubdirectory(p1Dir, p2Dir) {
				result[p1.Name] = append(result[p1.Name], fmt.Errorf("output directory %q of project %q is a subdirectory of output directory %q of project %q", p2Dir, p2.Name, p1Dir, p1.Name))
			}
		}
	}

	return result
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
	if cfg == nil {
		return errors.Errorf("cannot unmarshal into nil IRLocatorConfig")
	}

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
