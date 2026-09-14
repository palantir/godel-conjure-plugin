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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPublishParam_CreatesImmutableTargetInputs(t *testing.T) {
	inputIR := []byte(`{"value":"original"}`)
	irProvider := &countingIRProvider{irBytes: inputIR}
	projects := ConjureProjectParams{{
		ProjectName: "project",
		IRProvider:  irProvider,
		GroupID:     "com.palantir.project",
		Publish:     true,
	}}
	extensionsProviderCalls := 0
	versionerCalls := 0

	param, err := newPublishParam(
		projects,
		PublishParamOptions{
			ProjectDir: "project-dir",
			ExtensionsProvider: func(irBytes []byte, groupID, projectName, version string) (map[string]any, error) {
				extensionsProviderCalls++
				assert.Equal(t, []byte(`{"value":"original"}`), irBytes)
				assert.Equal(t, "com.palantir.project", groupID)
				assert.Equal(t, "project", projectName)
				assert.Equal(t, "1.2.3", version)
				return nil, nil
			},
		},
		func(projectDir string) (string, error) {
			versionerCalls++
			assert.Equal(t, "project-dir", projectDir)
			return "1.2.3", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, PublishParam{
		Version: "1.2.3",
		ConjureIR: []ConjureIRPublishParam{{
			ProjectName: "project",
			IR:          `{"value":"original"}`,
			GroupID:     "com.palantir.project",
		}},
	}, param)
	assert.Equal(t, 1, versionerCalls)
	assert.Equal(t, 1, irProvider.irBytesCalls)
	assert.Equal(t, 1, extensionsProviderCalls)

	// modifying the IR field does not mutate the stored IR
	irBytes := []byte(param.ConjureIR[0].IR)
	irBytes[0] = '!'
	inputIR[0] = '!'
	assert.Equal(t, []byte(`{"value":"original"}`), []byte(param.ConjureIR[0].IR))

	assert.Equal(t, "com.palantir.project", projects[0].GroupID)
	assert.Same(t, irProvider, projects[0].IRProvider)
}

func TestNewPublishParam_ResolvesInputsOnceAcrossPublishTargets(t *testing.T) {
	bothProvider := &countingIRProvider{irBytes: []byte(`{"project":"both"}`)}
	typeScriptOnlyProvider := &countingIRProvider{irBytes: []byte(`{"project":"typescript-only"}`)}
	skippedProvider := &countingIRProvider{irBytes: []byte(`{"project":"skipped"}`)}
	bothTypeScript := &TypeScriptParam{PackageName: "@palantir/both", Publish: true}
	projects := ConjureProjectParams{
		{
			ProjectName: "both",
			IRProvider:  bothProvider,
			Publish:     true,
			TypeScript:  bothTypeScript,
		},
		{
			ProjectName: "typescript-only",
			IRProvider:  typeScriptOnlyProvider,
			TypeScript:  &TypeScriptParam{PackageName: "@palantir/typescript-only", Publish: true},
		},
		{
			ProjectName: "skipped",
			IRProvider:  skippedProvider,
		},
	}
	extensionsProviderCalls := make(map[string]int)
	versionerCalls := 0

	param, err := newPublishParam(
		projects,
		PublishParamOptions{
			ExtensionsProvider: func(_ []byte, _, projectName, _ string) (map[string]any, error) {
				extensionsProviderCalls[projectName]++
				return map[string]any{"resolved-for": projectName}, nil
			},
		},
		func(string) (string, error) {
			versionerCalls++
			return "1.2.3", nil
		},
	)
	require.NoError(t, err)
	require.Len(t, param.ConjureIR, 1)
	require.Len(t, param.TypeScript, 2)
	assert.Equal(t, "1.2.3", param.Version)
	assert.Equal(t, "both", param.ConjureIR[0].ProjectName)
	assert.Equal(t, "both", param.TypeScript[0].ProjectName)
	assert.Equal(t, "typescript-only", param.TypeScript[1].ProjectName)
	assert.Equal(t, param.ConjureIR[0].IR, param.TypeScript[0].IR)
	assert.JSONEq(t, `{"project":"both","extensions":{"resolved-for":"both"}}`, param.TypeScript[0].IR)
	assert.JSONEq(t, `{"project":"typescript-only","extensions":{"resolved-for":"typescript-only"}}`, param.TypeScript[1].IR)
	assert.Equal(t, "@palantir/both", param.TypeScript[0].Config.PackageName)
	assert.Equal(t, "@palantir/typescript-only", param.TypeScript[1].Config.PackageName)
	assert.Equal(t, "1.2.3", param.TypeScript[0].PackageVersion)
	assert.Equal(t, "1.2.3", param.TypeScript[1].PackageVersion)
	assert.Equal(t, 1, versionerCalls)
	assert.Equal(t, 1, bothProvider.irBytesCalls)
	assert.Equal(t, 1, typeScriptOnlyProvider.irBytesCalls)
	assert.Zero(t, skippedProvider.irBytesCalls)
	assert.Equal(t, map[string]int{"both": 1, "typescript-only": 1}, extensionsProviderCalls)

	bothTypeScript.PackageName = "mutated"
	assert.Equal(t, "@palantir/both", param.TypeScript[0].Config.PackageName)
}

func TestNewPublishParam_ResolvesTypeScriptPackageVersion(t *testing.T) {
	param, err := newPublishParam(
		ConjureProjectParams{{
			ProjectName: "api",
			IRProvider:  &countingIRProvider{irBytes: []byte(`{}`)},
			TypeScript: &TypeScriptParam{
				PackageName:      "@palantir/api",
				Publish:          true,
				NpmVersionScheme: NpmVersionSchemeGeneratorMajor,
			},
		}},
		PublishParamOptions{},
		func(string) (string, error) { return "0.500.0", nil },
	)
	require.NoError(t, err)
	require.Len(t, param.TypeScript, 1)
	expectedVersion, err := npmPackageVersion(NpmVersionSchemeGeneratorMajor, "0.500.0")
	require.NoError(t, err)
	assert.Equal(t, expectedVersion, param.TypeScript[0].PackageVersion)
}

func TestNewPublishParam_RejectsUnspecifiedGitVersionForTypeScript(t *testing.T) {
	param, err := newPublishParam(
		ConjureProjectParams{{
			ProjectName: "api",
			IRProvider:  &countingIRProvider{irBytes: []byte(`{}`)},
			TypeScript:  &TypeScriptParam{PackageName: "@palantir/api", Publish: true},
		}},
		PublishParamOptions{},
		func(string) (string, error) { return "unspecified", nil },
	)
	require.EqualError(t, err, "unable to determine project version from Git")
	assert.Equal(t, PublishParam{}, param)
}

func TestNewPublishParam_ExplicitPublishFalseExcludesTypeScriptButKeepsIR(t *testing.T) {
	provider := &countingIRProvider{irBytes: []byte(`{}`)}

	param, err := newPublishParam(
		ConjureProjectParams{{
			ProjectName: "api",
			IRProvider:  provider,
			Publish:     true,
			TypeScript:  &TypeScriptParam{PackageName: "@palantir/api", Publish: false},
		}},
		PublishParamOptions{},
		// An unspecified Git version would be rejected if this project's TypeScript were still enabled.
		func(string) (string, error) { return "unspecified", nil },
	)
	require.NoError(t, err)
	require.Len(t, param.ConjureIR, 1)
	assert.Equal(t, "api", param.ConjureIR[0].ProjectName)
	assert.Empty(t, param.TypeScript)
}

func TestNewPublishParam_RejectsDuplicateNpmPackageNameAndVersion(t *testing.T) {
	firstProvider := &countingIRProvider{irBytes: []byte(`{}`)}
	secondProvider := &countingIRProvider{irBytes: []byte(`{}`)}

	param, err := newPublishParam(
		ConjureProjectParams{
			{
				ProjectName: "first",
				IRProvider:  firstProvider,
				TypeScript:  &TypeScriptParam{PackageName: "@palantir/api", Publish: true},
			},
			{
				ProjectName: "second",
				IRProvider:  secondProvider,
				TypeScript:  &TypeScriptParam{PackageName: "@palantir/api", Publish: true},
			},
		},
		PublishParamOptions{},
		func(string) (string, error) { return "1.2.3", nil },
	)
	require.EqualError(t, err, `projects "first" and "second" both resolve to npm package @palantir/api@1.2.3`)
	assert.Equal(t, PublishParam{}, param)
	assert.Zero(t, firstProvider.irBytesCalls, "duplicate check must run before any per-project IR resolution")
	assert.Zero(t, secondProvider.irBytesCalls, "duplicate check must run before any per-project IR resolution")
}

func TestNewPublishParam_FiltersProjectsPreservesOrderAndMergesExtensions(t *testing.T) {
	firstProvider := &countingIRProvider{
		irBytes: []byte(`{"extensions":{"keep":"existing","replace":"old"},"other":"value"}`),
	}
	skippedProvider := &countingIRProvider{irBytes: []byte(`{"skipped":true}`)}
	thirdProvider := &countingIRProvider{irBytes: []byte(`{}`)}
	projects := ConjureProjectParams{
		{
			ProjectName: "first",
			IRProvider:  firstProvider,
			GroupID:     "com.palantir.first",
			Publish:     true,
		},
		{
			ProjectName: "skipped",
			IRProvider:  skippedProvider,
		},
		{
			ProjectName: "third",
			IRProvider:  thirdProvider,
			GroupID:     "com.palantir.third",
			Publish:     true,
		},
	}
	extensionsProviderCalls := make(map[string]int)

	param, err := newPublishParam(
		projects,
		PublishParamOptions{
			ExtensionsProvider: func(_ []byte, _, projectName, _ string) (map[string]any, error) {
				extensionsProviderCalls[projectName]++
				if projectName == "first" {
					return map[string]any{
						"added":   "extension",
						"replace": "new",
					}, nil
				}
				return nil, nil
			},
		},
		func(string) (string, error) { return "2.0.0", nil },
	)
	require.NoError(t, err)
	require.Len(t, param.ConjureIR, 2)
	assert.Equal(t, []string{"first", "third"}, []string{param.ConjureIR[0].ProjectName, param.ConjureIR[1].ProjectName})
	assert.Equal(t, []string{"com.palantir.first", "com.palantir.third"}, []string{param.ConjureIR[0].GroupID, param.ConjureIR[1].GroupID})
	assert.Equal(t, 1, firstProvider.irBytesCalls)
	assert.Zero(t, skippedProvider.irBytesCalls)
	assert.Equal(t, 1, thirdProvider.irBytesCalls)
	assert.Equal(t, map[string]int{"first": 1, "third": 1}, extensionsProviderCalls)

	assert.JSONEq(t, `{
		"extensions": {
			"added": "extension",
			"keep": "existing",
			"replace": "new"
		},
		"other": "value"
	}`, param.ConjureIR[0].IR)
	assert.Equal(t, `{}`, param.ConjureIR[1].IR)
}

func TestNewPublishParam_NilExtensionsProviderDoesNotModifyIR(t *testing.T) {
	irProvider := &countingIRProvider{irBytes: []byte(`{"value":"original"}`)}

	param, err := newPublishParam(
		ConjureProjectParams{{
			ProjectName: "project",
			IRProvider:  irProvider,
			Publish:     true,
		}},
		PublishParamOptions{},
		func(string) (string, error) { return "1.2.3", nil },
	)
	require.NoError(t, err)
	require.Len(t, param.ConjureIR, 1)
	assert.Equal(t, `{"value":"original"}`, param.ConjureIR[0].IR)
	assert.Equal(t, 1, irProvider.irBytesCalls)
}

func TestNewPublishParam_GroupIDPrecedence(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		projectGroupID  string
		groupIDOverride string
		expected        string
	}{
		{
			name:     "empty",
			expected: "",
		},
		{
			name:           "project group ID",
			projectGroupID: "com.palantir.project",
			expected:       "com.palantir.project",
		},
		{
			name:            "override group ID",
			groupIDOverride: "com.palantir.override",
			expected:        "com.palantir.override",
		},
		{
			name:            "override wins",
			projectGroupID:  "com.palantir.project",
			groupIDOverride: "com.palantir.override",
			expected:        "com.palantir.override",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			irProvider := &countingIRProvider{irBytes: []byte(`{}`)}
			var providedGroupID string

			param, err := newPublishParam(
				ConjureProjectParams{{
					ProjectName: "project",
					GroupID:     testCase.projectGroupID,
					IRProvider:  irProvider,
					Publish:     true,
				}},
				PublishParamOptions{
					GroupIDOverride: testCase.groupIDOverride,
					ExtensionsProvider: func(_ []byte, groupID, _, _ string) (map[string]any, error) {
						providedGroupID = groupID
						return nil, nil
					},
				},
				func(string) (string, error) {
					return "1.2.3", nil
				},
			)
			require.NoError(t, err)
			require.Len(t, param.ConjureIR, 1)
			assert.Equal(t, testCase.expected, providedGroupID)
			assert.Equal(t, testCase.expected, param.ConjureIR[0].GroupID)
			assert.Equal(t, 1, irProvider.irBytesCalls)
		})
	}
}

func TestNewPublishParam_EmptySetAvoidsVersionAndIRResolution(t *testing.T) {
	notPublishedProvider := &countingIRProvider{irBytes: []byte(`{}`)}
	extensionsProviderCalls := 0
	versionerCalls := 0

	param, err := newPublishParam(
		ConjureProjectParams{{
			ProjectName: "not-published",
			IRProvider:  notPublishedProvider,
		}},
		PublishParamOptions{
			ExtensionsProvider: func(_ []byte, _, _, _ string) (map[string]any, error) {
				extensionsProviderCalls++
				return nil, nil
			},
		},
		func(string) (string, error) {
			versionerCalls++
			return "", errors.New("versioner should not be called")
		},
	)
	require.NoError(t, err)
	assert.Equal(t, PublishParam{}, param)
	assert.Zero(t, versionerCalls)
	assert.Zero(t, notPublishedProvider.irBytesCalls)
	assert.Zero(t, extensionsProviderCalls)
}

func TestNewPublishParam_ExtensionsProviderErrorReturnsNoPartialInputs(t *testing.T) {
	firstProvider := &countingIRProvider{irBytes: []byte(`{}`)}
	secondProvider := &countingIRProvider{irBytes: []byte(`{}`)}
	extensionError := errors.New("extension provider failed")
	extensionsProviderCalls := make(map[string]int)
	versionerCalls := 0

	param, err := newPublishParam(
		ConjureProjectParams{
			{
				ProjectName: "first",
				IRProvider:  firstProvider,
				Publish:     true,
			},
			{
				ProjectName: "second",
				IRProvider:  secondProvider,
				Publish:     true,
			},
		},
		PublishParamOptions{
			ExtensionsProvider: func(_ []byte, _, projectName, _ string) (map[string]any, error) {
				extensionsProviderCalls[projectName]++
				if projectName == "second" {
					return nil, extensionError
				}
				return nil, nil
			},
		},
		func(string) (string, error) {
			versionerCalls++
			return "1.2.3", nil
		},
	)
	require.ErrorIs(t, err, extensionError)
	assert.Equal(t, PublishParam{}, param)
	assert.Equal(t, 1, versionerCalls)
	assert.Equal(t, 1, firstProvider.irBytesCalls)
	assert.Equal(t, 1, secondProvider.irBytesCalls)
	assert.Equal(t, map[string]int{"first": 1, "second": 1}, extensionsProviderCalls)
}

func TestNewPublishParam_IRProviderErrorReturnsNoPartialInputs(t *testing.T) {
	firstProvider := &countingIRProvider{irBytes: []byte(`{}`)}
	irProviderError := errors.New("IR provider failed")
	secondProvider := &countingIRProvider{err: irProviderError}

	param, err := newPublishParam(
		ConjureProjectParams{
			{ProjectName: "first", IRProvider: firstProvider, Publish: true},
			{ProjectName: "second", IRProvider: secondProvider, Publish: true},
		},
		PublishParamOptions{},
		func(string) (string, error) { return "1.2.3", nil },
	)
	require.ErrorIs(t, err, irProviderError)
	assert.Equal(t, PublishParam{}, param)
	assert.Equal(t, 1, firstProvider.irBytesCalls)
	assert.Equal(t, 1, secondProvider.irBytesCalls)
}

type countingIRProvider struct {
	irBytes      []byte
	err          error
	irBytesCalls int
}

func (p *countingIRProvider) IRBytes() ([]byte, error) {
	p.irBytesCalls++
	return p.irBytes, p.err
}

func (p *countingIRProvider) GeneratedFromYAML() bool {
	return false
}
