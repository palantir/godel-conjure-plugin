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

package v1_test

import (
	"testing"

	v1 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v1"
	v2 "github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config/internal/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestSingleConjureConfigToV2(t *testing.T) {
	for _, tc := range []struct {
		name        string
		v1proj      v1.SingleConjureConfig
		projectName string
		want        v2.SingleConjureConfig
	}{
		{
			name: "exact v2 default match",
			v1proj: v1.SingleConjureConfig{
				OutputDir: "internal/generated/conjure/api",
				IRLocator: v1.IRLocatorConfig{
					Type:    v1.LocatorTypeYAML,
					Locator: "./conjure/api.yml",
				},
			},
			projectName: "api",
			want: v2.SingleConjureConfig{
				// OutputDir empty = uses v2 default
				IRLocator: v2.IRLocatorConfig{
					Type:    v2.LocatorTypeYAML,
					Locator: "./conjure/api.yml",
				},
				// Always skip delete when converting from v1 to preserve v1 behavior
				SkipDeleteGeneratedFiles: true,
			},
		},
		{
			name: "project name match optimization",
			v1proj: v1.SingleConjureConfig{
				OutputDir: "mag-api",
				IRLocator: v1.IRLocatorConfig{
					Type:    v1.LocatorTypeYAML,
					Locator: "./conjure/mag-api.yml",
				},
			},
			projectName: "mag-api",
			want: v2.SingleConjureConfig{
				OutputDir: ".",
				IRLocator: v2.IRLocatorConfig{
					Type:    v2.LocatorTypeYAML,
					Locator: "./conjure/mag-api.yml",
				},
				// Skip delete for safety, but allow project name appending
				SkipDeleteGeneratedFiles: true,
			},
		},
		{
			name: "custom path with escape valves",
			v1proj: v1.SingleConjureConfig{
				OutputDir: "custom/output",
				IRLocator: v1.IRLocatorConfig{
					Type:    v1.LocatorTypeYAML,
					Locator: "./api.yml",
				},
			},
			projectName: "myproject",
			want: v2.SingleConjureConfig{
				OutputDir: "custom/output",
				IRLocator: v2.IRLocatorConfig{
					Type:    v2.LocatorTypeYAML,
					Locator: "./api.yml",
				},
				OmitTopLevelProjectDir:   true,
				SkipDeleteGeneratedFiles: true,
			},
		},
		{
			name: "empty output-dir with escape valves",
			v1proj: v1.SingleConjureConfig{
				OutputDir: "",
				IRLocator: v1.IRLocatorConfig{
					Type:    v1.LocatorTypeYAML,
					Locator: "./api.yml",
				},
			},
			projectName: "myproject",
			want: v2.SingleConjureConfig{
				OutputDir: ".",
				IRLocator: v2.IRLocatorConfig{
					Type:    v2.LocatorTypeYAML,
					Locator: "./api.yml",
				},
				OmitTopLevelProjectDir:   true,
				SkipDeleteGeneratedFiles: true,
			},
		},
		{
			name: "dot output-dir with escape valves",
			v1proj: v1.SingleConjureConfig{
				OutputDir: ".",
				IRLocator: v1.IRLocatorConfig{
					Type:    v1.LocatorTypeYAML,
					Locator: "./api.yml",
				},
			},
			projectName: "myproject",
			want: v2.SingleConjureConfig{
				OutputDir: ".",
				IRLocator: v2.IRLocatorConfig{
					Type:    v2.LocatorTypeYAML,
					Locator: "./api.yml",
				},
				OmitTopLevelProjectDir:   true,
				SkipDeleteGeneratedFiles: true,
			},
		},
		{
			name: "preserves all other fields",
			v1proj: v1.SingleConjureConfig{
				OutputDir: "internal/generated/conjure/api",
				IRLocator: v1.IRLocatorConfig{
					Type:    v1.LocatorTypeRemote,
					Locator: "https://example.com/ir.json",
				},
				GroupID:     "com.palantir.test",
				Publish:     boolPtr(true),
				Server:      true,
				CLI:         true,
				AcceptFuncs: boolPtr(false),
				Extensions: map[string]any{
					"foo": "bar",
				},
			},
			projectName: "api",
			want: v2.SingleConjureConfig{
				IRLocator: v2.IRLocatorConfig{
					Type:    v2.LocatorTypeRemote,
					Locator: "https://example.com/ir.json",
				},
				GroupID:     "com.palantir.test",
				Publish:     boolPtr(true),
				Server:      true,
				CLI:         true,
				AcceptFuncs: boolPtr(false),
				Extensions: map[string]any{
					"foo": "bar",
				},
				// Always skip delete when converting from v1 to preserve v1 behavior
				SkipDeleteGeneratedFiles: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v1proj.ToV2(tc.projectName)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConjurePluginConfigToV2(t *testing.T) {
	for _, tc := range []struct {
		name  string
		v1cfg v1.ConjurePluginConfig
		want  v2.ConjurePluginConfig
	}{
		{
			name: "single project with clean upgrade",
			v1cfg: v1.ConjurePluginConfig{
				GroupID: "com.palantir.test",
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"api": {
						OutputDir: "internal/generated/conjure/api",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./api.yml",
						},
					},
				},
			},
			want: v2.ConjurePluginConfig{
				GroupID: "com.palantir.test",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"api": {
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./api.yml",
						},
						// Always skip delete when converting from v1 to preserve v1 behavior
						SkipDeleteGeneratedFiles: true,
					},
				},
				// No AllowConflictingOutputDirs (no conflicts)
			},
		},
		{
			name: "multiple projects with conflicts",
			v1cfg: v1.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"api-v1": {
						OutputDir: "shared",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./api-v1.yml",
						},
					},
					"api-v2": {
						OutputDir: "shared",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./api-v2.yml",
						},
					},
				},
			},
			want: v2.ConjurePluginConfig{
				AllowConflictingOutputDirs: true, // Conflict detected
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"api-v1": {
						OutputDir: "shared",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./api-v1.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"api-v2": {
						OutputDir: "shared",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./api-v2.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "multiple projects without conflicts",
			v1cfg: v1.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"api": {
						OutputDir: "internal/generated/conjure/api",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./api.yml",
						},
					},
					"backend": {
						OutputDir: "internal/generated/conjure/backend",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./backend.yml",
						},
					},
				},
			},
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"api": {
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./api.yml",
						},
						// Always skip delete when converting from v1 to preserve v1 behavior
						SkipDeleteGeneratedFiles: true,
					},
					"backend": {
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./backend.yml",
						},
						// Always skip delete when converting from v1 to preserve v1 behavior
						SkipDeleteGeneratedFiles: true,
					},
				},
				// No AllowConflictingOutputDirs (no conflicts)
			},
		},
		{
			name: "multiple projects with parent-child directory relationship",
			v1cfg: v1.ConjurePluginConfig{
				ProjectConfigs: map[string]v1.SingleConjureConfig{
					"parent": {
						OutputDir: "generated",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./parent.yml",
						},
					},
					"child": {
						OutputDir: "generated/subdir",
						IRLocator: v1.IRLocatorConfig{
							Type:    v1.LocatorTypeYAML,
							Locator: "./child.yml",
						},
					},
				},
			},
			want: v2.ConjurePluginConfig{
				AllowConflictingOutputDirs: true, // Parent-child conflict detected
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"parent": {
						OutputDir: "generated",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./parent.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"child": {
						OutputDir: "generated/subdir",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeYAML,
							Locator: "./child.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v1cfg.ToV2()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUpgradeConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		v1Config string
		want     v2.ConjurePluginConfig
	}{
		{
			name: "v1 config with custom output-dir gets escape valves",
			v1Config: `version: 1
projects:
  myproject:
    output-dir: custom/output
    ir-locator: ./conjure/api.yml
`,
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"myproject": {
						OutputDir: "custom/output",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./conjure/api.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config with output-dir matching v2 default (with project name) upgrades cleanly",
			v1Config: `version: 1
projects:
  api:
    output-dir: internal/generated/conjure/api
    ir-locator: ./conjure/api.yml
`,
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"api": {
						// OutputDir is empty, which defaults to v2.DefaultOutputDir
						// and project name will be appended
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./conjure/api.yml",
						},
						// Always skip delete when converting from v1 to preserve v1 behavior
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config with output-dir as internal/generated/conjure (without project name) gets escape valves",
			v1Config: `version: 1
projects:
  myservice:
    output-dir: internal/generated/conjure
    ir-locator: ./conjure/myservice.yml
`,
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"myservice": {
						// OutputDir empty (defaults to v2.DefaultOutputDir)
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./conjure/myservice.yml",
						},
						// Escape valves to preserve v1 behavior (generate directly to internal/generated/conjure)
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config with empty output-dir gets escape valves (v1 default was '.')",
			v1Config: `version: 1
projects:
  legacy:
    output-dir: ""
    ir-locator: ./api.yml
`,
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"legacy": {
						OutputDir: ".",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./api.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config with dot output-dir gets escape valves (same as empty)",
			v1Config: `version: 1
projects:
  myproject:
    output-dir: .
    ir-locator: ./api.yml
`,
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"myproject": {
						OutputDir: ".",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./api.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config where output-dir matches project name uses partial cleanup",
			v1Config: `version: 1
projects:
  mag-api:
    output-dir: mag-api
    ir-locator: ./api.yml
`,
			want: v2.ConjurePluginConfig{
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"mag-api": {
						OutputDir: ".",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./api.yml",
						},
						// Skip delete for safety, but allow project name appending
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config with conflicting output dirs sets AllowConflictingOutputDirs",
			v1Config: `version: 1
projects:
  project1:
    output-dir: shared
    ir-locator: ./api1.yml
  project2:
    output-dir: shared
    ir-locator: ./api2.yml
`,
			want: v2.ConjurePluginConfig{
				AllowConflictingOutputDirs: true,
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"project1": {
						OutputDir: "shared",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./api1.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
					"project2": {
						OutputDir: "shared",
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./api2.yml",
						},
						OmitTopLevelProjectDir:   true,
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
		{
			name: "v1 config preserves group-id, publish, server, cli, accept-funcs, extensions",
			v1Config: `version: 1
group-id: com.palantir.test
projects:
  api:
    output-dir: internal/generated/conjure/api
    ir-locator: ./api.yml
    group-id: com.palantir.override
    publish: true
    server: true
    cli: true
    accept-funcs: false
    extensions:
      foo: bar
`,
			want: v2.ConjurePluginConfig{
				GroupID: "com.palantir.test",
				ProjectConfigs: map[string]v2.SingleConjureConfig{
					"api": {
						IRLocator: v2.IRLocatorConfig{
							Type:    v2.LocatorTypeAuto,
							Locator: "./api.yml",
						},
						GroupID:     "com.palantir.override",
						Publish:     boolPtr(true),
						Server:      true,
						CLI:         true,
						AcceptFuncs: boolPtr(false),
						Extensions: map[string]any{
							"foo": "bar",
						},
						// Always skip delete when converting from v1 to preserve v1 behavior
						SkipDeleteGeneratedFiles: true,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Perform upgrade
			upgradedBytes, err := v1.UpgradeConfig([]byte(tc.v1Config))
			require.NoError(t, err)

			// Unmarshal result
			var got v2.ConjurePluginConfig
			err = yaml.UnmarshalStrict(upgradedBytes, &got)
			require.NoError(t, err)

			// Version should always be "2"
			assert.Equal(t, "2", got.Version)
			// Set version to empty for comparison (tc.want doesn't include it)
			got.Version = ""

			assert.Equal(t, tc.want, got)
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
