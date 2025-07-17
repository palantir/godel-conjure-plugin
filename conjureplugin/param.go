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
	"sort"
)

type ConjureProjectParams struct {
	Params map[string]ConjureProjectParam
}

type Param struct {
	Param ConjureProjectParam
	Key   string
}

func (p *ConjureProjectParams) OrderedParams() []Param {
	var keys []string
	for key := range p.Params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var out []Param
	for _, k := range keys {
		out = append(out, Param{
			Param: p.Params[k],
			Key:   k,
		})
	}
	return out
}

type ConjureProjectParam struct {
	OutputDir    string
	IRProvider   IRProvider
	IROutputPath string
	// Server will optionally generate server code in addition to client code for services specified in this project.
	Server bool
	// CLI will optionally generate cobra CLI bindings in addition to client code for services specified in this project.
	CLI bool
	// AcceptFuncs will optionally generate lambda based visitor code for unions specified in this project.
	AcceptFuncs bool
	// Publish specifies whether or not this Conjure project should be included in the "publish" operation.
	Publish bool
}
