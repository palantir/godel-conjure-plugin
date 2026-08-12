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

	"github.com/palantir/godel-conjure-plugin/v7/conjureplugin/config"
	v2 "github.com/palantir/godel-conjure-plugin/v7/conjureplugin/config/internal/v2"
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "local/yaml-dir",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "local/yaml-dir",
							},
							Publish: new(false),
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
     type: yaml
     locator: explicit/yaml-dir
`,
			config.ConjurePluginConfig{
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeYAML,
								Locator: "explicit/yaml-dir",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "http://foo.com/ir.json",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "http://foo.com/ir.json",
							},
							Publish: new(true),
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
`,
			config.ConjurePluginConfig{
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeRemote,
								Locator: "localhost:8080/ir.json",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "local/nonexistent-ir-file.json",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeIRFile,
								Locator: "local/nonexistent-ir-file.json",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeRemote,
								Locator: "localhost:8080/ir.json",
							},
							Server:      false,
							AcceptFuncs: new(true),
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeRemote,
								Locator: "localhost:8080/ir.json",
							},
							Server:      false,
							AcceptFuncs: new(true),
							Extensions: map[string]any{
								"foo":  "bar",
								"baz":  []any{1, 2},
								"blah": map[any]any{"key": "value"},
							},
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
   export-error-decoder: true
`,
			config.ConjurePluginConfig{
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeRemote,
								Locator: "localhost:8080/ir.json",
							},
							ExportErrorDecoder: true,
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
   error-parameter-format-json: true
`,
			config.ConjurePluginConfig{
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeRemote,
								Locator: "localhost:8080/ir.json",
							},
							ErrorParameterFormatJSON: true,
						},
					},
				},
			},
		},
		{
			`
projects:
  project:
   ir-locator: local/yaml-dir
   typescript:
     package-name: "@palantir/example-api"
     npm-version-scheme: generator-major
     npm-publisher-provider: couchdb
     flavorized-aliases: true
`,
			config.ConjurePluginConfig{
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "local/yaml-dir",
							},
							TypeScript: &v2.TypeScriptConfig{
								PackageName:          "@palantir/example-api",
								NpmVersionScheme:     "generator-major",
								NpmPublisherProvider: "couchdb",
								FlavorizedAliases:    true,
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "local/yaml-dir",
							},
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project",
						Config: v2.SingleConjureConfig{
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
				ProjectConfigs: v2.ConjureProjectConfigs{
					{
						Name: "project-1",
						Config: v2.SingleConjureConfig{
							OutputDir: "outputDir1",
							IRLocator: v2.IRLocatorConfig{
								Type:    v2.LocatorTypeAuto,
								Locator: "local/yaml-dir1",
							},
						},
					},
					{
						Name: "project-2",
						Config: v2.SingleConjureConfig{
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
		},
	} {
		var got config.ConjurePluginConfig
		err := yaml.Unmarshal([]byte(tc.in), &got)
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		assert.Equal(t, tc.want, got, "Case %d: %s", i, tc.name)
	}
}
