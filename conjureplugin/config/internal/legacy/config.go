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

package legacy

import (
	v0 "github.com/palantir/godel-conjure-plugin/v4/conjureplugin/config/internal/v0"
	"github.com/palantir/godel/v2/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConfigWithLegacy struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline"`
	Config                           `yaml:",inline"`
}

type Config struct {
	ConjureProjectConfigs conjureProjectConfig `yaml:"conjure-projects"`
}

type conjureProjectConfig yaml.MapSlice

func (a *conjureProjectConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var mapSlice yaml.MapSlice
	if err := unmarshal(&mapSlice); err != nil {
		return err
	}
	// values of MapSlice are known to be ConjureProjectConfig, so read them out as such
	for i, v := range mapSlice {
		bytes, err := yaml.Marshal(v.Value)
		if err != nil {
			return err
		}
		var currCfg ConjureProjectConfig
		if err := yaml.Unmarshal(bytes, &currCfg); err != nil {
			return err
		}
		mapSlice[i].Value = currCfg
	}
	*a = conjureProjectConfig(mapSlice)
	return nil
}

type ConjureProjectConfig struct {
	ProjectFile string `yaml:"project-file"`
	SkipGet     bool   `yaml:"skip-get"`
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var legacyCfg ConfigWithLegacy
	if err := yaml.UnmarshalStrict(cfgBytes, &legacyCfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal conjure-plugin legacy configuration")
	}

	keyToOrder := make(map[string]int)
	for i, mapItem := range legacyCfg.Config.ConjureProjectConfigs {
		keyToOrder[mapItem.Key.(string)] = i
	}

	conjureProjectsV0 := make(map[string]v0.ConjureProjectConfig)
	for _, mapItem := range legacyCfg.Config.ConjureProjectConfigs {
		currKey := mapItem.Key.(string)
		currVal := mapItem.Value.(ConjureProjectConfig)

		conjureProjectsV0[currKey] = v0.ConjureProjectConfig{
			Order:       keyToOrder[currKey],
			ProjectFile: currVal.ProjectFile,
			SkipGet:     currVal.SkipGet,
		}
	}

	v0Cfg := v0.Config{
		ConjureProjects: conjureProjectsV0,
	}
	upgradedBytes, err := yaml.Marshal(v0Cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal conjure-plugin v0 configuration")
	}
	return upgradedBytes, nil
}
