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

package backcompat

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Input represents the JSON input sent to the backcompat asset.
type Input struct {
	Type                   string                `json:"type"` // Either "checkBackCompat" or "acceptBackCompatBreaks"
	CheckBackCompat        *CheckBackCompatInput `json:"checkBackCompat,omitempty"`
	AcceptBackCompatBreaks *AcceptBreaksInput    `json:"acceptBackCompatBreaks,omitempty"`
}

// CheckBackCompatInput contains the inputs for checking backcompat.
type CheckBackCompatInput struct {
	CurrentIR       string `json:"currentIR"`       // Path to a file containing the current Conjure IR to check
	Project         string `json:"project"`         // Name of the Conjure project being validated
	GroupID         string `json:"groupId"`         // Maven group ID for the project
	GodelProjectDir string `json:"godelProjectDir"` // Root directory of the gödel project
}

// AcceptBreaksInput contains the inputs for accepting backcompat breaks.
type AcceptBreaksInput struct {
	CurrentIR       string `json:"currentIR"`       // Path to a file containing the current Conjure IR to accept breaks for
	Project         string `json:"project"`         // Name of the Conjure project
	GroupID         string `json:"groupId"`         // Maven group ID for the project
	GodelProjectDir string `json:"godelProjectDir"` // Root directory of the gödel project
}

// AssetHandler defines the interface that backcompat assets should implement.
// This interface provides a structured way to handle different backcompat operations.
type AssetHandler interface {
	// CheckBackCompat validates backward compatibility between the current IR and a base IR.
	// Returns an error if compatibility issues are found or if the check fails.
	// The implementation should:
	//   - Exit with code 0 if no incompatibilities are found
	//   - Exit with code 1 if incompatibilities are found (write details to stderr)
	//   - Exit with code 2+ if an error occurs during execution
	CheckBackCompat(input *CheckBackCompatInput) error

	// AcceptBackCompatBreaks accepts any compatibility breaks identified by CheckBackCompat.
	// This typically involves writing acknowledgment entries to lockfiles or similar mechanisms.
	// Returns an error if the acceptance operation fails.
	AcceptBackCompatBreaks(input *AcceptBreaksInput) error
}

// ParseInput parses the JSON input argument and returns the typed Input structure.
// This helper function simplifies input parsing for asset implementations.
func ParseInput(jsonArg string) (*Input, error) {
	var input Input
	if err := json.Unmarshal([]byte(jsonArg), &input); err != nil {
		return nil, errors.Wrap(err, "failed to parse input JSON")
	}

	// Validate that the input has the correct structure
	switch input.Type {
	case "checkBackCompat":
		if input.CheckBackCompat == nil {
			return nil, errors.New("input type is 'checkBackCompat' but checkBackCompat field is nil")
		}
	case "acceptBackCompatBreaks":
		if input.AcceptBackCompatBreaks == nil {
			return nil, errors.New("input type is 'acceptBackCompatBreaks' but acceptBackCompatBreaks field is nil")
		}
	default:
		return nil, errors.Errorf("unknown input type: %s", input.Type)
	}

	return &input, nil
}

// HandleInput is a convenience function that parses the input and dispatches to the appropriate
// handler method based on the operation type.
// This function streamlines asset implementation by handling all the routing logic.
//
// Example usage:
//
//	func main() {
//	    if len(os.Args) != 2 {
//	        os.Exit(2)
//	    }
//
//	    arg := os.Args[1]
//	    if arg == "_assetInfo" {
//	        fmt.Println(`{"type":"backcompat"}`)
//	        return
//	    }
//
//	    handler := &MyBackCompatHandler{}
//	    if err := backcompat.HandleInput(arg, handler); err != nil {
//	        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
//	        os.Exit(2)
//	    }
//	}
func HandleInput(jsonArg string, handler AssetHandler) error {
	input, err := ParseInput(jsonArg)
	if err != nil {
		return err
	}

	switch input.Type {
	case "checkBackCompat":
		return handler.CheckBackCompat(input.CheckBackCompat)
	case "acceptBackCompatBreaks":
		return handler.AcceptBackCompatBreaks(input.AcceptBackCompatBreaks)
	default:
		return errors.Errorf("unknown operation type: %s", input.Type)
	}
}
