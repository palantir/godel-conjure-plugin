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
	"github.com/palantir/godel-conjure-plugin/v6/assetapi"
	assetapiinternal "github.com/palantir/godel-conjure-plugin/v6/internal/assetapi"
	"github.com/spf13/cobra"
)

// BackCompatParams contains the parameters for backcompat checking operations.
type BackCompatParams struct {
	GroupID         string
	Project         string
	CurrentIR       string
	GodelProjectDir string
}

// Checker is an interface for checking backward compatibility of Conjure definitions.
// Implementations of this interface can be used with NewCheckerRootCommand to create
// a CLI-based backcompat asset that conforms to the backcompat asset API.
type Checker interface {
	// CheckBackCompat checks backward compatibility for the given parameters.
	// It should return an error if backcompat breaks are found or if the check fails to execute.
	CheckBackCompat(params BackCompatParams) error

	// AcceptBackCompatBreaks accepts backward compatibility breaks for the given parameters.
	// It should only return an error if the operation fails to execute, not if breaks are found.
	AcceptBackCompatBreaks(params BackCompatParams) error
}

// NewCheckerRootCommand creates a Cobra root command for an asset that checks backward compatibility.
// The returned command has the required subcommands hooked up for reporting the asset type and performing
// backcompat checks according to the asset spec using the provided checker as its implementation.
func NewCheckerRootCommand(name, description string, checker Checker) *cobra.Command {
	rootCmd := assetapiinternal.NewAssetRootCmd(assetapi.ConjureBackCompat, name, description)

	checkCmd := newCheckBackCompatCmd(checker)
	rootCmd.AddCommand(checkCmd)

	acceptCmd := newAcceptBackCompatBreaksCmd(checker)
	rootCmd.AddCommand(acceptCmd)

	return rootCmd
}

const (
	CheckBackCompatCommand        = "check-backcompat"
	AcceptBackCompatBreaksCommand = "accept-backcompat-breaks"

	groupIDFlagName         = "group-id"
	projectFlagName         = "project"
	currentIRFlagName       = "current-ir"
	godelProjectDirFlagName = "godel-project-dir"
)

func newCheckBackCompatCmd(checker Checker) *cobra.Command {
	var params BackCompatParams
	cmd := &cobra.Command{
		Use:   CheckBackCompatCommand,
		Short: "Check backward compatibility of Conjure definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checker.CheckBackCompat(params)
		},
	}
	addBackCompatFlags(cmd, &params)
	return cmd
}

func newAcceptBackCompatBreaksCmd(checker Checker) *cobra.Command {
	var params BackCompatParams
	cmd := &cobra.Command{
		Use:   AcceptBackCompatBreaksCommand,
		Short: "Accept current backward compatibility breaks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checker.AcceptBackCompatBreaks(params)
		},
	}
	addBackCompatFlags(cmd, &params)

	return cmd
}

func addBackCompatFlags(cmd *cobra.Command, params *BackCompatParams) {
	cmd.Flags().StringVar(&params.GroupID, groupIDFlagName, "", "Group ID of the Conjure project")
	cmd.Flags().StringVar(&params.Project, projectFlagName, "", "Name of the Conjure project")
	cmd.Flags().StringVar(&params.CurrentIR, currentIRFlagName, "", "Path to the current Conjure IR file")
	cmd.Flags().StringVar(&params.GodelProjectDir, godelProjectDirFlagName, "", "Path to the godel project directory")
	markFlagsRequired(cmd, groupIDFlagName, projectFlagName, currentIRFlagName, godelProjectDirFlagName)
}

func markFlagsRequired(cmd *cobra.Command, flagNames ...string) {
	for _, currFlagName := range flagNames {
		if err := cmd.MarkFlagRequired(currFlagName); err != nil {
			panic(err)
		}
	}
}
