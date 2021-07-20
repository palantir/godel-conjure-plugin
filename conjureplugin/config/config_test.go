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

package config_test

import (
	"testing"

	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config"
	v1 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestReadConfig(t *testing.T) {
	for i, tc := range []struct {
		in   string
		want config.ConjurePluginConfig
	}{
		{
			`
projects:
  project:
    output-dir: outputDir
    ir-locator: local/yaml-dir
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator: local/yaml-dir
   publish: false
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						Publish: boolPtr(false),
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator:
     type: yaml
     locator: explicit/yaml-dir
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "explicit/yaml-dir",
						},
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator: http://foo.com/ir.json
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "http://foo.com/ir.json",
						},
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator: http://foo.com/ir.json
   publish: true
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "http://foo.com/ir.json",
						},
						Publish: boolPtr(true),
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator:
     type: remote
     locator: localhost:8080/ir.json
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeRemote,
							Locator: "localhost:8080/ir.json",
						},
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator: local/nonexistent-ir-file.json
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/nonexistent-ir-file.json",
						},
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator:
     type: ir-file
     locator: local/nonexistent-ir-file.json
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeIRFile,
							Locator: "local/nonexistent-ir-file.json",
						},
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator:
     type: remote
     locator: localhost:8080/ir.json
   server: true
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeRemote,
							Locator: "localhost:8080/ir.json",
						},
						Server: true,
					},
				},
			},
		},
		{
			`
projects:
 project:
   output-dir: outputDir
   ir-locator:
     type: remote
     locator: localhost:8080/ir.json
   accept-funcs: true
`,
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeRemote,
							Locator: "localhost:8080/ir.json",
						},
						Server:      false,
						AcceptFuncs: boolPtr(true),
					},
				},
			},
		},
	} {
		var got config.ConjurePluginConfig
		err := yaml.Unmarshal([]byte(tc.in), &got)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, "Case %d", i)
	}
}

func TestConjurePluginConfigToParam(t *testing.T) {
	for i, tc := range []struct {
		in   config.ConjurePluginConfig
		want conjureplugin.ConjureProjectParams
	}{
		{
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
					},
				},
			},
			conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:     true,
						AcceptFuncs: true,
					},
				},
			},
		},
		{
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "input.yml",
						},
					},
				},
			},
			conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("input.yml"),
						Publish:     true,
						AcceptFuncs: true,
					},
				},
			},
		},
		{
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "input.json",
						},
						AcceptFuncs: boolPtr(true),
					},
				},
			},
			conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalFileIRProvider("input.json"),
						AcceptFuncs: true,
					},
				},
			},
		},
		{
			config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "input.json",
						},
					},
				},
			},
			conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalFileIRProvider("input.json"),
						AcceptFuncs: true,
					},
				},
			},
		},
	} {
		got, err := tc.in.ToParams()
		require.NoError(t, err, "Case %d", i)
		assert.Equal(t, tc.want, got, "Case %d", i)
	}
}

func boolPtr(in bool) *bool {
	return &in
}
