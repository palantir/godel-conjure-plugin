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
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

// BackCompatChecker represents a wrapper around a backcompat asset executable.
type BackCompatChecker interface {
	// CheckBackCompat runs the asset's backcompat check for the specified project.
	// It executes the asset as a command-line tool with the relevant arguments.
	// If the command exits with code 1, it indicates backcompat breaks were found and returns an error specific to that case.
	// Any other execution errors are wrapped and returned.
	CheckBackCompat(groupID, project string, currentIR string, godelProjectDir string) error

	// AcceptBackCompatBreaks runs the asset's backcompat check for the specified project,
	// but only returns an error if the command fails to execute, not if backcompat breaks are found.
	// This is used to accept and record the presence of backcompat breaks.
	AcceptBackCompatBreaks(groupID, project string, currentIR string, godelProjectDir string) error
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
) error {
	cmd := exec.Command(b.asset,
		"check-backcompat",
		"--group-id", groupID,
		"--project", project,
		"--current-ir", currentIR,
		"--godel-project-dir", godelProjectDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		return nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return fmt.Errorf(`conjure breaks found in project %s\nto accept breaks run "./godelw conjure-accept-backcompat-breaks"`, project)
	}

	return errors.Wrapf(err, "failed to execute check conjure backcompat on project %q", project)
}

func (b *backCompatCheckerImpl) AcceptBackCompatBreaks(
	groupID, project string,
	currentIR string,
	godelProjectDir string,
) error {
	cmd := exec.Command(b.asset,
		"accept-backcompat-breaks",
		"--group-id", groupID,
		"--project", project,
		"--current-ir", currentIR,
		"--godel-project-dir", godelProjectDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to execute accept conjure backcompat breaks on project %q", project)
	}

	return nil
}
