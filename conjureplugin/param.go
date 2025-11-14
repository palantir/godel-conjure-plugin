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
	"errors"

	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
)

type ConjureProjectParams struct {
	SortedKeys []string
	Params     map[string]ConjureProjectParam
}

func (p *ConjureProjectParams) OrderedParams() []ConjureProjectParam {
	var out []ConjureProjectParam
	for _, k := range p.SortedKeys {
		out = append(out, p.Params[k])
	}
	return out
}

type ConjureProjectParam struct {
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
}

// ForEach iterates over all project parameters in the order specified by SortedKeys,
// invoking the provided function for each project name and its associated parameter.
// It accumulates and returns any errors produced by the function calls using errors.Join.
func (p *ConjureProjectParams) ForEach(fn func(project string, param ConjureProjectParam) error) error {
	var err error

	for _, project := range p.SortedKeys {
		err = errors.Join(err, fn(project, p.Params[project]))
	}

	return err
}

// ForEachBackCompatProject iterates over all project parameters that should run backcompat checks
// (i.e., projects where SkipConjureBackcompat is false and IR is generated from YAML).
// For each eligible project, it generates the IR bytes, writes them to a temporary file,
// and invokes the provided function with the project name, parameter, and IR file path.
func (p *ConjureProjectParams) ForEachBackCompatProject(
	fn func(project string, param ConjureProjectParam, irFile string) error,
) error {
	return p.ForEach(func(project string, param ConjureProjectParam) error {
		if param.SkipConjureBackcompat {
			return nil
		}
		if !param.IRProvider.GeneratedFromYAML() {
			return nil
		}

		bytes, err := param.IRProvider.IRBytes()
		if err != nil {
			return err
		}

		file, err := tempfilecreator.WriteBytesToTempFile(bytes)
		if err != nil {
			return err
		}

		return fn(project, param, file)
	})
}
