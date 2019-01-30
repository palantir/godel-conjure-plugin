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
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/palantir/godel-conjure-plugin/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/conjureplugin/config/internal/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig v1.ConjurePluginConfig

func ToConjurePluginConfig(in *ConjurePluginConfig) *v1.ConjurePluginConfig {
	return (*v1.ConjurePluginConfig)(in)
}

func (c *ConjurePluginConfig) ToParams() (conjureplugin.ConjureProjectParams, error) {
	var keys []string
	for k := range c.ProjectConfigs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	params := make(map[string]conjureplugin.ConjureProjectParam)
	for key, currConfig := range c.ProjectConfigs {
		irProvider, err := (*IRLocatorConfig)(&currConfig.IRLocator).ToIRProvider()
		if err != nil {
			return conjureplugin.ConjureProjectParams{}, errors.Wrapf(err, "failed to convert configuration for %s to provider", key)
		}

		publishVal := false
		// if value for "publish" is not specified, treat as "true" only if provider generates IR from YAML
		if currConfig.Publish == nil {
			publishVal = irProvider.GeneratedFromYAML()
		}
		params[key] = conjureplugin.ConjureProjectParam{
			OutputDir:  currConfig.OutputDir,
			IRProvider: irProvider,
			Server:     currConfig.Server,
			Publish:    publishVal,
		}
	}
	return conjureplugin.ConjureProjectParams{
		SortedKeys: keys,
		Params:     params,
	}, nil
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
