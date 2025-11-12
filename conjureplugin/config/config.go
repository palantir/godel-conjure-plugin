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
	v2 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v2"
	"github.com/palantir/godel/v2/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// ConjurePluginConfig is now a type alias to v2.ConjurePluginConfig (the canonical version).
// v1 configs are automatically translated to v2 at load time.
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
func (c *ConjurePluginConfig) ToParams() (_ conjureplugin.ConjureProjectParams, warnings []error, err error) {
	sortedKeys := slices.Sorted(maps.Keys(c.ProjectConfigs))

	var outputDirs []string
	params := make(map[string]conjureplugin.ConjureProjectParam)
	for _, key := range sortedKeys {
		currConfig := c.ProjectConfigs[key]
		// Calculate the actual output directory
		outputDir := currConfig.OutputDir
		if outputDir == "" {
			outputDir = v2.DefaultOutputDir
		}
		if !currConfig.OmitTopLevelProjectDir {
			outputDir = filepath.Join(outputDir, key)
		}

		outputDirs = append(outputDirs, outputDir)

		irLocatorConfig := IRLocatorConfig(currConfig.IRLocator)
		irProvider, err := (&irLocatorConfig).ToIRProvider()
		if err != nil {
			return conjureplugin.ConjureProjectParams{}, nil, errors.Wrapf(err, "failed to convert configuration for %s to provider", key)
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
		params[key] = conjureplugin.ConjureProjectParam{
			OutputDir:                outputDir,
			IRProvider:               irProvider,
			AcceptFuncs:              acceptFuncsFlag,
			Server:                   currConfig.Server,
			CLI:                      currConfig.CLI,
			Publish:                  publishVal,
			GroupID:                  groupID,
			SkipDeleteGeneratedFiles: currConfig.SkipDeleteGeneratedFiles,
		}
	}

	warnings = checkDirConflicts(outputDirs)

	return conjureplugin.ConjureProjectParams{
		SortedKeys: sortedKeys,
		Params:     params,
	}, warnings, nil
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
		return nil, errors.Errorf("locator cannot be empty")
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
	// Detect config version
	version, err := versionedconfig.ConfigVersion(inputBytes)
	if err != nil {
		return ConjurePluginConfig{}, errors.WithStack(err)
	}

	// Handle version-specific loading
	var v2Bytes []byte
	switch version {
	case "", "1":
		// v1 config (or unversioned, which we treat as v1): translate to v2 for runtime use
		// Note: We use TranslateToV2 (not UpgradeConfig) because we want runtime translation
		// for forward compatibility, but UpgradeConfig intentionally does NOT upgrade v1→v2
		// to prevent automated tools from blindly upgrading configs.
		v2Bytes, err = v1.TranslateToV2(inputBytes)
		if err != nil {
			return ConjurePluginConfig{}, errors.Wrapf(err, "failed to translate v1 config to v2")
		}
	case "2":
		// Already v2, use as-is
		v2Bytes = inputBytes
	default:
		return ConjurePluginConfig{}, errors.Errorf("unsupported configuration version: %s", version)
	}

	// Unmarshal as v2 config
	var cfg ConjurePluginConfig
	if err := yaml.UnmarshalStrict(v2Bytes, &cfg); err != nil {
		return ConjurePluginConfig{}, errors.WithStack(err)
	}
	return cfg, nil
}

// checkDirConflicts checks for directory conflicts including exact duplicates and parent-child relationships.
// Returns a slice of errors describing all conflicts found.
// Note: This implementation uses filepath.Clean for normalization but does not resolve to absolute paths.
// A more complete implementation would use filepath.Abs and resolve symlinks to catch all cases,
// but we assume users are well-intentioned and this catches the common cases.
func checkDirConflicts(dirs []string) []error {
	var conflicts []error

	slices.Sort(dirs)

	for i, dir1 := range dirs {
		for _, dir2 := range dirs[i+1:] {
			if dirsConflict(dir1, dir2) {
				conflicts = append(conflicts, errors.Errorf(
					"OutputDir %q and OutputDir %q have conflicting output directories (same directory or parent-child relationship), which may cause conflicts when generating Conjure output",
					dir1, dir2))
			}
		}
	}

	return conflicts
}

// dirsConflict checks if two directories conflict (are equal or have a parent-child relationship).
func dirsConflict(dir1, dir2 string) bool {
	// Normalize paths
	dir1 = filepath.Clean(dir1)
	dir2 = filepath.Clean(dir2)

	// Check for exact match
	if dir1 == dir2 {
		return true
	}

	// Check if either is a child of the other
	return isChild(dir1, dir2) || isChild(dir2, dir1)
}

// isChild checks if child is a subdirectory of parent.
// Paths are normalized with filepath.Clean before comparison.
func isChild(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	rel, err := filepath.Rel(parent, child)
	return err == nil && !strings.HasPrefix(rel, "..") && rel != "."
}
