// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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
	"strings"
	"testing"

	"github.com/palantir/godel-conjure-plugin/v7/conjureplugin/config"
	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Existing config-package tests exercise the auto-detection path (empty/"auto" Type) extensively. This test covers
// the explicit locator-type values, which the auto-detection tests don't reach, plus the unknown-type error case.
func TestIRProviderFromLocatorConfig_ExplicitTypes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     config.IRLocatorConfig
		want    IRProvider
		wantErr string
	}{
		{
			name: "explicit remote",
			cfg:  config.IRLocatorConfig{Type: "remote", Locator: "http://example.com/api.conjure.json"},
			want: NewHTTPIRProvider("http://example.com/api.conjure.json"),
		},
		{
			name: "explicit yaml",
			cfg:  config.IRLocatorConfig{Type: "yaml", Locator: "some/yaml-dir"},
			want: NewLocalYAMLIRProvider("some/yaml-dir"),
		},
		{
			name: "explicit ir-file",
			cfg:  config.IRLocatorConfig{Type: "ir-file", Locator: "some/file.json"},
			want: NewLocalFileIRProvider("some/file.json"),
		},
		{
			name:    "unknown type",
			cfg:     config.IRLocatorConfig{Type: "bogus", Locator: "some/file.json"},
			wantErr: "unknown locator type: bogus",
		},
		{
			name:    "empty locator",
			cfg:     config.IRLocatorConfig{Type: "yaml", Locator: ""},
			wantErr: "locator cannot be empty",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := irProviderFromLocatorConfig(&tc.cfg)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateProjectName(t *testing.T) {
	for _, tc := range []struct {
		name        string
		projectName string
		wantError   string
	}{
		{
			name:        "valid project name",
			projectName: "my-project",
			wantError:   "",
		},
		{
			name:        "valid project name with underscores",
			projectName: "my_project",
			wantError:   "",
		},
		{
			name:        "valid project name with numbers",
			projectName: "project123",
			wantError:   "",
		},
		{
			name:        "valid project name with mixed characters",
			projectName: "my-project_v2",
			wantError:   "",
		},
		{
			name:        "invalid project name with forward slash",
			projectName: "my/project",
			wantError:   `project name "my/project" cannot contain path separators (/ or \)`,
		},
		{
			name:        "invalid project name with backslash",
			projectName: "my\\project",
			wantError:   `project name "my\\project" cannot contain path separators (/ or \)`,
		},
		{
			name:        "invalid project name with multiple forward slashes",
			projectName: "my/project/foo",
			wantError:   `project name "my/project/foo" cannot contain path separators (/ or \)`,
		},
		{
			name:        "invalid project name is dot",
			projectName: ".",
			wantError:   `project name cannot be "."`,
		},
		{
			name:        "invalid project name is double dot",
			projectName: "..",
			wantError:   `project name cannot be ".."`,
		},
		{
			name:        "valid project name starting with dot",
			projectName: ".hidden-project",
			wantError:   "",
		},
		{
			name:        "valid project name with spaces",
			projectName: "my project",
			wantError:   "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProjectName(tc.projectName)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
		})
	}
}

// Fixtures are built from YAML config text (rather than v2-package struct literals) and parsed via
// config.ReadConfigFromBytes, the same entry point cmd/run.go uses via ReadConfigFromFile, since this
// package cannot import conjureplugin/config/internal/v2.
func TestParamsFromConfig_GroupID(t *testing.T) {
	for _, tc := range []struct {
		name string
		yaml string
		want ConjureProjectParams
	}{
		{
			name: "top-level group-id is inherited by project",
			yaml: `
version: 2
group-id: com.palantir.signals
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
					GroupID:     "com.palantir.signals",
				},
			},
		},
		{
			name: "per-project group-id overrides top-level",
			yaml: `
version: 2
group-id: com.palantir.default
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
    group-id: com.palantir.override
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
					GroupID:     "com.palantir.override",
				},
			},
		},
		{
			name: "no group-id specified",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
					GroupID:     "",
				},
			},
		},
		{
			name: "multiple projects with different group-ids",
			yaml: `
version: 2
group-id: com.palantir.default
projects:
  project-1:
    output-dir: outputDir1
    omit-top-level-project-dir: true
    ir-locator: input1.yml
  project-2:
    output-dir: outputDir2
    omit-top-level-project-dir: true
    ir-locator: input2.yml
    group-id: com.palantir.custom
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir1",
					IRProvider:  NewLocalYAMLIRProvider("input1.yml"),
					Publish:     true,
					AcceptFuncs: true,
					GroupID:     "com.palantir.default",
				},
				{
					ProjectName: "project-2",
					OutputDir:   "outputDir2",
					IRProvider:  NewLocalYAMLIRProvider("input2.yml"),
					Publish:     true,
					AcceptFuncs: true,
					GroupID:     "com.palantir.custom",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.ReadConfigFromBytes([]byte(tc.yaml))
			require.NoError(t, err)

			got, _, err := ParamsFromConfig(&cfg)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParamsFromConfig_TypeScript(t *testing.T) {
	t.Run("defaults version scheme and generator service modes", func(t *testing.T) {
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript:
      package-name: "@palantir/example-api"
`))
		require.NoError(t, err)

		params, _, err := ParamsFromConfig(&cfg)
		require.NoError(t, err)
		require.Len(t, params, 1)
		assert.Equal(t, &TypeScriptParam{
			PackageName:              "@palantir/example-api",
			NpmVersionScheme:         NpmVersionSchemeGit,
			NpmPublisherProvider:     typescript.NpmPublisherProviderArtifactory,
			GenerateThrowingServices: true,
		}, params[0].TypeScript)
	})

	t.Run("accepts a scope containing characters npm disallows in the name", func(t *testing.T) {
		// npm's own package name validation permits "~" (and other encodeURIComponent-safe characters) in the scope
		// even though it disallows them in the name after the scope.
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript:
      package-name: "@scope~/example-api"
`))
		require.NoError(t, err)

		params, _, err := ParamsFromConfig(&cfg)
		require.NoError(t, err)
		require.Len(t, params, 1)
		assert.Equal(t, "@scope~/example-api", params[0].TypeScript.PackageName)
	})

	t.Run("preserves explicit values", func(t *testing.T) {
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript:
      package-name: "@palantir/example-api"
      npm-version-scheme: generator-major
      npm-publisher-provider: couchdb
      flavorized-aliases: true
      node-compatible-modules: true
      readonly-interfaces: true
      generate-throwing-services: false
      generate-non-throwing-services: true
`))
		require.NoError(t, err)

		params, _, err := ParamsFromConfig(&cfg)
		require.NoError(t, err)
		require.Len(t, params, 1)
		assert.Equal(t, &TypeScriptParam{
			PackageName:                 "@palantir/example-api",
			NpmVersionScheme:            NpmVersionSchemeGeneratorMajor,
			NpmPublisherProvider:        typescript.NpmPublisherProviderCouchDB,
			FlavorizedAliases:           true,
			NodeCompatibleModules:       true,
			ReadonlyInterfaces:          true,
			GenerateThrowingServices:    false,
			GenerateNonThrowingServices: true,
		}, params[0].TypeScript)
	})

	t.Run("absent block does not opt in", func(t *testing.T) {
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
`))
		require.NoError(t, err)

		params, _, err := ParamsFromConfig(&cfg)
		require.NoError(t, err)
		require.Len(t, params, 1)
		assert.Nil(t, params[0].TypeScript)
	})

	t.Run("missing package name", func(t *testing.T) {
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript: {}
`))
		require.NoError(t, err)

		_, _, err = ParamsFromConfig(&cfg)
		require.ErrorContains(t, err, "typescript package-name must be specified")
	})

	t.Run("unknown version scheme", func(t *testing.T) {
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript:
      package-name: "@palantir/example-api"
      npm-version-scheme: unknown
`))
		require.NoError(t, err)

		_, _, err = ParamsFromConfig(&cfg)
		require.ErrorContains(t, err, "npm-version-scheme must be one of")
	})

	t.Run("unknown publisher provider", func(t *testing.T) {
		cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript:
      package-name: "@palantir/example-api"
      npm-publisher-provider: unknown
`))
		require.NoError(t, err)

		_, _, err = ParamsFromConfig(&cfg)
		require.ErrorContains(t, err, "npm-publisher-provider must be one of")
	})

	for _, tc := range []struct {
		name        string
		packageName string
		wantError   string
	}{
		{
			name:        "unscoped package name",
			packageName: "example-api",
			wantError:   "must be a full scoped npm package name",
		},
		{
			name:        "empty scope",
			packageName: "@/example-api",
			wantError:   "must be a full scoped npm package name",
		},
		{
			name:        "scope with no package name",
			packageName: "@palantir/",
			wantError:   "must be a full scoped npm package name",
		},
		{
			name:        "invalid scope character",
			packageName: "@Palantir/example-api",
			wantError:   "invalid npm scope",
		},
		{
			name:        "invalid name character",
			packageName: "@palantir/Example-API",
			wantError:   "invalid npm name",
		},
		{
			name:        "name with an extra slash",
			packageName: "@palantir/example/api",
			wantError:   "invalid npm name",
		},
		{
			name:        "name starting with a dot",
			packageName: "@palantir/.example-api",
			wantError:   `must not have a name starting with "."`,
		},
		{
			name:        "name with a disallowed special character",
			packageName: "@palantir/example~api",
			wantError:   "invalid npm name",
		},
		{
			name:        "package name exceeds npm's length limit",
			packageName: "@palantir/" + strings.Repeat("a", 214),
			wantError:   "exceeds npm's 214-character length limit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.ReadConfigFromBytes([]byte(`
version: 2
projects:
  project-1:
    output-dir: outputDir
    ir-locator: input.yml
    typescript:
      package-name: "` + tc.packageName + `"
`))
			require.NoError(t, err)

			_, _, err = ParamsFromConfig(&cfg)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestParamsFromConfig(t *testing.T) {
	for i, tc := range []struct {
		yaml string
		want ConjureProjectParams
	}{
		{
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
				},
			},
		},
		{
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: input.yml
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("input.yml"),
					Publish:     true,
					AcceptFuncs: true,
				},
			},
		},
		{
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    accept-funcs: true
    ir-locator: input.json
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalFileIRProvider("input.json"),
					AcceptFuncs: true,
				},
			},
		},
		{
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: input.json
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalFileIRProvider("input.json"),
					AcceptFuncs: true,
				},
			},
		},
		{
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    export-error-decoder: true
    error-parameter-format-json: true
    ir-locator: input.json
`,
			want: ConjureProjectParams{
				{
					ProjectName:              "project-1",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalFileIRProvider("input.json"),
					AcceptFuncs:              true,
					ExportErrorDecoder:       true,
					ErrorParameterFormatJSON: true,
				},
			},
		},
	} {
		cfg, err := config.ReadConfigFromBytes([]byte(tc.yaml))
		require.NoError(t, err, "Case %d", i)

		got, _, err := ParamsFromConfig(&cfg)
		require.NoError(t, err, "Case %d", i)
		assert.Equal(t, tc.want, got, "Case %d", i)
	}
}

func TestParamsFromConfig_Warnings(t *testing.T) {
	for i, tc := range []struct {
		name         string
		yaml         string
		want         ConjureProjectParams
		wantWarnings []string
	}{
		{
			name: "No warnings for single project",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
				},
			},
			wantWarnings: nil,
		},
		{
			name: "No warnings for multiple projects with different output directories",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: outputDir-2
    omit-top-level-project-dir: true
    ir-locator: local-2/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName: "project-1",
					OutputDir:   "outputDir",
					IRProvider:  NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
				},
				{
					ProjectName: "project-2",
					OutputDir:   "outputDir-2",
					IRProvider:  NewLocalYAMLIRProvider("local-2/yaml-dir"),
					Publish:     true,
					AcceptFuncs: true,
				},
			},
			wantWarnings: nil,
		},
		{
			name: "Warning for multiple projects with the same output directory",
			yaml: `
version: 2
allow-conflicting-output-dirs: true
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName:              "project-1",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-2",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local-2/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
			},
			wantWarnings: []string{
				`project "project-1" and "project-2" have the same output directory "outputDir"`,
				`project "project-2" and "project-1" have the same output directory "outputDir"`,
			},
		},
		{
			name: "Warning for multiple projects with the same output directory after normalization",
			yaml: `
version: 2
allow-conflicting-output-dirs: true
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: ./outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName:              "project-1",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-2",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local-2/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
			},
			wantWarnings: []string{
				`project "project-1" and "project-2" have the same output directory "outputDir"`,
				`project "project-2" and "project-1" have the same output directory "outputDir"`,
			},
		},
		{
			name: "Multiple warnings for multiple projects with the same output directory after normalization",
			yaml: `
version: 2
allow-conflicting-output-dirs: true
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: ./outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
  project-3:
    output-dir: outputDir-other
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-3/yaml-dir
  project-4:
    output-dir: outputDir-other/
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-4/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName:              "project-1",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-2",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local-2/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-3",
					OutputDir:                "outputDir-other",
					IRProvider:               NewLocalYAMLIRProvider("local-3/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-4",
					OutputDir:                "outputDir-other",
					IRProvider:               NewLocalYAMLIRProvider("local-4/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
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
			yaml: `
version: 2
allow-conflicting-output-dirs: true
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: outputDir/subdir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName:              "project-1",
					OutputDir:                "outputDir",
					IRProvider:               NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-2",
					OutputDir:                "outputDir/subdir",
					IRProvider:               NewLocalYAMLIRProvider("local-2/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
			},
			wantWarnings: []string{
				`output directory "outputDir/subdir" of project "project-2" is a subdirectory of output directory "outputDir" of project "project-1"`,
			},
		},
		{
			name: "Warning for parent-child directory relationship with normalization",
			yaml: `
version: 2
allow-conflicting-output-dirs: true
projects:
  project-1:
    output-dir: base/dir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: ./base/dir/../dir/nested
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			want: ConjureProjectParams{
				{
					ProjectName:              "project-1",
					OutputDir:                "base/dir",
					IRProvider:               NewLocalYAMLIRProvider("local/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
				{
					ProjectName:              "project-2",
					OutputDir:                "base/dir/nested",
					IRProvider:               NewLocalYAMLIRProvider("local-2/yaml-dir"),
					Publish:                  true,
					AcceptFuncs:              true,
					SkipDeleteGeneratedFiles: true,
				},
			},
			wantWarnings: []string{
				`output directory "base/dir/nested" of project "project-2" is a subdirectory of output directory "base/dir" of project "project-1"`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.ReadConfigFromBytes([]byte(tc.yaml))
			require.NoError(t, err)

			got, gotWarnings, err := ParamsFromConfig(&cfg)
			require.NoError(t, err, "Case %d", i)
			assert.Equal(t, tc.want, got, "Case %d", i)

			assert.Equal(t, len(tc.wantWarnings), len(gotWarnings), "Case %d", i)
			for j := range tc.wantWarnings {
				assert.EqualError(t, gotWarnings[j], tc.wantWarnings[j], "Case %d", i)
			}
		})
	}
}

func TestParamsFromConfig_Errors(t *testing.T) {
	for i, tc := range []struct {
		name      string
		yaml      string
		wantError string
	}{
		{
			name: "Error for same output directory when conflicts not allowed",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			wantError: "output directory conflicts detected: project \"project-1\" and \"project-2\" have the same output directory \"outputDir\"\nproject \"project-2\" and \"project-1\" have the same output directory \"outputDir\"",
		},
		{
			name: "Error for parent-child directory relationship when conflicts not allowed",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: base/dir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: base/dir/nested
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			wantError: "output directory conflicts detected: output directory \"base/dir/nested\" of project \"project-2\" is a subdirectory of output directory \"base/dir\" of project \"project-1\"",
		},
		{
			name: "Error when attempting to delete with same output directory",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local-2/yaml-dir
`,
			wantError: "project \"project-1\" cannot delete generated files when output directories conflict\nproject \"project-1\" and \"project-2\" have the same output directory \"outputDir\"",
		},
		{
			name: "Error when attempting to delete with nested output directory",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: base/dir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: base/dir/nested
    omit-top-level-project-dir: true
    ir-locator: local-2/yaml-dir
`,
			wantError: "project \"project-1\" cannot delete generated files when output directories conflict\noutput directory \"base/dir/nested\" of project \"project-2\" is a subdirectory of output directory \"base/dir\" of project \"project-1\"",
		},
		{
			name: "Error when attempting to delete with one project having skip=false and conflicts exist",
			yaml: `
version: 2
projects:
  project-1:
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
  project-2:
    output-dir: outputDir
    omit-top-level-project-dir: true
    skip-delete-generated-files: true
    ir-locator: local-2/yaml-dir
`,
			wantError: "project \"project-1\" cannot delete generated files when output directories conflict\nproject \"project-1\" and \"project-2\" have the same output directory \"outputDir\"",
		},
		{
			name: "Error for invalid project name with forward slash",
			yaml: `
version: 2
projects:
  "project/invalid":
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			wantError: `project name "project/invalid" cannot contain path separators (/ or \)`,
		},
		{
			name: "Error for invalid project name with backslash",
			yaml: `
version: 2
projects:
  'project\invalid':
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			wantError: `project name "project\\invalid" cannot contain path separators (/ or \)`,
		},
		{
			name: "Error for project name that is dot",
			yaml: `
version: 2
projects:
  ".":
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			wantError: `project name cannot be "."`,
		},
		{
			name: "Error for project name that is double dot",
			yaml: `
version: 2
projects:
  "..":
    output-dir: outputDir
    omit-top-level-project-dir: true
    ir-locator: local/yaml-dir
`,
			wantError: `project name cannot be ".."`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.ReadConfigFromBytes([]byte(tc.yaml))
			require.NoError(t, err)

			got, gotWarnings, err := ParamsFromConfig(&cfg)
			require.Error(t, err, "Case %d", i)
			assert.EqualError(t, err, tc.wantError, "Case %d", i)
			assert.Empty(t, gotWarnings, "Case %d", i)
			assert.Empty(t, got, "Case %d", i)
		})
	}
}
