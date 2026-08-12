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

package typescript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareNpmPublishPlanGroupsProjectsByProviderPolicy(t *testing.T) {
	npmrcFile := filepath.Join(t.TempDir(), "external.npmrc")
	require.NoError(t, os.WriteFile(npmrcFile, []byte("registry=https://registry.example.com/\n"), 0600))
	plan, err := PrepareNpmPublishPlan([]NpmPublishProject{
		{ProjectIndex: 0, ProjectName: "alpha-1", PackageName: "@alpha/one"},
		{ProjectIndex: 1, ProjectName: "alpha-2", PackageName: "@alpha/two"},
		{ProjectIndex: 2, ProjectName: "beta", PackageName: "@beta/one"},
		{ProjectIndex: 3, ProjectName: "couch", PackageName: "@gamma/one", PublisherProvider: NpmPublisherProviderCouchDB},
		{ProjectIndex: 4, ProjectName: "couch-2", PackageName: "@omega/one", PublisherProvider: NpmPublisherProviderCouchDB},
		{ProjectIndex: 5, ProjectName: "external", PackageName: "@delta/one", PublisherProvider: NpmPublisherProviderNpmrc},
	}, NpmPublishPlanConfig{
		PublishRegistry: "https://registry.example.com/",
		Token:           "secret-token",
		NpmrcFile:       npmrcFile,
	})
	require.NoError(t, err)
	require.Len(t, plan.Groups, 4)
	assert.Equal(t, []int{0, 1}, plan.Groups[0].ProjectIndexes)
	assert.Equal(t, []int{2}, plan.Groups[1].ProjectIndexes)
	assert.Equal(t, []int{3, 4}, plan.Groups[2].ProjectIndexes)
	assert.Equal(t, []int{5}, plan.Groups[3].ProjectIndexes)
	assert.Equal(t, npmrcFile, plan.Groups[3].InstallNpmrcPath)
	assert.Equal(t, npmrcFile, plan.Groups[3].PublishNpmrcPath)

	generatedPath := plan.Groups[0].PublishNpmrcPath
	require.FileExists(t, generatedPath)
	require.NoError(t, plan.Close())
	_, err = os.Stat(generatedPath)
	assert.True(t, os.IsNotExist(err), "generated npmrc should be removed")
	assert.FileExists(t, npmrcFile, "caller-owned npmrc should not be removed")
}

func TestPrepareNpmPublishPlanArtifactoryScopeValidation(t *testing.T) {
	_, err := PrepareNpmPublishPlan([]NpmPublishProject{
		{ProjectIndex: 0, ProjectName: "api", PackageName: "unscoped"},
	}, NpmPublishPlanConfig{PublishRegistry: "https://registry.example.com"})
	require.ErrorContains(t, err, "must include a scope for Artifactory publishing")
}

func TestPrepareNpmPublishPlanRejectsUnknownProvider(t *testing.T) {
	_, err := PrepareNpmPublishPlan([]NpmPublishProject{
		{ProjectIndex: 0, ProjectName: "api", PackageName: "@example/api", PublisherProvider: "unknown"},
	}, NpmPublishPlanConfig{PublishRegistry: "https://registry.example.com"})
	require.ErrorContains(t, err, `TypeScript project "api" has unsupported npm publisher provider "unknown"`)
}
