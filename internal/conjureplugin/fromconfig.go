// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package conjureplugin

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v7/conjureplugin/config"
	pkgerror "github.com/pkg/errors"
)

// ParamsFromConfig returns the ConjureProjectParams representation of cfg. This function performs semantic
// validation of the configuration.
//
// Semantic issues with configuration are classified as either warnings or errors. Warnings are considered issues that
// the caller may want to be alerted or warned about, but for which the configuration is still legal/valid. Errors are
// issues that cause the configuration to be considered invalid.
//
// Currently, if multiple Conjure projects have the same output directory (after normalization using filepath.Clean),
// this is considered to be warning. The returned warning is an error created using errors.Join that contains one error
// per output path shared by multiple projects.
func ParamsFromConfig(cfg *config.ConjurePluginConfig) (_ ConjureProjectParams, warnings []error, _ error) {
	conflicts := config.ToConjurePluginConfig(cfg).OutputDirConflicts()

	var params ConjureProjectParams
	for _, project := range cfg.ProjectConfigs {
		projectName := project.Name
		if err := validateProjectName(projectName); err != nil {
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

		irProvider, err := irProviderFromLocatorConfig((*config.IRLocatorConfig)(&currConfig.IRLocator))
		if err != nil {
			return nil, nil, pkgerror.Wrapf(err, "failed to convert configuration for %s to provider", projectName)
		}

		groupID := cfg.GroupID
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
		typeScript, err := typeScriptParamFromConfig((*config.TypeScriptConfig)(currConfig.TypeScript))
		if err != nil {
			return nil, nil, pkgerror.Wrapf(err, "invalid typescript configuration for project %q", projectName)
		}

		params = append(params, ConjureProjectParam{
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
			ExportErrorDecoder:       currConfig.ExportErrorDecoder,
			ErrorParameterFormatJSON: currConfig.ErrorParameterFormatJSON,
			TypeScript:               typeScript,
		})
	}
	var err error
	if !cfg.AllowConflictingOutputDirs {
		for _, project := range cfg.ProjectConfigs {
			err = errors.Join(append([]error{err}, conflicts[project.Name]...)...)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("output directory conflicts detected: %w", err)
		}
	}

	for _, project := range cfg.ProjectConfigs {
		warnings = append(warnings, conflicts[project.Name]...)
	}

	return params, warnings, nil
}

// irProviderFromLocatorConfig resolves cfg into an IRProvider, auto-detecting the locator type from the locator
// string when one is not explicitly specified.
func irProviderFromLocatorConfig(cfg *config.IRLocatorConfig) (IRProvider, error) {
	if cfg.Locator == "" {
		return nil, pkgerror.Errorf("locator cannot be empty")
	}

	locatorType := config.LocatorType(cfg.Type)
	if locatorType == "" || locatorType == config.LocatorTypeAuto {
		if parsedURL, err := url.Parse(cfg.Locator); err == nil && parsedURL.Scheme != "" {
			// if locator can be parsed as a URL and it has a scheme explicitly specified, assume it is remote
			locatorType = config.LocatorTypeRemote
		} else {
			// treat as local: determine if path should be used as file or directory
			switch lowercaseLocator := strings.ToLower(cfg.Locator); {
			case strings.HasSuffix(lowercaseLocator, ".yml") || strings.HasSuffix(lowercaseLocator, ".yaml"):
				locatorType = config.LocatorTypeYAML
			case strings.HasSuffix(lowercaseLocator, ".json"):
				locatorType = config.LocatorTypeIRFile
			default:
				// assume path is to local YAML directory
				locatorType = config.LocatorTypeYAML

				// if path exists and is a file, treat path as an IR file
				if fi, err := os.Stat(cfg.Locator); err == nil && !fi.IsDir() {
					locatorType = config.LocatorTypeIRFile
				}
			}
		}
	}

	switch locatorType {
	case config.LocatorTypeRemote:
		return NewHTTPIRProvider(cfg.Locator), nil
	case config.LocatorTypeYAML:
		return NewLocalYAMLIRProvider(cfg.Locator), nil
	case config.LocatorTypeIRFile:
		return NewLocalFileIRProvider(cfg.Locator), nil
	default:
		return nil, pkgerror.Errorf("unknown locator type: %s", locatorType)
	}
}

// typeScriptParamFromConfig converts cfg into a TypeScriptParam, applying defaults and validating that the
// configuration is well-formed. Returns nil if cfg is nil.
func typeScriptParamFromConfig(cfg *config.TypeScriptConfig) (*TypeScriptParam, error) {
	if cfg == nil {
		return nil, nil
	}
	if cfg.PackageName == "" {
		return nil, errors.New("typescript package-name must be specified")
	}

	versionScheme := NpmVersionScheme(cfg.NpmVersionScheme)
	if versionScheme == "" {
		versionScheme = NpmVersionSchemeGit
	}
	if versionScheme != NpmVersionSchemeGit && versionScheme != NpmVersionSchemeGeneratorMajor {
		return nil, errors.New("npm-version-scheme must be one of \"" + string(NpmVersionSchemeGit) + "\" or \"" + string(NpmVersionSchemeGeneratorMajor) + "\"")
	}

	generateThrowingServices := true
	if cfg.GenerateThrowingServices != nil {
		generateThrowingServices = *cfg.GenerateThrowingServices
	}

	return &TypeScriptParam{
		PackageName:                 cfg.PackageName,
		NpmVersionScheme:            versionScheme,
		FlavorizedAliases:           cfg.FlavorizedAliases,
		NodeCompatibleModules:       cfg.NodeCompatibleModules,
		ReadonlyInterfaces:          cfg.ReadonlyInterfaces,
		GenerateThrowingServices:    generateThrowingServices,
		GenerateNonThrowingServices: cfg.GenerateNonThrowingServices,
	}, nil
}

// validateProjectName validates that a project name is safe to use as part of a file path.
// Returns an error if:
// - The name contains forward slashes (/) or backslashes (\)
// - The name is "." or ".."
func validateProjectName(projectName string) error {
	if strings.Contains(projectName, "/") || strings.Contains(projectName, "\\") {
		return pkgerror.Errorf("project name %q cannot contain path separators (/ or \\)", projectName)
	}
	if projectName == "." || projectName == ".." {
		return pkgerror.Errorf("project name cannot be %q", projectName)
	}
	return nil
}
