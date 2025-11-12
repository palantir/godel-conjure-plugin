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
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	v1 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig v1.ConjurePluginConfig

func ToConjurePluginConfig(in *ConjurePluginConfig) *v1.ConjurePluginConfig {
	return (*v1.ConjurePluginConfig)(in)
}

// ToParams returns the conjureplugin.ConjureProjectParams representation of the receiver. This function performs
// semantic validation of the configuration.
//
// Semantic issues with configuration are classified as either warnings or errors. Warnings are considered issues that
// the caller may want to be alerted or warned about, but for which the configuration is still legal/valid. Errors are
// issues that cause the configuration to be considered invalid.
//
// Currently, if multiple Conjure projects have the same output directory (after normalization using filepath.Clean),
// this is considered to be warning. The returned warning is an error created using errors.Join that contains one error
// per output path shared by multiple projects.
func (c *ConjurePluginConfig) ToParams() (_ conjureplugin.ConjureProjectParams, warnings []error, err error) {
	sortedKeys := slices.Sorted(maps.Keys(c.ProjectConfigs))

	seenDirs := make(map[string][]string)
	params := make(map[string]conjureplugin.ConjureProjectParam)
	for _, key := range sortedKeys {
		currConfig := c.ProjectConfigs[key]

		// normalize outputDir
		outputDir := filepath.Clean(currConfig.OutputDir)
		seenDirs[outputDir] = append(seenDirs[outputDir], key)

		irProvider, err := (*IRLocatorConfig)(&currConfig.IRLocator).ToIRProvider()
		if err != nil {
			return conjureplugin.ConjureProjectParams{}, nil, errors.Wrapf(err, "failed to convert configuration for %s to provider", key)
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
			OutputDir:   outputDir,
			IRProvider:  irProvider,
			AcceptFuncs: acceptFuncsFlag,
			Server:      currConfig.Server,
			CLI:         currConfig.CLI,
			Publish:     publishVal,
			GroupID:     groupID,
		}
	}

	for _, outputDir := range slices.Sorted(maps.Keys(seenDirs)) {
		projects := seenDirs[outputDir]
		if len(projects) <= 1 {
			continue
		}
		warnings = append(warnings, errors.Errorf("Projects %v are configured with the same outputDir %q, which may cause conflicts when generating Conjure output", projects, outputDir))
	}

	return conjureplugin.ConjureProjectParams{
		SortedKeys: sortedKeys,
		Params:     params,
	}, warnings, nil
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
	bytes, err := os.ReadFile(f)
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
