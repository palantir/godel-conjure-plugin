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
	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/pkg/errors"
)

// BackCompatAsset provides methods for performing backcompat validation and accepting breaks.
type BackCompatAsset struct {
	Asset string // path to the backcompat asset executable, empty if no asset configured
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
	// If no backcompat asset is configured, skip silently
	if b.Asset == "" {
		return nil
	}

	// Only check compatibility for IRs generated from YAML sources.
	// The plugin only checks IRs that are actually defined in the project.
	if !param.IRProvider.GeneratedFromYAML() {
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
				GodelProjectDir: godelProjectDir,
			},
		})
	default:
		return errors.Errorf("unknown operation type: %s", operationType)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %s input", operationType)
	}

	cmd := exec.Command(b.Asset, string(arg))
	output, err := cmd.CombinedOutput()

	// Check exit code to determine success/failure
	if err != nil {
		// Extract exit code from error
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()

			// Exit code 1: Found incompatibilities (for checkBackCompat mode)
			// Exit code 2+: Actual error occurred
			if exitCode == 1 {
				// Found incompatibilities - return error with the asset's output
				if len(output) > 0 {
					return errors.New(string(output))
				}
				return errors.New("compatibility check failed")
			}

			// Exit code 2+: actual error
			if len(output) > 0 {
				return errors.Errorf("asset execution failed (exit code %d):\n%s", exitCode, string(output))
			}
			return errors.Errorf("asset execution failed with exit code %d", exitCode)
		}

		// Non-exit error (e.g., command not found)
		return errors.Wrapf(err, "failed to execute %v", cmd.Args)
	}

	// Exit code 0: Success
	return nil
}
