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
	"path"
	"sort"
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

func (c *ConjurePluginConfig) ToParams(stdout io.Writer) (conjureplugin.ConjureProjectParams, error) {
	var keys []string
	for k := range c.ProjectConfigs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	seenDirs := make(map[string][]string)
	params := make(map[string]conjureplugin.ConjureProjectParam)
	for key, currConfig := range c.ProjectConfigs {
		// Calculate the actual output directory
		outputDir := currConfig.OutputDir
		if outputDir == "" {
			outputDir = v2.DefaultOutputDir
		}
		if !currConfig.OmitTopLevelProjectDir {
			outputDir = filepath.Join(outputDir, key)
		}

		seenDirs[outputDir] = append(seenDirs[outputDir], key)

		irLocatorConfig := IRLocatorConfig(currConfig.IRLocator)
		irProvider, err := (&irLocatorConfig).ToIRProvider()
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

	// Check for conflicting output directories
	for outputDir, projects := range seenDirs {
		if len(projects) > 1 {
			message := fmt.Sprintf(
				"Duplicate outputDir detected in Conjure config: '%s'\n"+
					"  Conflicting projects: %v\n"+
					"  [NOTE] Multiple projects sharing the same outputDir can cause code generation to overwrite itself, which may result in 'conjure --verify' failures or other unexpected issues.\n",
				outputDir, projects,
			)
			if c.AllowConflictingOutputDirs {
				// Downgrade to warning
				_, _ = fmt.Fprintf(stdout, "[WARNING] %s", message)
			} else {
				// Error by default
				return conjureplugin.ConjureProjectParams{}, errors.Errorf("conflicting output directories: %s", message)
			}
		}
	}

	return conjureplugin.ConjureProjectParams{
		SortedKeys: keys,
		Params:     params,
	}, nil
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
	bytes, err := ioutil.ReadFile(f)
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
