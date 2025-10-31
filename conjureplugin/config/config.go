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

package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	v1 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v1"
	v2 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v2"
	"github.com/palantir/godel/v2/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type v1Config = v1.ConjurePluginConfig
type v2Config = v2.ConjurePluginConfig

// ConjurePluginConfig is a union type that can hold either v1 or v2 configuration.
type ConjurePluginConfig struct {
	versionedconfig.ConfigWithVersion `yaml:",inline,omitempty"`
	v1Config                          `yaml:",inline,omitempty"`
	v2Config                          `yaml:",inline,omitempty"`
}

func (c *ConjurePluginConfig) ToParams(stdout io.Writer) (conjureplugin.ConjureProjectParams, error) {
	// Delegate to version-specific implementation based on version field
	switch c.Version {
	case "1":
		return toParamsV1(&c.v1Config, stdout)
	case "2":
		return toParamsV2(&c.v2Config, stdout)
	default:
		panic("unknown version")
	}
}

// toParamsV1 converts a v1 config to ConjureProjectParams.
// V1 behavior: use output-dir as-is, default to "." if empty, always skip cleanup.
func toParamsV1(c *v1.ConjurePluginConfig, stdout io.Writer) (conjureplugin.ConjureProjectParams, error) {
	var keys []string
	for k := range c.ProjectConfigs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	seenDirs := make(map[string][]string)
	params := make(map[string]conjureplugin.ConjureProjectParam)
	for key, currConfig := range c.ProjectConfigs {
		seenDirs[currConfig.OutputDir] = append(seenDirs[currConfig.OutputDir], key)

		irProvider, err := (*IRLocatorConfig)(&currConfig.IRLocator).ToIRProvider()
		if err != nil {
			return conjureplugin.ConjureProjectParams{}, errors.Wrapf(err, "failed to convert configuration for %s to provider", key)
		}

		groupID := c.GroupID
		if currConfig.GroupID != "" {
			groupID = currConfig.GroupID
		}

		publishVal := false
		// if value for "publish" is not specified, treat as "true" only if provider generates IR from YAML
		if currConfig.Publish == nil {
			publishVal = irProvider.GeneratedFromYAML()
		}
		acceptFuncsFlag := true
		if currConfig.AcceptFuncs != nil {
			acceptFuncsFlag = *currConfig.AcceptFuncs
		}
		params[key] = conjureplugin.ConjureProjectParam{
			OutputDir:   currConfig.OutputDir,
			IRProvider:  irProvider,
			AcceptFuncs: acceptFuncsFlag,
			Server:      currConfig.Server,
			CLI:         currConfig.CLI,
			Publish:     publishVal,
			GroupID:     groupID,
		}
	}

	for outputDir, projects := range seenDirs {
		if len(projects) > 1 {
			_, _ = fmt.Fprintf(stdout,
				"[WARNING] Duplicate outputDir detected in Conjure config (godel/config/conjure-plugin.yml): '%s'\n"+
					"  Conflicting projects: %v\n"+
					"  [NOTE] Multiple projects sharing the same outputDir can cause code generation to overwrite itself, which may result in 'conjure --verify' failures or other unexpected issues.\n",
				outputDir, projects,
			)
		}
	}

	return conjureplugin.ConjureProjectParams{
		SortedKeys: keys,
		Params:     params,
	}, nil
}

// toParamsV2 converts a v2 config to ConjureProjectParams.
// V2 behavior: default to internal/generated/conjure, append project name, enable cleanup by default.
func toParamsV2(c *v2.ConjurePluginConfig, stdout io.Writer) (conjureplugin.ConjureProjectParams, error) {
	// TODO: implement v2-specific parameter generation
	return conjureplugin.ConjureProjectParams{}, errors.New("toParamsV2 not yet implemented")
}

type SingleConjureConfig v1.SingleConjureConfig

func ToSingleConjureConfig(in *SingleConjureConfig) *v1.SingleConjureConfig {
	return (*v1.SingleConjureConfig)(in)
}

type LocatorType v1.LocatorType

type IRLocatorConfig v1.IRLocatorConfig

func ToIRLocatorConfig(in *IRLocatorConfig) *v1.IRLocatorConfig {
	return (*v1.IRLocatorConfig)(in)
}

func (cfg *IRLocatorConfig) ToIRProvider() (conjureplugin.IRProvider, error) {
	if cfg.Locator == "" {
		return nil, errors.Errorf("locator cannot be empty")
	}

	locatorType := cfg.Type
	if locatorType == "" || locatorType == v1.LocatorTypeAuto {
		if parsedURL, err := url.Parse(cfg.Locator); err == nil && parsedURL.Scheme != "" {
			// if locator can be parsed as a URL and it has a scheme explicitly specified, assume it is remote
			locatorType = v1.LocatorTypeRemote
		} else {
			// treat as local: determine if path should be used as file or directory
			switch lowercaseLocator := strings.ToLower(cfg.Locator); {
			case strings.HasSuffix(lowercaseLocator, ".yml") || strings.HasSuffix(lowercaseLocator, ".yaml"):
				locatorType = v1.LocatorTypeYAML
			case strings.HasSuffix(lowercaseLocator, ".json"):
				locatorType = v1.LocatorTypeIRFile
			default:
				// assume path is to local YAML directory
				locatorType = v1.LocatorTypeYAML

				// if path exists and is a file, treat path as an IR file
				if fi, err := os.Stat(cfg.Locator); err == nil && !fi.IsDir() {
					locatorType = v1.LocatorTypeIRFile
				}
			}
		}
	}

	switch locatorType {
	case v1.LocatorTypeRemote:
		return conjureplugin.NewHTTPIRProvider(cfg.Locator), nil
	case v1.LocatorTypeYAML:
		return conjureplugin.NewLocalYAMLIRProvider(cfg.Locator), nil
	case v1.LocatorTypeIRFile:
		return conjureplugin.NewLocalFileIRProvider(cfg.Locator), nil
	default:
		return nil, errors.Errorf("unknown locator type: %s", locatorType)
	}
}

func ReadConfigFromFile(f string) (ConjurePluginConfig, error) {
	bytes, err := ioutil.ReadFile(f)
	if err != nil {
		return ConjurePluginConfig{}, errors.WithStack(err)
	}
	return ReadConfigFromBytes(bytes)
}

func ReadConfigFromBytes(inputBytes []byte) (ConjurePluginConfig, error) {
	var cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(inputBytes, &cfg); err != nil {
		return ConjurePluginConfig{}, errors.WithStack(err)
	}
	return cfg, nil
}
