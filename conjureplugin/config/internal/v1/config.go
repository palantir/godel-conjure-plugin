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

// ToV2Config converts a v1 config to v2 config with escape valves enabled to preserve v1 behavior.
// This is used for runtime translation to enable forward compatibility.
func (v1cfg *ConjurePluginConfig) ToV2Config() v2.ConjurePluginConfig {
	v2cfg := v2.ConjurePluginConfig{
		GroupID:                    v1cfg.GroupID,
		AllowConflictingOutputDirs: true, // In v1, conflicts were warnings not errors
		ProjectConfigs:             make(map[string]v2.SingleConjureConfig),
	}

	for name, v1proj := range v1cfg.ProjectConfigs {
		v2proj := v2.SingleConjureConfig{
			OutputDir: v1proj.OutputDir,
			IRLocator: v2.IRLocatorConfig{
				Type:    v2.LocatorType(v1proj.IRLocator.Type),
				Locator: v1proj.IRLocator.Locator,
			},
			GroupID:     v1proj.GroupID,
			Publish:     v1proj.Publish,
			Server:      v1proj.Server,
			CLI:         v1proj.CLI,
			AcceptFuncs: v1proj.AcceptFuncs,
			Extensions:  v1proj.Extensions,
			// Enable escape valves to preserve exact v1 behavior
			OmitTopLevelProjectDir:   true, // v1 didn't append project name
			SkipDeleteGeneratedFiles: true, // v1 didn't clean up old files
		}
		v2cfg.ProjectConfigs[name] = v2proj
	}

	return v2cfg
}

// TranslateToV2 translates v1 configuration to v2 for runtime use (enabling forward compatibility).
// This is separate from UpgradeConfig, which is used by the upgrade-config command and intentionally
// does NOT upgrade v1 configs. See UpgradeConfig for rationale.
func TranslateToV2(cfgBytes []byte) ([]byte, error) {
	// Unmarshal as v1 config
	var v1cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &v1cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v1 configuration")
	}

	// Convert to v2 config
	v2cfg := v1cfg.ToV2Config()
	// Note: We intentionally do NOT set v2cfg.Version = "2" here, as this is a runtime translation
	// and the original config should remain as v1 from the user's perspective.

	// Marshal back to bytes
	v2bytes, err := yaml.Marshal(v2cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal v2 configuration")
	}

	return v2bytes, nil
}

// UpgradeConfig validates v1 configuration and returns it unchanged.
//
// IMPORTANT: This function intentionally does NOT upgrade v1 configs to v2.
//
// Rationale:
// The upgrade-config command is automatically run during ./godelw update, which is frequently
// triggered by automated tools like Excavator. We do NOT want to automatically upgrade all
// v1 configs to v2 because:
//
// 1. A mechanical v1→v2 translation with all escape valves enabled (omit-top-level-project-dir: true
//    and skip-delete-generated-files: true) doesn't solve any of the problems that v2 was designed
//    to address (orphaned files, output directory conflicts, non-standard placement).
//
// 2. Keeping a config as v1 preserves a valuable signal that "this project hasn't been deliberately
//    migrated to v2 standards yet." A mechanically translated v2 config with escape valves looks
//    like it has been migrated but actually hasn't, hiding the need for a proper migration.
//
// 3. Projects should remain on v1 config until they are ready to adopt v2 standards properly,
//    either by following the standard conventions or by making a conscious decision to use
//    escape valves for legitimate reasons.
//
// Projects can manually upgrade to v2 when they are ready by either:
// - Adopting v2 standards (internal/generated/conjure/{ProjectName}/, with cleanup enabled)
// - Explicitly using escape valves if needed for their specific use case
//
// The ToV2Config() method remains available for use by the config loading logic to translate
// v1 configs to v2 at runtime (enabling forward compatibility), but this upgrade path should
// not be triggered by the upgrade-config command.
func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	// Validate by attempting to unmarshal
	var v1cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &v1cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v1 configuration")
	}
	// Return the original bytes unchanged (validated but not upgraded)
	return cfgBytes, nil
}
