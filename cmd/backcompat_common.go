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

package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	backcompatvalidator "github.com/palantir/godel-conjure-plugin/v6/internal/backcompat-validator"
	pkgerrors "github.com/pkg/errors"
)

// runBackcompatOperation executes a backcompat operation (check or accept) on projects.
// It collects all errors before returning, allowing all projects to be processed.
func runBackcompatOperation(
	stdout io.Writer,
	projectFlag string,
	operation func(asset *backcompatvalidator.BackCompatAsset, projectName string, param conjureplugin.ConjureProjectParam, projectDir string) error,
) error {
	parsedConfigSet, err := toProjectParams(configFileFlag, stdout)
	if err != nil {
		return err
	}
	if err := os.Chdir(projectDirFlag); err != nil {
		return pkgerrors.Wrapf(err, "failed to set working directory")
	}

	asset := backcompatvalidator.New(configFileFlag, assetsFlag)

	if projectFlag != "" {
		// Run operation for specific project
		param, ok := parsedConfigSet.Params[projectFlag]
		if !ok {
			return pkgerrors.Errorf("project %s not found in configuration", projectFlag)
		}
		return operation(asset, projectFlag, param, projectDirFlag)
	}

	// Run operation for all projects, collecting errors
	type projectError struct {
		projectName string
		err         error
	}
	var failures []projectError

	for projectName, param := range parsedConfigSet.Params {
		if opErr := operation(asset, projectName, param, projectDirFlag); opErr != nil {
			failures = append(failures, projectError{
				projectName: projectName,
				err:         opErr,
			})
		}
	}

	if len(failures) == 0 {
		return nil
	}

	// Format a clear error message showing all failed projects
	var errMsg strings.Builder
	if len(failures) == 1 {
		errMsg.WriteString(fmt.Sprintf("operation failed for project %s:\n", failures[0].projectName))
		errMsg.WriteString(fmt.Sprintf("  %v", failures[0].err))
	} else {
		errMsg.WriteString(fmt.Sprintf("operation failed for %d projects:\n", len(failures)))
		for _, failure := range failures {
			errMsg.WriteString(fmt.Sprintf("  - %s: %v\n", failure.projectName, failure.err))
		}
	}

	return pkgerrors.New(errMsg.String())
}
