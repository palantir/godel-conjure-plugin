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
	// Publish specifies whether or not this Conjure project should be included in the "publish" operation.
	Publish bool
}
