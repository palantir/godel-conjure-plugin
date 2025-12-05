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
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/internal/backcompatasset"
	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	backcompatCmdName = "backcompat"
)

var backcompatCmd = &cobra.Command{
	Use:   backcompatCmdName,
	Short: "Check backward compatibility of Conjure definitions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackCompatCommand(
			cmd,
			func(project string, param conjureplugin.ConjureProjectParam, irFile string, execParam backcompatasset.ExecParam) error {
				isBackwardsCompatible, err := loadedAssets.ConjureBackcompat.CheckBackCompat(param.GroupID, project, irFile, projectDirFlagVal, execParam)
				if err != nil {
					return err
				}
				if !isBackwardsCompatible {
					return fmt.Errorf("Conjure definition for project %q is not backwards compatible with latest published version", project)
				}
				return nil
			},
			func(failedProjects map[string]error) error {
				projects := slices.Collect(maps.Keys(failedProjects))
				sort.Strings(projects)

				errorMsg := fmt.Sprintf("Conjure projects had backwards compatibility issues: %s\n", strings.Join(projects, ", "))
				errorMsg += fmt.Sprintf("If the breaks are intentional please run `./godelw %s` to accept them", conjureAcceptBackCompatBreaksTaskName)
				return fmt.Errorf("%s", errorMsg)
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(backcompatCmd)
}

func runBackCompatCommand(
	cmd *cobra.Command,
	runCmd func(project string, param conjureplugin.ConjureProjectParam, irFile string, execParam backcompatasset.ExecParam) error,
	errorHandler func(map[string]error) error,
) error {
	if loadedAssets.ConjureBackcompat == nil {
		return nil
	}

	execParam := backcompatasset.ExecParam{
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.OutOrStderr(),
		Debug:  debugFlagVal,
	}

	projectParams, err := toProjectParams(configFileFlagVal, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	if err := os.Chdir(projectDirFlagVal); err != nil {
		return errors.Wrapf(err, "failed to set working directory")
	}

	if errs := projectParams.ForEach(func(project string, param conjureplugin.ConjureProjectParam) error {
		if param.SkipConjureBackcompat {
			return nil
		}
		if !param.IRProvider.GeneratedFromYAML() {
			return nil
		}

		bytes, err := param.IRProvider.IRBytes()
		if err != nil {
			return errors.Wrapf(err, "failed to generate IR bytes for project %q", project)
		}

		file, err := tempfilecreator.WriteBytesToTempFile(bytes)
		if err != nil {
			return errors.Wrapf(err, "failed to create temporary IR file for project %q", project)
		}

		return runCmd(project, param, file, execParam)
	}); len(errs) > 0 {
		return errorHandler(errs)
	}
	return nil
}
