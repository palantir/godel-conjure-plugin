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
	"fmt"
	"os"
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config"
	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"gopkg.in/yaml.v3"
)

// BackCompatAsset provides methods for performing backcompat validation and accepting breaks.
type BackCompatAsset struct {
	configFile string
	assets     []string
	debug      bool
}

// New creates a new BackCompatAsset that discovers and invokes backcompat assets.
//
// Parameters:
//   - configFile:  The path to the plugin configuration file.
//   - assets:      A list of asset executable paths to be queried for backcompat validation.
//   - debug:       Enable debug logging.
//
// Returns:
//   - *BackCompatAsset: An asset handler that provides methods for checking backcompat and accepting breaks.
func New(configFile string, assets []string, debug bool) *BackCompatAsset {
	return &BackCompatAsset{
		configFile: configFile,
		assets:     assets,
		debug:      debug,
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

func (b *BackCompatAsset) debugf(format string, args ...interface{}) {
	if b.debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func (b *BackCompatAsset) runOperation(projectName string, param conjureplugin.ConjureProjectParam, godelProjectDir string, operationType string) error {
	b.debugf("Starting %s for project %s", operationType, projectName)
	b.debugf("GroupID: %s, Publish: %v", param.GroupID, param.Publish)

	if param.GroupID == "" {
		b.debugf("Skipping project %s: no group-id configured", projectName)
		// Skip projects without group-id
		return nil
	}
	if !param.Publish {
		b.debugf("Skipping project %s: publish is false", projectName)
		// Skip projects that don't publish
		return nil
	}

	irBytes, err := param.IRProvider.IRBytes()
	if err != nil {
		return fmt.Errorf("failed to get IR bytes: %w", err)
	}
	b.debugf("Got IR bytes: %d bytes", len(irBytes))

	irFile, err := tempfilecreator.WriteBytesToTempFile(irBytes)
	if err != nil {
		return fmt.Errorf("failed to write IR to temp file: %w", err)
	}
	b.debugf("Wrote IR to temp file: %s", irFile)

	projectConfig, err := getProjectConfig(b.configFile, projectName)
	if err != nil {
		return fmt.Errorf("failed to get project config: %w", err)
	}
	b.debugf("Got project config for %s", projectName)

	// Discover backcompat assets
	b.debugf("Discovering backcompat assets from %d total assets", len(b.assets))
	var backcompatAssets []string
	for _, asset := range b.assets {
		b.debugf("Checking asset: %s", asset)
		cmd := exec.Command(asset, "_assetInfo")
		stdout, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("%w: failed to execute %v\nstdout:\n%s\nstderr:\n%s", err, cmd.Args, string(stdout), string(exitErr.Stderr))
			}
			return fmt.Errorf("%w: failed to execute %v\nstdout:\n%s", err, cmd.Args, string(stdout))
		}

		var response assetInfoResponse
		if err := json.Unmarshal(stdout, &response); err != nil {
			return fmt.Errorf("failed to parse asset info response: %w", err)
		}

		if response.Type == nil {
			return fmt.Errorf("invalid response from calling %v; wanted a JSON object with a `type` key; but got:\n%v", cmd.Args, string(stdout))
		}

		if *response.Type == "backcompat" {
			b.debugf("Asset %s is a backcompat asset", asset)
			backcompatAssets = append(backcompatAssets, asset)
		} else {
			b.debugf("Asset %s is type %s, not backcompat", asset, *response.Type)
		}
	}

	// Validate that exactly one backcompat asset is present
	b.debugf("Found %d backcompat assets", len(backcompatAssets))
	if len(backcompatAssets) == 0 {
		b.debugf("No backcompat assets configured, skipping")
		// No backcompat assets configured, skip silently
		return nil
	}
	if len(backcompatAssets) > 1 {
		return fmt.Errorf("multiple backcompat assets detected (%d), but only one is supported. Please configure exactly one backcompat asset", len(backcompatAssets))
	}

	asset := backcompatAssets[0]
	b.debugf("Using backcompat asset: %s", asset)

	// Invoke the asset with the appropriate operation
	var arg []byte
	switch operationType {
	case "checkBackCompat":
		arg, err = json.Marshal(Input{
			Type: "checkBackCompat",
			CheckBackCompat: &CheckBackCompatInput{
				CurrentIR:       irFile,
				Project:         projectName,
				GroupID:         param.GroupID,
				ProjectConfig:   projectConfig,
				GodelProjectDir: godelProjectDir,
			},
		})
	case "acceptBackCompatBreaks":
		arg, err = json.Marshal(Input{
			Type: "acceptBackCompatBreaks",
			AcceptBackCompatBreaks: &AcceptBreaksInput{
				CurrentIR:       irFile,
				Project:         projectName,
				GroupID:         param.GroupID,
				ProjectConfig:   projectConfig,
				GodelProjectDir: godelProjectDir,
			},
		})
	default:
		return fmt.Errorf("unknown operation type: %s", operationType)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal %s input: %w", operationType, err)
	}
	b.debugf("Marshaled input for %s: %s", operationType, string(arg))

	cmd := exec.Command(asset, string(arg))
	cmd.Stderr = os.Stderr
	b.debugf("Executing: %v", cmd.Args)
	stdout, err := cmd.Output()
	if err != nil {
		b.debugf("Command failed with error: %v", err)
		if _, ok := err.(*exec.ExitError); ok {
			// Print the stdout which contains the user-facing error message
			b.debugf("ExitError stdout: %s", string(stdout))
			return fmt.Errorf("%s", string(stdout))
		}
		return fmt.Errorf("%w: failed to execute %v\nstdout:\n%s", err, cmd.Args, string(stdout))
	}

	b.debugf("Command succeeded, stdout: %s", string(stdout))

	// Success case: asset should output {}
	var result map[string]any
	if err := json.Unmarshal(stdout, &result); err != nil {
		return fmt.Errorf("failed to parse %s result: %w", operationType, err)
	}

	b.debugf("Successfully completed %s for project %s", operationType, projectName)
	return nil
}

func getProjectConfig(configFile string, projectName string) (map[string]any, error) {
	cfg, err := config.ReadConfigFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	projectCfg, ok := cfg.ProjectConfigs[projectName]
	if !ok {
		return nil, fmt.Errorf("project %s not found in config", projectName)
	}

	yamlBytes, err := yaml.Marshal(projectCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal project config to YAML: %w", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(yamlBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML to map: %w", err)
	}

	return result, nil
}

// Input represents the JSON input sent to the backcompat asset.
type Input struct {
	Type                   string                `json:"type"`
	CheckBackCompat        *CheckBackCompatInput `json:"checkBackCompat,omitempty"`
	AcceptBackCompatBreaks *AcceptBreaksInput    `json:"acceptBackCompatBreaks,omitempty"`
}

// CheckBackCompatInput contains the inputs for checking backcompat.
type CheckBackCompatInput struct {
	CurrentIR       string         `json:"currentIR"`
	Project         string         `json:"project"`
	GroupID         string         `json:"groupId"`
	ProjectConfig   map[string]any `json:"projectConfig"`
	GodelProjectDir string         `json:"godelProjectDir"`
}

// AcceptBreaksInput contains the inputs for accepting backcompat breaks.
type AcceptBreaksInput struct {
	CurrentIR       string         `json:"currentIR"`
	Project         string         `json:"project"`
	GroupID         string         `json:"groupId"`
	ProjectConfig   map[string]any `json:"projectConfig"`
	GodelProjectDir string         `json:"godelProjectDir"`
}

type assetInfoResponse struct {
	Type *string `json:"type"`
}
