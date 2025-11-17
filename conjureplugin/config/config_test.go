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
	v2 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v2"
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeRemote,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeIRFile,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeRemote,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeRemote,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeRemote,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "input.yml",
						},
						OmitTopLevelProjectDir: true,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "input.json",
						},
						AcceptFuncs:            toPtr(true),
						OmitTopLevelProjectDir: true,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "input.json",
						},
						OmitTopLevelProjectDir: true,
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
		got, _, err := tc.in.ToParams()
		require.NoError(t, err, "Case %d", i)
		assert.Equal(t, tc.want, got, "Case %d", i)
	}
}

func TestConjurePluginConfigToParam_Warnings(t *testing.T) {
	for i, tc := range []struct {
		name         string
		in           config.ConjurePluginConfig
		want         conjureplugin.ConjureProjectParams
		wantWarnings []string
	}{
		{
			name: "No warnings for single project",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
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
					},
				},
			},
			wantWarnings: nil,
		},
		{
			name: "No warnings for multiple projects with different output directories",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
					},
					"project-2": {
						OutputDir: "outputDir-2",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
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
						OutputDir:   "outputDir",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:     true,
						AcceptFuncs: true,
					},
					"project-2": {
						OutputDir:   "outputDir-2",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("local-2/yaml-dir"),
						Publish:     true,
						AcceptFuncs: true,
					},
				},
			},
			wantWarnings: nil,
		},
		{
			name: "Warning for multiple projects with the same output directory",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: true,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
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
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-2/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantWarnings: []string{
				`project "project-1" and "project-2" have the same output directory "outputDir"`,
				`project "project-2" and "project-1" have the same output directory "outputDir"`,
			},
		},
		{
			name: "Warning for multiple projects with the same output directory after normalization",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: true,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "./outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
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
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-2/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantWarnings: []string{
				`project "project-1" and "project-2" have the same output directory "outputDir"`,
				`project "project-2" and "project-1" have the same output directory "outputDir"`,
			},
		},
		{
			name: "Multiple warnings for multiple projects with the same output directory after normalization",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: true,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "./outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-3": {
						OutputDir: "outputDir-other",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-3/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-4": {
						OutputDir: "outputDir-other/",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-4/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			want: conjureplugin.ConjureProjectParams{
				SortedKeys: []string{
					"project-1",
					"project-2",
					"project-3",
					"project-4",
				},
				Params: map[string]conjureplugin.ConjureProjectParam{
					"project-1": {
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-2/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-3": {
						OutputDir:                "outputDir-other",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-3/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-4": {
						OutputDir:                "outputDir-other",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-4/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantWarnings: []string{
				`project "project-1" and "project-2" have the same output directory "outputDir"`,
				`project "project-2" and "project-1" have the same output directory "outputDir"`,
				`project "project-3" and "project-4" have the same output directory "outputDir-other"`,
				`project "project-4" and "project-3" have the same output directory "outputDir-other"`,
			},
		},
		{
			name: "Warning for parent-child directory relationship",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: true,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "outputDir/subdir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
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
						OutputDir:                "outputDir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir:                "outputDir/subdir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-2/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantWarnings: []string{
				`output directory "outputDir/subdir" of project "project-2" is a subdirectory of output directory "outputDir" of project "project-1"`,
			},
		},
		{
			name: "Warning for parent-child directory relationship with normalization",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: true,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "base/dir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "./base/dir/../dir/nested",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
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
						OutputDir:                "base/dir",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir:                "base/dir/nested",
						IRProvider:               conjureplugin.NewLocalYAMLIRProvider("local-2/yaml-dir"),
						Publish:                  true,
						AcceptFuncs:              true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantWarnings: []string{
				`output directory "base/dir/nested" of project "project-2" is a subdirectory of output directory "base/dir" of project "project-1"`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, gotWarnings, err := tc.in.ToParams()
			require.NoError(t, err, "Case %d", i)
			assert.Equal(t, tc.want, got, "Case %d", i)

			assert.Equal(t, len(tc.wantWarnings), len(gotWarnings), "Case %d", i)
			for i := range tc.wantWarnings {
				assert.EqualError(t, gotWarnings[i], tc.wantWarnings[i], "Case %d", i)
			}
		})
	}
}

func TestConjurePluginConfigToParam_Errors(t *testing.T) {
	for i, tc := range []struct {
		name      string
		in        config.ConjurePluginConfig
		wantError string
	}{
		{
			name: "Error for same output directory when conflicts not allowed",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: false,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantError: "output directory conflicts detected: project \"project-1\" and \"project-2\" have the same output directory \"outputDir\"\nproject \"project-2\" and \"project-1\" have the same output directory \"outputDir\"",
		},
		{
			name: "Error for parent-child directory relationship when conflicts not allowed",
			in: config.ConjurePluginConfig{
				AllowConflictingOutputDirs: false,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "base/dir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project-2": {
						OutputDir: "base/dir/nested",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantError: "output directory conflicts detected: output directory \"base/dir/nested\" of project \"project-2\" is a subdirectory of output directory \"base/dir\" of project \"project-1\"",
		},
		{
			name: "Error when attempting to delete with same output directory",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: false,
					},
					"project-2": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: false,
					},
				},
			},
			wantError: "project \"project-1\" cannot delete generated files when output directories conflict\nproject \"project-1\" and \"project-2\" have the same output directory \"outputDir\"",
		},
		{
			name: "Error when attempting to delete with nested output directory",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "base/dir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: false,
					},
					"project-2": {
						OutputDir: "base/dir/nested",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: false,
					},
				},
			},
			wantError: "project \"project-1\" cannot delete generated files when output directories conflict\noutput directory \"base/dir/nested\" of project \"project-2\" is a subdirectory of output directory \"base/dir\" of project \"project-1\"",
		},
		{
			name: "Error when attempting to delete with one project having skip=false and conflicts exist",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: false,
					},
					"project-2": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local-2/yaml-dir",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
			wantError: "project \"project-1\" cannot delete generated files when output directories conflict\nproject \"project-1\" and \"project-2\" have the same output directory \"outputDir\"",
		},
		{
			name: "Error for invalid project name with forward slash",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project/invalid": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
					},
				},
			},
			wantError: `project name "project/invalid" cannot contain path separators (/ or \)`,
		},
		{
			name: "Error for invalid project name with backslash",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project\\invalid": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
					},
				},
			},
			wantError: `project name "project\\invalid" cannot contain path separators (/ or \)`,
		},
		{
			name: "Error for project name that is dot",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					".": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
					},
				},
			},
			wantError: `project name cannot be "."`,
		},
		{
			name: "Error for project name that is double dot",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"..": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
					},
				},
			},
			wantError: `project name cannot be ".."`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, gotWarnings, err := tc.in.ToParams()
			require.Error(t, err, "Case %d", i)
			assert.EqualError(t, err, tc.wantError, "Case %d", i)
			assert.Empty(t, gotWarnings, "Case %d", i)
			assert.Empty(t, got.Params, "Case %d", i)
		})
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
				GroupID: "com.palantir.signals",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
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
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						GroupID: "com.palantir.project",
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
				GroupID: "com.palantir.default",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir1",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir1",
						},
					},
					"project-2": {
						OutputDir: "outputDir2",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir2",
						},
						GroupID: "com.palantir.override",
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
				GroupID: "com.palantir.signals",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
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
						GroupID:     "com.palantir.signals",
					},
				},
			},
		},
		{
			name: "per-project group-id overrides top-level",
			in: config.ConjurePluginConfig{
				GroupID: "com.palantir.default",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						GroupID:                "com.palantir.override",
						OmitTopLevelProjectDir: true,
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
						GroupID:     "com.palantir.override",
					},
				},
			},
		},
		{
			name: "no group-id specified",
			in: config.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "local/yaml-dir",
						},
						OmitTopLevelProjectDir: true,
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
						GroupID:     "",
					},
				},
			},
		},
		{
			name: "multiple projects with different group-ids",
			in: config.ConjurePluginConfig{
				GroupID: "com.palantir.default",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project-1": {
						OutputDir: "outputDir1",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "input1.yml",
						},
						OmitTopLevelProjectDir: true,
					},
					"project-2": {
						OutputDir: "outputDir2",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "input2.yml",
						},
						GroupID:                "com.palantir.custom",
						OmitTopLevelProjectDir: true,
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
						GroupID:     "com.palantir.default",
					},
					"project-2": {
						OutputDir:   "outputDir2",
						IRProvider:  conjureplugin.NewLocalYAMLIRProvider("input2.yml"),
						Publish:     true,
						AcceptFuncs: true,
						GroupID:     "com.palantir.custom",
					},
				},
			},
		},
	} {
		got, _, err := tc.in.ToParams()
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		assert.Equal(t, tc.want, got, "Case %d: %s", i, tc.name)
	}
}

func toPtr[T any](in T) *T {
	return &in
}
