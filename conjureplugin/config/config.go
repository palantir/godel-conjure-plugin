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
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	v2 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v2"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/validate"
	pkgerror "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig v2.ConjurePluginConfig

func ToConjurePluginConfig(in *ConjurePluginConfig) *v2.ConjurePluginConfig {
	return (*v2.ConjurePluginConfig)(in)
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
func (c *ConjurePluginConfig) ToParams() (_ conjureplugin.ConjureProjectParams, warnings []error, _ error) {
	conflicts := ToConjurePluginConfig(c).OutputDirConflicts()

	var params conjureplugin.ConjureProjectParams
	for _, project := range c.ProjectConfigs {
		projectName := project.Name
		if err := validate.ValidateProjectName(projectName); err != nil {
			return nil, nil, err
		}

		currConfig := project.Config

		outputDir := currConfig.ResolvedOutputDir(projectName)

		if !currConfig.SkipDeleteGeneratedFiles && len(conflicts[projectName]) > 0 {
			return nil, nil, errors.Join(append(
				[]error{fmt.Errorf("project %q cannot delete generated files when output directories conflict", projectName)},
				conflicts[projectName]...,
			)...)
		}

		irProvider, err := (*IRLocatorConfig)(&currConfig.IRLocator).ToIRProvider()
		if err != nil {
			return nil, nil, pkgerror.Wrapf(err, "failed to convert configuration for %s to provider", projectName)
		}

		groupID := c.GroupID
		if currConfig.GroupID != "" {
			groupID = currConfig.GroupID
		}

		var publishVal bool
		// if value for "publish" is not specified, treat as "true" only if provider generates IR from YAML
		if currConfig.Publish == nil {
			publishVal = irProvider.GeneratedFromYAML()
		} else {
			publishVal = *currConfig.Publish
		}
		acceptFuncsFlag := true
		if currConfig.AcceptFuncs != nil {
			acceptFuncsFlag = *currConfig.AcceptFuncs
		}

		// Resolve CGR module version: project-override > plugin-level > default
		cgrVersion, err := getVersionValueFromConfig(projectName, "cgr-module-version", 2, currConfig.CGRModuleVersion, c.CGRModuleVersion, []int{2, 3})
		if err != nil {
			return nil, nil, err
		}

		// Resolve WGS module version: project-override > plugin-level > default
		wgsVersion, err := getVersionValueFromConfig(projectName, "wgs-module-version", 2, currConfig.WGSModuleVersion, c.WGSModuleVersion, []int{2, 3})
		if err != nil {
			return nil, nil, err
		}

		params = append(params, conjureplugin.ConjureProjectParam{
			ProjectName:              projectName,
			OutputDir:                outputDir,
			IRProvider:               irProvider,
			AcceptFuncs:              acceptFuncsFlag,
			Server:                   currConfig.Server,
			CLI:                      currConfig.CLI,
			Publish:                  publishVal,
			GroupID:                  groupID,
			SkipConjureBackcompat:    currConfig.SkipBackCompat,
			SkipDeleteGeneratedFiles: currConfig.SkipDeleteGeneratedFiles,
			CGRModuleVersion:         cgrVersion,
			WGSModuleVersion:         wgsVersion,
		})
	}
func getVersionValueFromConfig(projectName, variableName string, defaultVal int, projectConfigVal, pluginConfigVal *int, validValues []int) (int, error) {
	validValuesMap := make(map[int]struct{})
	for _, v := range validValues {
		validValuesMap[v] = struct{}{}
	}

	versionVal := defaultVal
	source := "builtin default"

	// get value from project config or plugin config if specified (in that order)
	if projectConfigVal != nil {
		versionVal = *projectConfigVal
		source = fmt.Sprintf("project %q configuration", projectName)
	} else if pluginConfigVal != nil {
		versionVal = *pluginConfigVal
		source = "plugin configuration"
	}

	// if version value is not valid, return error
	if _, ok := validValuesMap[versionVal]; !ok {
		return 0, fmt.Errorf("%s has invalid %s value %d: valid values are %v", source, variableName, versionVal, slices.Sorted(maps.Keys(validValuesMap)))
	}

	return versionVal, nil
}
	var err error
	if !c.AllowConflictingOutputDirs {
		for _, project := range c.ProjectConfigs {
			err = errors.Join(append([]error{err}, conflicts[project.Name]...)...)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("output directory conflicts detected: %w", err)
		}
	}

	for _, project := range c.ProjectConfigs {
		warnings = append(warnings, conflicts[project.Name]...)
	}

	return params, warnings, nil
}

type SingleConjureConfig v2.SingleConjureConfig

func ToSingleConjureConfig(in *SingleConjureConfig) *v2.SingleConjureConfig {
	return (*v2.SingleConjureConfig)(in)
}

type LocatorType v2.LocatorType

type IRLocatorConfig v2.IRLocatorConfig

func ToIRLocatorConfig(in *IRLocatorConfig) *v2.IRLocatorConfig {
	return (*v2.IRLocatorConfig)(in)
}

func (cfg *IRLocatorConfig) ToIRProvider() (conjureplugin.IRProvider, error) {
	if cfg.Locator == "" {
		return nil, pkgerror.Errorf("locator cannot be empty")
	}

	locatorType := cfg.Type
	if locatorType == "" || locatorType == v2.LocatorTypeAuto {
		if parsedURL, err := url.Parse(cfg.Locator); err == nil && parsedURL.Scheme != "" {
			// if locator can be parsed as a URL and it has a scheme explicitly specified, assume it is remote
			locatorType = v2.LocatorTypeRemote
		} else {
			// treat as local: determine if path should be used as file or directory
			switch lowercaseLocator := strings.ToLower(cfg.Locator); {
			case strings.HasSuffix(lowercaseLocator, ".yml") || strings.HasSuffix(lowercaseLocator, ".yaml"):
				locatorType = v2.LocatorTypeYAML
			case strings.HasSuffix(lowercaseLocator, ".json"):
				locatorType = v2.LocatorTypeIRFile
			default:
				// assume path is to local YAML directory
				locatorType = v2.LocatorTypeYAML

				// if path exists and is a file, treat path as an IR file
				if fi, err := os.Stat(cfg.Locator); err == nil && !fi.IsDir() {
					locatorType = v2.LocatorTypeIRFile
				}
			}
		}
	}

	switch locatorType {
	case v2.LocatorTypeRemote:
		return conjureplugin.NewHTTPIRProvider(cfg.Locator), nil
	case v2.LocatorTypeYAML:
		return conjureplugin.NewLocalYAMLIRProvider(cfg.Locator), nil
	case v2.LocatorTypeIRFile:
		return conjureplugin.NewLocalFileIRProvider(cfg.Locator), nil
	default:
		return nil, pkgerror.Errorf("unknown locator type: %s", locatorType)
	}
}

func ReadConfigFromFile(f string) (ConjurePluginConfig, error) {
	bytes, err := os.ReadFile(f)
	if err != nil {
		return ConjurePluginConfig{}, pkgerror.WithStack(err)
	}
	return ReadConfigFromBytes(bytes)
}

func ReadConfigFromBytes(inputBytes []byte) (ConjurePluginConfig, error) {
	configBytes, err := UpgradeConfig(inputBytes)
	if err != nil {
		return ConjurePluginConfig{}, pkgerror.Wrapf(err, "failed to upgrade configuration")
	}
	var cfg v2.ConjurePluginConfig
	if err := yaml.UnmarshalStrict(configBytes, &cfg); err != nil {
		return ConjurePluginConfig{}, pkgerror.WithStack(err)
	}
	return ConjurePluginConfig(cfg), nil
}
