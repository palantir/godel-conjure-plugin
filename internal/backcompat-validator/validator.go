// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package backcompatvalidator

import (
	"encoding/json"
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/backcompat"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config"
	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// BackCompatAsset provides methods for performing backcompat validation and accepting breaks.
type BackCompatAsset struct {
	configFile string
	assets     []string
}

// New creates a new BackCompatAsset that discovers and invokes backcompat assets.
//
// Parameters:
//   - configFile:  The path to the plugin configuration file.
//   - assets:      A list of asset executable paths to be queried for backcompat validation.
//
// Returns:
//   - *BackCompatAsset: An asset handler that provides methods for checking backcompat and accepting breaks.
func New(configFile string, assets []string) *BackCompatAsset {
	return &BackCompatAsset{
		configFile: configFile,
		assets:     assets,
	}
}

// CheckBackCompat validates API compatibility between the current IR and the previously published IR.
// It returns an error if validation fails.
func (b *BackCompatAsset) CheckBackCompat(projectName string, param conjureplugin.ConjureProjectParam, godelProjectDir string) error {
	return b.runOperation(projectName, param, godelProjectDir, "checkBackCompat")
}

// AcceptBackCompatBreaks accepts backcompat breaks for a given project by writing lockfile entries.
func (b *BackCompatAsset) AcceptBackCompatBreaks(projectName string, param conjureplugin.ConjureProjectParam, godelProjectDir string) error {
	return b.runOperation(projectName, param, godelProjectDir, "acceptBackCompatBreaks")
}

func (b *BackCompatAsset) runOperation(projectName string, param conjureplugin.ConjureProjectParam, godelProjectDir string, operationType string) error {
	if param.GroupID == "" {
		// Skip projects without group-id
		return nil
	}
	if !param.Publish {
		// Skip projects that don't publish
		return nil
	}

	irBytes, err := param.IRProvider.IRBytes()
	if err != nil {
		return errors.Wrapf(err, "failed to get IR bytes")
	}

	irFile, err := tempfilecreator.WriteBytesToTempFile(irBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to write IR to temp file")
	}

	projectConfig, err := getProjectConfig(b.configFile, projectName)
	if err != nil {
		return errors.Wrapf(err, "failed to get project config")
	}

	// Discover backcompat assets
	var backcompatAssets []string
	for _, asset := range b.assets {
		cmd := exec.Command(asset, "_assetInfo")
		output, err := cmd.Output()
		if err != nil {
			return errors.Wrapf(err, "failed to execute %v\nOutput:\n%s", cmd.Args, string(output))
		}

		var response assetInfoResponse
		if err := json.Unmarshal(output, &response); err != nil {
			return errors.Wrapf(err, "failed to parse asset info response")
		}

		if response.Type == nil {
			return errors.Errorf(`invalid response from calling %v; wanted a JSON object with a "type" key; but got:\n%v`, cmd.Args, string(output))
		}

		if *response.Type == "backcompat" {
			backcompatAssets = append(backcompatAssets, asset)
		}
	}

	// Validate that exactly one backcompat asset is present
	if len(backcompatAssets) == 0 {
		// No backcompat assets configured, skip silently
		return nil
	}
	if len(backcompatAssets) > 1 {
		return errors.Errorf("multiple backcompat assets detected (%d), but only one is supported. Please configure exactly one backcompat asset", len(backcompatAssets))
	}

	asset := backcompatAssets[0]

	// Invoke the asset with the appropriate operation
	var arg []byte
	switch operationType {
	case "checkBackCompat":
		arg, err = json.Marshal(backcompat.Input{
			Type: "checkBackCompat",
			CheckBackCompat: &backcompat.CheckBackCompatInput{
				CurrentIR:       irFile,
				Project:         projectName,
				GroupID:         param.GroupID,
				ProjectConfig:   projectConfig,
				GodelProjectDir: godelProjectDir,
			},
		})
	case "acceptBackCompatBreaks":
		arg, err = json.Marshal(backcompat.Input{
			Type: "acceptBackCompatBreaks",
			AcceptBackCompatBreaks: &backcompat.AcceptBreaksInput{
				CurrentIR:       irFile,
				Project:         projectName,
				GroupID:         param.GroupID,
				ProjectConfig:   projectConfig,
				GodelProjectDir: godelProjectDir,
			},
		})
	default:
		return errors.Errorf("unknown operation type: %s", operationType)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %s input", operationType)
	}

	cmd := exec.Command(asset, string(arg))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to execute %v\nOutput:\n%s", cmd.Args, string(output))
	}

	return nil
}

func getProjectConfig(configFile string, projectName string) (map[string]any, error) {
	cfg, err := config.ReadConfigFromFile(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config file")
	}

	projectCfg, ok := cfg.ProjectConfigs[projectName]
	if !ok {
		return nil, errors.Errorf("project %s not found in config", projectName)
	}

	yamlBytes, err := yaml.Marshal(projectCfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal project config to YAML")
	}

	var result map[string]any
	if err := yaml.Unmarshal(yamlBytes, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal YAML to map")
	}

	return result, nil
}

type assetInfoResponse struct {
	Type *string `json:"type"`
}
