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
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ConjurePluginConfig v1.ConjurePluginConfig

func ToConjurePluginConfig(in *ConjurePluginConfig) *v1.ConjurePluginConfig {
	return (*v1.ConjurePluginConfig)(in)
}

func (c *ConjurePluginConfig) ToParams(cliGroupID *string, stdout io.Writer) (conjureplugin.ConjureProjectParams, error) {
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

		publishVal := false
		// if value for "publish" is not specified, treat as "true" only if provider generates IR from YAML
		if currConfig.Publish == nil {
			publishVal = irProvider.GeneratedFromYAML()
		}
		acceptFuncsFlag := true
		if currConfig.AcceptFuncs != nil {
			acceptFuncsFlag = *currConfig.AcceptFuncs
		}

		var groupID *string
		if c.GroupID != nil {
			groupID = c.GroupID
		}
		if currConfig.GroupID != nil {
			groupID = currConfig.GroupID
		}
		if cliGroupID != nil {
			groupID = cliGroupID
		}

		if groupID == nil {
			return conjureplugin.ConjureProjectParams{}, errors.Errorf("group-id must be specified by command line or config")
		}

		params[key] = conjureplugin.ConjureProjectParam{
			OutputDir:   currConfig.OutputDir,
			IRProvider:  irProvider,
			AcceptFuncs: acceptFuncsFlag,
			Server:      currConfig.Server,
			CLI:         currConfig.CLI,
			Publish:     publishVal,
			GroupID:     *groupID,
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

// ResolveGroupID determines the group ID for a given project based on the following precedence:
// 1. CLI flag override (if cliGroupID is non-nil, indicating it was explicitly provided)
// 2. Per-project group-id in config
// 3. Top-level group-id in config
// Returns an error if no group-id is found.
func (c *ConjurePluginConfig) ResolveGroupID(projectKey string, cliGroupID *string) (string, error) {
	// If CLI flag was explicitly provided, use CLI value
	if cliGroupID != nil {
		return *cliGroupID, nil
	}

	// Check per-project group-id first
	if projectConfig, ok := c.ProjectConfigs[projectKey]; ok {
		if projectConfig.GroupID != nil {
			return *projectConfig.GroupID, nil
		}
	}

	// Fall back to top-level group-id
	if c.GroupID != nil {
		return *c.GroupID, nil
	}

	// No group-id found
	return "", errors.Errorf("group-id must be specified via CLI flag (--group-id) or in configuration (top-level or per-project) for project %s", projectKey)
}
