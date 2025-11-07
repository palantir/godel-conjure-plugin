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

package v2

import (
	v1 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v1"
	"github.com/palantir/godel/v2/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig struct {
	versionedconfig.ConfigWithVersion `yaml:",inline,omitempty"`
	// GroupID is the default group ID for all projects. Individual projects can override this.
	GroupID string `yaml:"group-id,omitempty"`
	// AllowConflictingOutputDirs, when true, downgrades output directory conflicts from errors to warnings.
	// Defaults to false (conflicts are errors).
	AllowConflictingOutputDirs bool                           `yaml:"allow-conflicting-output-dirs,omitempty"`
	ProjectConfigs             map[string]SingleConjureConfig `yaml:"projects"`
}

type SingleConjureConfig struct {
	// OutputDir is the base directory where generated code will be placed.
	// In v2, defaults to "internal/generated/conjure" if not specified.
	// By default, each project generates into {OutputDir}/{ProjectName}/ subdirectory.
	OutputDir string          `yaml:"output-dir,omitempty"`
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
	// OmitTopLevelProjectDir, when true, skips creating the {ProjectName} subdirectory.
	// Generated code will be placed directly in OutputDir instead of OutputDir/{ProjectName}/.
	// Defaults to false (project subdirectory is created).
	OmitTopLevelProjectDir bool `yaml:"omit-top-level-project-dir,omitempty"`
	// SkipDeleteGeneratedFiles, when true, skips cleanup of old generated files before regeneration.
	// Defaults to false (cleanup is performed).
	SkipDeleteGeneratedFiles bool `yaml:"skip-delete-generated-files,omitempty"`
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

// UpgradeConfig translates v1 configuration to v2, or validates v2 configuration.
// V1 configs are automatically translated with escape valves enabled to preserve exact v1 behavior:
// - omit-top-level-project-dir: true (generate directly into output-dir)
// - skip-delete-generated-files: true (no cleanup)
//
// V2 configs are validated and returned unchanged.
func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	// First, check if this is already a v2 config
	var versionCheck versionedconfig.ConfigWithVersion
	if err := yaml.Unmarshal(cfgBytes, &versionCheck); err == nil && versionCheck.Version == "2" {
		// Already v2, just validate and return
		var v2Cfg ConjurePluginConfig
		if err := yaml.UnmarshalStrict(cfgBytes, &v2Cfg); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v2 configuration")
		}
		return cfgBytes, nil
	}

	// Not v2, treat as v1 and upgrade
	var v1Cfg v1.ConjurePluginConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &v1Cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v1 configuration")
	}

	// Translate v1 to v2 with escape valves to preserve v1 behavior
	v2Cfg := ConjurePluginConfig{
		GroupID:                    v1Cfg.GroupID,
		AllowConflictingOutputDirs: true, // v1 only warns about conflicts
		ProjectConfigs:             make(map[string]SingleConjureConfig),
	}

	// Set version to "2"
	v2Cfg.ConfigWithVersion = versionedconfig.ConfigWithVersion{Version: "2"}

	for projectName, v1Project := range v1Cfg.ProjectConfigs {
		v2Project := SingleConjureConfig{
			OutputDir: v1Project.OutputDir,
			IRLocator: IRLocatorConfig{
				Type:    LocatorType(v1Project.IRLocator.Type),
				Locator: v1Project.IRLocator.Locator,
			},
			GroupID:     v1Project.GroupID,
			Publish:     v1Project.Publish,
			Server:      v1Project.Server,
			CLI:         v1Project.CLI,
			AcceptFuncs: v1Project.AcceptFuncs,
			Extensions:  v1Project.Extensions,
			// Enable escape valves to preserve v1 behavior
			OmitTopLevelProjectDir:   true,
			SkipDeleteGeneratedFiles: true,
		}

		// Special case: if output-dir was empty in v1, it defaulted to "."
		// Preserve this behavior explicitly
		if v2Project.OutputDir == "" {
			v2Project.OutputDir = "."
		}

		v2Cfg.ProjectConfigs[projectName] = v2Project
	}

	// Marshal back to YAML
	outputBytes, err := yaml.Marshal(v2Cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal v2 configuration")
	}

	return outputBytes, nil
}
