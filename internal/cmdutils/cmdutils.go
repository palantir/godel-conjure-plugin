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

package cmdutils

import (
	"os/exec"

	"github.com/palantir/pkg/safejson"
	"github.com/pkg/errors"
)

// GetCommandOutputAsJSON runs the provided exec.Cmd, unmarshals the command's stdout as JSON into a value of type T,
// and returns the unmarshaled value. Returns an error if an error occurs while running the command or unmarshaling the
// output.
func GetCommandOutputAsJSON[T any](cmd *exec.Cmd) (T, error) {
	// needed to return zero value in case of error
	var returnVal T

	stdout, err := RunCommand(cmd)
	if err != nil {
		return returnVal, err
	}

	if err := safejson.Unmarshal(stdout, &returnVal); err != nil {
		return returnVal, errors.Wrapf(err, "failed to unmarshal output as JSON")
	}
	return returnVal, nil
}

// RunCommand runs the provided exec.Cmd and returns the result of cmd.Output() if the returned error is nil. If the
// returned error is non-nil, returns a wrapped error that includes the command's stdout and stderr.
func RunCommand(cmd *exec.Cmd) ([]byte, error) {
	stdoutOutput, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdoutOutput, errors.Wrapf(err, "failed to execute %v\nstdout:\n%s\nstderr:\n%s", cmd.Args, string(stdoutOutput), string(exitErr.Stderr))
		}
		return stdoutOutput, errors.Wrapf(err, "failed to execute %v\nstdout:\n%s", cmd.Args, string(stdoutOutput))
	}
	return stdoutOutput, nil
}
