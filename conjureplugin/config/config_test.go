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
	"io"
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
						Publish: toPtr(false),
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
						Publish: toPtr(true),
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
   cli: true
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
						CLI:    true,
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
						AcceptFuncs: toPtr(true),
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
   extensions:
     foo: bar
     baz:
       - 1
       - 2
     blah:
       key: value
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
						AcceptFuncs: toPtr(true),
						Extensions: map[string]any{
							"foo":  "bar",
							"baz":  []any{1, 2},
							"blah": map[any]any{"key": "value"},
						},
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
						AcceptFuncs: toPtr(true),
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
		got, err := tc.in.ToParams(io.Discard)
		require.NoError(t, err, "Case %d", i)
		assert.Equal(t, tc.want, got, "Case %d", i)
	}
}

func TestGroupIDConfiguration(t *testing.T) {
	for i, tc := range []struct {
		name string
		in   string
		want config.ConjurePluginConfig
	}{
		{
			name: "top-level group-id only",
			in: `
group-id: com.palantir.signals
projects:
  project:
    output-dir: outputDir
    ir-locator: local/yaml-dir
`,
			want: config.ConjurePluginConfig{
				GroupID: toPtr("com.palantir.signals"),
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
			name: "per-project group-id only",
			in: `
projects:
  project:
    output-dir: outputDir
    ir-locator: local/yaml-dir
    group-id: com.palantir.project
`,
			want: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						GroupID: toPtr("com.palantir.project"),
					},
				},
			},
		},
		{
			name: "both top-level and per-project group-id",
			in: `
group-id: com.palantir.default
projects:
  project-1:
    output-dir: outputDir1
    ir-locator: local/yaml-dir1
  project-2:
    output-dir: outputDir2
    ir-locator: local/yaml-dir2
    group-id: com.palantir.override
`,
			want: config.ConjurePluginConfig{
				GroupID: toPtr("com.palantir.default"),
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir1",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir1",
						},
					},
					"project-2": {
						OutputDir: "outputDir2",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir2",
						},
						GroupID: toPtr("com.palantir.override"),
					},
				},
			},
		},
	} {
		var got config.ConjurePluginConfig
		err := yaml.Unmarshal([]byte(tc.in), &got)
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		assert.Equal(t, tc.want, got, "Case %d: %s", i, tc.name)
	}
}

func TestGroupIDToParams(t *testing.T) {
	for i, tc := range []struct {
		name string
		in   config.ConjurePluginConfig
		want conjureplugin.ConjureProjectParams
	}{
		{
			name: "top-level group-id is inherited by project",
			in: config.ConjurePluginConfig{
				GroupID: toPtr("com.palantir.signals"),
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
			want: conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:     true,
						AcceptFuncs: true,
						GroupID:     toPtr("com.palantir.signals"),
					},
				},
			},
		},
		{
			name: "per-project group-id overrides top-level",
			in: config.ConjurePluginConfig{
				GroupID: toPtr("com.palantir.default"),
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						GroupID: toPtr("com.palantir.override"),
					},
				},
			},
			want: conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:     true,
						AcceptFuncs: true,
						GroupID:     toPtr("com.palantir.override"),
					},
				},
			},
		},
		{
			name: "no group-id specified",
			in: config.ConjurePluginConfig{
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
			want: conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:     true,
						AcceptFuncs: true,
						GroupID:     nil,
					},
				},
			},
		},
		{
			name: "multiple projects with different group-ids",
			in: config.ConjurePluginConfig{
				GroupID: toPtr("com.palantir.default"),
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir1",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "input1.yml",
						},
					},
					"project-2": {
						OutputDir: "outputDir2",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeAuto,
							Locator: "input2.yml",
						},
						GroupID: toPtr("com.palantir.custom"),
					},
				},
			},
			want: conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
					"project-2",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:   "outputDir1",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("input1.yml"),
						Publish:     true,
						AcceptFuncs: true,
						GroupID:     toPtr("com.palantir.default"),
					},
					"project-2": {
						OutputDir:   "outputDir2",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("input2.yml"),
						Publish:     true,
						AcceptFuncs: true,
						GroupID:     toPtr("com.palantir.custom"),
					},
				},
			},
		},
	} {
		got, err := tc.in.ToParams(io.Discard)
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		assert.Equal(t, tc.want, got, "Case %d: %s", i, tc.name)
	}
}

func toPtr[T any](in T) *T {
	return &in
}
