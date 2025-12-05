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
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/backcompatasset"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
)

// BackCompatChecker represents a wrapper around a backcompat asset executable.
type BackCompatChecker interface {
	// CheckBackCompat runs the asset's backcompat check for the specified project.
	// It executes the asset as a command-line tool with the relevant arguments.
	// If the command exits with code 1, it indicates backcompat breaks were found and returns an error specific to that case.
	// Any other execution errors are wrapped and returned.
	CheckBackCompat(groupID, project string, currentIR string, godelProjectDir string, cmdParams conjureplugin.CmdParams) error

	// AcceptBackCompatBreaks runs the asset's accept operation for the specified project.
	// This records/accepts the current state as the baseline for future backcompat checks.
	// Returns an error only if the operation fails to execute, not based on the presence of breaks.
	AcceptBackCompatBreaks(groupID, project string, currentIR string, godelProjectDir string, cmdParams conjureplugin.CmdParams) error
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
	cmdParams conjureplugin.CmdParams,
) error {
	execCmd := exec.Command(b.asset,
		backcompatasset.CheckBackCompatCommand,
		"--"+backcompatasset.GroupIDFlagName, groupID,
		"--"+backcompatasset.ProjectFlagName, project,
		"--"+backcompatasset.CurrentIRFlagName, currentIR,
		"--"+backcompatasset.GodelProjectDirFlagName, godelProjectDir,
	)
	execCmd.Stdout = cmdParams.Stdout
	execCmd.Stderr = cmdParams.Stderr

	if cmdParams.Debug {
		_, _ = fmt.Fprintf(execCmd.Stderr, "CheckBackCompat: running command %v\n", execCmd)
	}
	err := execCmd.Run()

	if err == nil {
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		_, _ = fmt.Fprintf(execCmd.Stderr, "Conjure breaks found in project %q\n", project)
		return fmt.Errorf("conjure breaks found in project %q", project)
	}

	_, _ = fmt.Fprintf(execCmd.Stderr, "Failed to execute check conjure backcompat on project %q\n", project)
	return fmt.Errorf("failed to execute check conjure backcompat on project %q", project)
}

func (b *backCompatCheckerImpl) AcceptBackCompatBreaks(
	groupID, project string,
	currentIR string,
	godelProjectDir string,
	cmdParams conjureplugin.CmdParams,
) error {
	execCmd := exec.Command(b.asset,
		backcompatasset.AcceptBackCompatBreaksCommand,
		"--"+backcompatasset.GroupIDFlagName, groupID,
		"--"+backcompatasset.ProjectFlagName, project,
		"--"+backcompatasset.CurrentIRFlagName, currentIR,
		"--"+backcompatasset.GodelProjectDirFlagName, godelProjectDir,
	)
	execCmd.Stdout = cmdParams.Stdout
	execCmd.Stderr = cmdParams.Stderr

	if cmdParams.Debug {
		_, _ = fmt.Fprintf(execCmd.Stderr, "AcceptBackCompatBreaks: running command %v\n", execCmd)
	}

	err := execCmd.Run()
	if err != nil {
		_, _ = fmt.Fprintf(cmdParams.Stderr, "Failed to accept conjure backcompat breaks for project %q\n", project)
		return fmt.Errorf("failed to execute accept conjure backcompat breaks on project %q", project)
	}

	return nil
}
