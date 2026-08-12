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
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type artifactoryAuthFetcher func(authURL, username, password string) (string, error)

type artifactoryPublisher struct {
	fetchAuth artifactoryAuthFetcher
}

func (artifactoryPublisher) groupKey(packageName string) (string, error) {
	if !strings.HasPrefix(packageName, "@") {
		return "", errors.Errorf("npm package %q must include a scope for Artifactory publishing", packageName)
	}
	scope, _, ok := strings.Cut(packageName[1:], "/")
	if !ok || scope == "" {
		return "", errors.Errorf("npm package %q must include a scope for Artifactory publishing", packageName)
	}
	return scope, nil
}

func (p artifactoryPublisher) prepare(req npmConfigRequest) (npmConfigFiles, error) {
	return prepareGeneratedNpmConfig(req, p.authenticate)
}

func (p artifactoryPublisher) authenticate(req npmConfigRequest) (npmAuth, error) {
	scopes, err := packageScopes(req.packageNames)
	if err != nil {
		return npmAuth{}, err
	}
	if len(scopes) != 1 {
		return npmAuth{}, errors.Errorf(
			"artifactory npm publisher requires exactly one package scope per authentication request, found %d",
			len(scopes))
	}
	authURL := req.publishRegistry + "/auth/" + scopes[0]
	fragment, err := p.fetchAuth(authURL, req.username, req.password)
	if err != nil {
		return npmAuth{}, errors.Wrapf(err, "failed to authenticate npm scope %q", scopes[0])
	}
	return npmAuth{fragment: fragment}, nil
}

func fetchArtifactoryNpmAuth(authURL, username, password string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, authURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create Artifactory npm authentication request")
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "text/plain")

	resp, err := npmAuthHTTPClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "Artifactory npm authentication request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", errors.Errorf("Artifactory npm authentication request failed with status %d", resp.StatusCode)
	}

	contents, err := readNpmAuthResponse(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read Artifactory npm authentication response")
	}
	if strings.TrimSpace(string(contents)) == "" {
		return "", errors.New("Artifactory npm authentication response was empty")
	}
	return string(contents), nil
}
