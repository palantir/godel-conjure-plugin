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
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNpmrcDirectToken(t *testing.T) {
	got, err := generateNpmrc(npmrcConfig{
		PackageNames:    []string{"@zeta/api", "@palantir/api", "@palantir/other"},
		PublishRegistry: "https://publish.example.com/api/npm/internal-release/",
		InstallRegistry: "https://read.example.com/api/npm/all-npm///",
	}, npmAuth{token: &tokenCredential{key: "_authToken", value: "secret-token", alwaysAuth: true}})
	require.NoError(t, err)

	assert.Equal(t, "https://publish.example.com/api/npm/internal-release", got.PublishRegistry)
	assert.Equal(t, "https://read.example.com/api/npm/all-npm", got.InstallRegistry)
	assert.Equal(t, "registry=https://read.example.com/api/npm/all-npm/\n"+
		"@palantir:registry=https://read.example.com/api/npm/all-npm/\n"+
		"@zeta:registry=https://read.example.com/api/npm/all-npm/\n", got.InstallContents)
	assert.Equal(t, "registry=https://publish.example.com/api/npm/internal-release/\n"+
		"@palantir:registry=https://publish.example.com/api/npm/internal-release/\n"+
		"@zeta:registry=https://publish.example.com/api/npm/internal-release/\n"+
		"//publish.example.com/api/npm/internal-release/:_authToken=secret-token\n"+
		"always-auth=true\n", got.PublishContents)
}

func TestGenerateNpmrcTreatsExplicitlyIdenticalRegistriesAsShared(t *testing.T) {
	got, err := generateNpmrc(npmrcConfig{
		PackageNames:    []string{"@palantir/api"},
		PublishRegistry: "https://registry.example.com/api/npm/internal-release/",
		InstallRegistry: "https://registry.example.com/api/npm/internal-release",
	}, npmAuth{token: &tokenCredential{key: "_authToken", value: "secret-token", alwaysAuth: true}})
	require.NoError(t, err)
	assert.Contains(t, got.InstallContents, "_authToken=secret-token")
	assert.Equal(t, got.InstallContents, got.PublishContents)
}

func TestPrepareGeneratedNpmConfigDirectTokenBypassesAuthentication(t *testing.T) {
	files, err := prepareGeneratedNpmConfig(npmConfigRequest{
		packageNames:    []string{"@palantir/api"},
		publishRegistry: "https://registry.example.com",
		token:           "secret-token",
	}, func(npmConfigRequest) (npmAuth, error) {
		t.Fatal("authentication must not run for a direct token")
		return npmAuth{}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, files.close()) })
	contents, err := os.ReadFile(files.publishPath)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "_authToken=secret-token")
}

func TestArtifactoryPublisherAuthentication(t *testing.T) {
	const authResponse = "//registry.example.com/api/npm/internal-release/:username=build-user\n" +
		"//registry.example.com/api/npm/internal-release/:_password=YnVpbGQtcGFzc3dvcmQ=\n" +
		"//registry.example.com/api/npm/internal-release/:email=build-user@example.com\n" +
		"//registry.example.com/api/npm/internal-release/:always-auth=true\n"
	publisher := artifactoryPublisher{fetchAuth: func(authURL, username, password string) (string, error) {
		assert.Equal(t, "https://registry.example.com/api/npm/internal-release/auth/palantir", authURL)
		assert.Equal(t, "build-user", username)
		assert.Equal(t, "build-password", password)
		return authResponse, nil
	}}
	req := npmConfigRequest{
		packageNames:    []string{"@palantir/api", "@palantir/other"},
		publishRegistry: "https://registry.example.com/api/npm/internal-release",
		username:        "build-user",
		password:        "build-password",
	}
	auth, err := publisher.authenticate(req)
	require.NoError(t, err)
	got, err := generateNpmrc(npmrcConfig{
		PackageNames:    req.packageNames,
		PublishRegistry: req.publishRegistry,
	}, auth)
	require.NoError(t, err)
	assert.Contains(t, got.PublishContents, authResponse)
	assert.Equal(t, got.InstallContents, got.PublishContents)
}

func TestArtifactoryPublisherRequiresOneScope(t *testing.T) {
	publisher := artifactoryPublisher{fetchAuth: func(string, string, string) (string, error) {
		t.Fatal("credential exchange must not run for multiple scopes")
		return "", nil
	}}
	_, err := publisher.authenticate(npmConfigRequest{
		packageNames:    []string{"@alpha/api", "@beta/api"},
		publishRegistry: "https://registry.example.com/api/npm/internal-release",
		username:        "build-user",
		password:        "build-password",
	})
	require.ErrorContains(t, err, "exactly one package scope")
}

func TestCouchDBPublisherAuthentication(t *testing.T) {
	publisher := couchDBPublisher{fetchToken: func(registry, username, password string) (string, error) {
		assert.Equal(t, "https://registry.example.com/custom", registry)
		assert.Equal(t, "build-user", username)
		assert.Equal(t, "build-password", password)
		return "registry-token", nil
	}}
	req := npmConfigRequest{
		packageNames:    []string{"@alpha/api", "@beta/api"},
		publishRegistry: "https://registry.example.com/custom",
		username:        "build-user",
		password:        "build-password",
	}
	auth, err := publisher.authenticate(req)
	require.NoError(t, err)
	got, err := generateNpmrc(npmrcConfig{
		PackageNames:    req.packageNames,
		PublishRegistry: req.publishRegistry,
	}, auth)
	require.NoError(t, err)
	assert.Contains(t, got.PublishContents, "//registry.example.com/custom/:_authToken=registry-token\n")
	assert.Contains(t, got.PublishContents, "@alpha:registry=https://registry.example.com/custom/\n")
	assert.Contains(t, got.PublishContents, "@beta:registry=https://registry.example.com/custom/\n")
}

func TestGenerateNpmrcLeavesDistinctInstallRegistryUnauthenticated(t *testing.T) {
	got, err := generateNpmrc(npmrcConfig{
		PackageNames:    []string{"@palantir/api"},
		PublishRegistry: "https://publish.example.com/api/npm/internal-release",
		InstallRegistry: "https://publish.example.com/api/npm/all-npm",
	}, npmAuth{fragment: "_auth = base64-credentials"})
	require.NoError(t, err)
	assert.Equal(t, "registry=https://publish.example.com/api/npm/all-npm/\n"+
		"@palantir:registry=https://publish.example.com/api/npm/all-npm/\n", got.InstallContents)
	assert.NotContains(t, got.InstallContents, "_auth")
	assert.Contains(t, got.PublishContents, "_auth = base64-credentials")
}

func TestFetchArtifactoryNpmAuth(t *testing.T) {
	originalClient := npmAuthHTTPClient
	npmAuthHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://registry.example.com/api/npm/internal-release/auth/palantir", req.URL.String())
		username, password, ok := req.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "build-user", username)
		assert.Equal(t, "build-password", password)
		assert.Equal(t, "text/plain", req.Header.Get("Accept"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("  _auth=opaque\n\n")),
		}, nil
	})}
	t.Cleanup(func() { npmAuthHTTPClient = originalClient })

	fragment, err := fetchArtifactoryNpmAuth(
		"https://registry.example.com/api/npm/internal-release/auth/palantir",
		"build-user",
		"build-password",
	)
	require.NoError(t, err)
	assert.Equal(t, "  _auth=opaque\n\n", fragment)
}

func TestFetchCouchDBToken(t *testing.T) {
	originalClient := npmAuthHTTPClient
	npmAuthHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPut, req.Method)
		assert.Equal(t, "https://registry.example.com/custom/-/user/org.couchdb.user:build-user", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Accept"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		requestBody, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var body map[string]string
		require.NoError(t, json.Unmarshal(requestBody, &body))
		assert.Equal(t, map[string]string{"name": "build-user", "password": "build-password"}, body)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"token":"registry-token","ignored":true}`)),
		}, nil
	})}
	t.Cleanup(func() { npmAuthHTTPClient = originalClient })

	token, err := fetchCouchDBToken("https://registry.example.com/custom", "build-user", "build-password")
	require.NoError(t, err)
	assert.Equal(t, "registry-token", token)
}

func TestFetchArtifactoryNpmAuthRejectsEmptyResponse(t *testing.T) {
	originalClient := npmAuthHTTPClient
	npmAuthHTTPClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("  \n")),
		}, nil
	})}
	t.Cleanup(func() { npmAuthHTTPClient = originalClient })

	_, err := fetchArtifactoryNpmAuth("https://registry.example.com/auth/palantir", "user", "password")
	require.ErrorContains(t, err, "empty")
}

func TestNpmConfigurationValidation(t *testing.T) {
	t.Run("publish registry required", func(t *testing.T) {
		_, err := generateNpmrc(npmrcConfig{}, npmAuth{})
		require.ErrorContains(t, err, "registry URL must be specified")
	})
	t.Run("registry must be absolute HTTP URL", func(t *testing.T) {
		_, err := generateNpmrc(npmrcConfig{PublishRegistry: "registry.example.com"}, npmAuth{})
		require.ErrorContains(t, err, "absolute HTTP(S)")
	})
	t.Run("scope must be valid", func(t *testing.T) {
		_, err := generateNpmrc(npmrcConfig{
			PublishRegistry: "https://registry.example.com",
			PackageNames:    []string{"@invalid:scope/api"},
		}, npmAuth{})
		require.ErrorContains(t, err, "invalid scope")
	})
	t.Run("provider must be supported", func(t *testing.T) {
		_, err := npmPublisherFor("unknown")
		require.ErrorContains(t, err, "unsupported npm publisher provider")
	})
	for _, testCase := range []struct {
		name string
		req  npmConfigRequest
		want string
	}{
		{
			name: "token and username password are exclusive",
			req:  npmConfigRequest{username: "user", password: "password", token: "token"},
			want: "not both",
		},
		{
			name: "username and password are paired",
			req:  npmConfigRequest{username: "user"},
			want: "specified together",
		},
		{
			name: "token cannot inject npm configuration",
			req:  npmConfigRequest{token: "token\nregistry=https://evil.example.com"},
			want: "must not contain a newline",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.ErrorContains(t, validateNpmCredentials(testCase.req), testCase.want)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
