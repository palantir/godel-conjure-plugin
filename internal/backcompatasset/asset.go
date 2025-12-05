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

package backcompatasset

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/backcompatasset"
	"github.com/pkg/errors"
)

// ExecParam specifies the parameters for executing a command. Specifies stdout, stderr, and debug mode.
type ExecParam struct {
	Stdout io.Writer
	Stderr io.Writer
	Debug  bool
}

type BackCompatChecker interface {
	// CheckBackCompat runs the asset's backcompat check for the specified project.
	// Returns true if the backwards compatibility check passes, false otherwise.
	// Returns an error only if the operation fails to execute, not based on the presence of breaks.
	// If there are any details about the failure, they are written to the Stdout of the provided ExecParam.
	CheckBackCompat(groupID, project string, currentIR string, godelProjectDir string, execParam ExecParam) (bool, error)

	// AcceptBackCompatBreaks runs the asset's accept operation for the specified project.
	// This records/accepts the current state as the baseline for future backcompat checks.
	// Returns an error only if the operation fails to execute, not based on the presence of breaks.
	AcceptBackCompatBreaks(groupID, project string, currentIR string, godelProjectDir string, execParam ExecParam) error
}

type backCompatCheckerImpl struct {
	asset string
}

func New(asset string) BackCompatChecker {
	return &backCompatCheckerImpl{
		asset: asset,
	}
}

func (b *backCompatCheckerImpl) CheckBackCompat(
	groupID, project string,
	currentIR string,
	godelProjectDir string,
	execParam ExecParam,
) (bool, error) {
	execCmd := exec.Command(b.asset,
		backcompatasset.CheckBackCompatCommand,
		"--"+backcompatasset.GroupIDFlagName, groupID,
		"--"+backcompatasset.ProjectFlagName, project,
		"--"+backcompatasset.CurrentIRFlagName, currentIR,
		"--"+backcompatasset.GodelProjectDirFlagName, godelProjectDir,
	)
	execCmd.Stdout = execParam.Stdout
	execCmd.Stderr = execParam.Stderr

	if execParam.Debug {
		_, _ = fmt.Fprintf(execCmd.Stderr, "CheckBackCompat: running command %v\n", execCmd)
	}
	err := execCmd.Run()

	// command executed successfully: compatible
	if err == nil {
		return true, nil
	}

	// if command exited with exit code 1, indicates that IR is not backwards compatible:
	// return false, but no error
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	// treat any other error or exit code as an error
	return false, errors.Wrapf(err, "failed to check back compatibility for project %v", project)
}

func (b *backCompatCheckerImpl) AcceptBackCompatBreaks(
	groupID, project string,
	currentIR string,
	godelProjectDir string,
	execParam ExecParam,
) error {
	execCmd := exec.Command(b.asset,
		backcompatasset.AcceptBackCompatBreaksCommand,
		"--"+backcompatasset.GroupIDFlagName, groupID,
		"--"+backcompatasset.ProjectFlagName, project,
		"--"+backcompatasset.CurrentIRFlagName, currentIR,
		"--"+backcompatasset.GodelProjectDirFlagName, godelProjectDir,
	)
	execCmd.Stdout = execParam.Stdout
	execCmd.Stderr = execParam.Stderr

	if execParam.Debug {
		_, _ = fmt.Fprintf(execCmd.Stderr, "AcceptBackCompatBreaks: running command %v\n", execCmd)
	}

	if err := execCmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to execute accept conjure backcompat breaks for project %q", project)
	}
	return nil
}
