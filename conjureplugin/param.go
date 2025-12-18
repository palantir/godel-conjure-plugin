// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package conjureplugin

import (
	"io"
)

// CmdParams specifies the parameters for executing logic within the context of a command. Specifies the output streams
// and debug configuration.
type CmdParams struct {
	Stdout io.Writer
	Stderr io.Writer
	Debug  bool
}

type ConjureProjectParams []ConjureProjectParam

type ConjureProjectParam struct {
	// The name of the Conjure project. Must be unique across projects.
	ProjectName  string
	OutputDir    string
	IRProvider   IRProvider
	IROutputPath string
	// GroupID is the group ID for this project
	GroupID string
	// Server will optionally generate server code in addition to client code for services specified in this project.
	Server bool
	// CLI will optionally generate cobra CLI bindings in addition to client code for services specified in this project.
	CLI bool
	// AcceptFuncs will optionally generate lambda based visitor code for unions specified in this project.
	AcceptFuncs bool
	// Publish specifies whether or not this Conjure project should be included in the "publish" operation.
	Publish bool
	// SkipConjureBackcompat specifies whether or not backcompat checks should be skipped for this Conjure project.
	SkipConjureBackcompat bool
	// SkipDeleteGeneratedFiles skips cleanup of old generated files before regeneration.
	// When false (default), deletes all Conjure-generated files in the output directory tree before regenerating.
	// When true, preserves v1 behavior (no cleanup).
	SkipDeleteGeneratedFiles bool
	// CGRModuleVersion specifies which module version of conjure-go-runtime to use in generated code.
	// If not specified, does not specify a version and uses the default chosen by the conjure-go generator
	CGRModuleVersion int `yaml:"cgr-module-version,omitempty"`
	// WGSModuleVersion specifies which module version of witchcraft-go-server to use in generated code.
	// If not specified, does not specify a version and uses the default chosen by the conjure-go generator
	WGSModuleVersion int `yaml:"wgs-module-version,omitempty"`
}
