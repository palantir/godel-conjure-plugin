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
	"github.com/palantir/godel/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig struct {
	versionedconfig.ConfigWithVersion `yaml:",inline,omitempty"`
	ProjectConfigs                    map[string]SingleConjureConfig `yaml:"projects"`
}

type SingleConjureConfig struct {
	OutputDir string          `yaml:"output-dir"`
	IRLocator IRLocatorConfig `yaml:"ir-locator"`
	// Publish specifies whether or not the IR specified by this project should be included in the publish operation.
	// If this value is not explicitly specified in configuration, it is treated as "true" for YAML sources of IR and
	// "false" for all other sources.
	Publish *bool `yaml:"publish"`
	// Server indicates if we will generate server code. Currently this is behind a feature flag and is subject to change.
	Server bool `yaml:"server,omitempty"`
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

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin v1 configuration")
	}
	return cfgBytes, nil
}
