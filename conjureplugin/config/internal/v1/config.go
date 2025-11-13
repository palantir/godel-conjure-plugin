// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package v1

import (
	"path/filepath"
	"strings"

	v2 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v2"
	"github.com/palantir/godel/v2/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig struct {
	versionedconfig.ConfigWithVersion `yaml:",inline,omitempty"`
	// GroupID is the default group ID for all projects. Individual projects can override this.
	GroupID        string                         `yaml:"group-id,omitempty"`
	ProjectConfigs map[string]SingleConjureConfig `yaml:"projects"`
}

type SingleConjureConfig struct {
	OutputDir string          `yaml:"output-dir"`
	IRLocator IRLocatorConfig `yaml:"ir-locator"`
	// GroupID is the group ID for this project. If not specified, the top-level group-id is used.
	GroupID string `yaml:"group-id,omitempty"`
	// Publish specifies whether or not the IR specified by this project should be included in the publish operation.
	// If this value is not explicitly specified in configuration, it is treated as "true" for YAML sources of IR and
	// "false" for all other sources.
	Publish *bool `yaml:"publish"`
	// Server indicates if we will generate server code. Currently this is behind a feature flag and is subject to change.
	Server bool `yaml:"server,omitempty"`
	// CLI indicates if we will generate cobra CLI bindings. Currently this is behind a feature flag and is subject to change.
	CLI bool `yaml:"cli,omitempty"`
	// AcceptFuncs indicates if we will generate lambda based visitor code.
	// Currently this is behind a feature flag and is subject to change.
	AcceptFuncs *bool `yaml:"accept-funcs,omitempty"`
	// Extensions contain metadata for consumption by assets of type `conjure-ir-extensions-provider`.
	Extensions map[string]any `yaml:"extensions,omitempty"`
}

type LocatorType string

const (
	LocatorTypeAuto   = LocatorType("auto")
	LocatorTypeRemote = LocatorType("remote")
	LocatorTypeYAML   = LocatorType("yaml")
	LocatorTypeIRFile = LocatorType("ir-file")
)

// IRLocatorConfig is configuration that specifies a locator. It can be specified as a YAML string or as a full YAML
// object. If it is specified as a YAML string, then the string is used as the value of "Locator" and LocatorTypeAuto is
// used as the value of the type.
type IRLocatorConfig struct {
	Type    LocatorType `yaml:"type"`
	Locator string      `yaml:"locator"`
}

func (cfg *IRLocatorConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strInput string
	if err := unmarshal(&strInput); err == nil && strInput != "" {
		// input was specified as a string: use string as value of locator with "auto" type
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

func (v1proj *SingleConjureConfig) ToV2(projectName string) v2.SingleConjureConfig {
	v2proj := v2.SingleConjureConfig{
		IRLocator: v2.IRLocatorConfig{
			Type:    v2.LocatorType(v1proj.IRLocator.Type),
			Locator: v1proj.IRLocator.Locator,
		},
		GroupID:                  v1proj.GroupID,
		Publish:                  v1proj.Publish,
		Server:                   v1proj.Server,
		CLI:                      v1proj.CLI,
		AcceptFuncs:              v1proj.AcceptFuncs,
		Extensions:               v1proj.Extensions,
		SkipDeleteGeneratedFiles: true,
	}

	outputDir := v1proj.OutputDir

	normalizedOutput := filepath.Clean(outputDir)
	v2Default := filepath.Clean(filepath.Join(v2.DefaultOutputDir, projectName))

	if normalizedOutput == v2Default {
		// Case: output-dir is "internal/generated/conjure/{ProjectName}"
		// This matches v2 defaults exactly, so omit output-dir entirely
		// (v2proj.OutputDir remains empty, which defaults to v2.DefaultOutputDir)
		// No escape valves needed
	} else if normalizedOutput == filepath.Clean(v2.DefaultOutputDir) {
		// Case: output-dir is "internal/generated/conjure" (base directory without project name)
		// In v1 this generated directly to that directory. To preserve this behavior,
		// we omit output-dir (so it defaults to v2.DefaultOutputDir) and use BOTH escape valves.
		// (v2proj.OutputDir remains empty)
		v2proj.OmitTopLevelProjectDir = true
	} else if normalizedOutput == filepath.Clean(projectName) {
		// Case: output-dir matches project name (e.g., "mag-api" for project "mag-api")
		// Optimization: set output-dir to "." and let v2 append project name
		// This gives us `./{ProjectName}` which is equivalent to v1 behavior.
		// We still need skip-delete to be safe, but can enable project name appending.
		v2proj.OutputDir = "."
	} else {
		// Case: custom output directory (including empty or ".") needs escape valves
		// to preserve v1 behavior
		v2proj.OutputDir = normalizedOutput
		v2proj.OmitTopLevelProjectDir = true
	}

	return v2proj
}

// ToV2 intelligently converts a v1 config to v2, attempting to use v2 defaults when possible.
// This is the conversion logic used by UpgradeConfig.
func (v1cfg *ConjurePluginConfig) ToV2() v2.ConjurePluginConfig {
	// Create v2 config with intelligent field mapping
	v2cfg := v2.ConjurePluginConfig{
		ConfigWithVersion: versionedconfig.ConfigWithVersion{Version: "2"},
		GroupID:           v1cfg.GroupID,
		ProjectConfigs:    make(map[string]v2.SingleConjureConfig),
	}

	// Convert each project using the per-project conversion logic
	for projectName, v1proj := range v1cfg.ProjectConfigs {
		v2cfg.ProjectConfigs[projectName] = v1proj.ToV2(projectName)
	}

	// Check if we need to allow conflicting output directories.
	// We detect conflicts the same way ToParams does: exact same directory AND parent-child relationships.
	// Calculate the actual output directories for each project
	outputDirs := make(map[string][]string)
	for projectName, proj := range v2cfg.ProjectConfigs {
		actualOutputDir := proj.OutputDir
		if actualOutputDir == "" {
			actualOutputDir = v2.DefaultOutputDir
		}
		if !proj.OmitTopLevelProjectDir {
			actualOutputDir = filepath.Join(actualOutputDir, projectName)
		}
		actualOutputDir = filepath.Clean(actualOutputDir)
		outputDirs[actualOutputDir] = append(outputDirs[actualOutputDir], projectName)
	}

	// Check for conflicts: exact same directory or parent-child relationships
	// (matching v1 behavior where conflicts were warnings, not errors)
	needsConflictAllowance := false

	// Check for exact same directory
	for _, projects := range outputDirs {
		if len(projects) > 1 {
			needsConflictAllowance = true
			break
		}
	}

	// Check for parent-child directory relationships
	if !needsConflictAllowance {
		sortedDirs := make([]string, 0, len(outputDirs))
		for dir := range outputDirs {
			sortedDirs = append(sortedDirs, dir)
		}
		for i, dir1 := range sortedDirs {
			for _, dir2 := range sortedDirs[i+1:] {
				if isChild(dir1, dir2) || isChild(dir2, dir1) {
					needsConflictAllowance = true
					break
				}
			}
			if needsConflictAllowance {
				break
			}
		}
	}

	if needsConflictAllowance {
		v2cfg.AllowConflictingOutputDirs = true
	}

	return v2cfg
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var v1cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &v1cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v1 configuration")
	}

	v2cfg := v1cfg.ToV2()

	if v2cfg.Version != "2" {
		panic("aradinsky, fix this")
	}

	v2bytes, err := yaml.Marshal(v2cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal upgraded v2 configuration")
	}

	return v2bytes, nil
}

// isChild checks if child is a subdirectory of parent.
// Paths are normalized with filepath.Clean before comparison.
// todo(aradinsky); depend on the public one
func isChild(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	rel, err := filepath.Rel(parent, child)
	return err == nil && !strings.HasPrefix(rel, "..") && rel != "."
}
