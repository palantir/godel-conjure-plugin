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
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultArtifactoryScope = "palantir"
)

type npmAuthRequest struct {
	packageNames []string
	registry     string
	username     string
	password     string
}

func authenticateArtifactory(authRequest npmAuthRequest) (npmAuth, error) {
	scopes, err := artifactoryScopes(authRequest.packageNames)
	if err != nil {
		return npmAuth{}, err
	}
	if len(scopes) == 0 {
		return npmAuth{}, errors.New("cannot authenticate with Artifactory without an npm package")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Do not forward basic auth cred to a redirected host
			return http.ErrUseLastResponse
		},
	}
	npmrcContents := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		authURL := authRequest.registry + "/auth/" + url.PathEscape(scope)
		req, err := http.NewRequest(http.MethodGet, authURL, nil)
		if err != nil {
			return npmAuth{}, errors.Wrap(err, "failed to create Artifactory npm authentication request")
		}
		req.SetBasicAuth(authRequest.username, authRequest.password)
		req.Header.Set("Accept", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			return npmAuth{}, errors.Wrap(err, "Artifactory npm authentication request failed")
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		if resp.StatusCode != http.StatusOK {
			return npmAuth{}, errors.Errorf("Artifactory npm authentication request failed with status %d", resp.StatusCode)
		}
		contents, err := io.ReadAll(resp.Body)
		if err != nil {
			return npmAuth{}, errors.Wrap(err, "failed to read Artifactory npm authentication response")
		}
		npmrcContent := strings.TrimSpace(string(contents))
		if npmrcContent == "" {
			return npmAuth{}, errors.New("Artifactory npm authentication response was empty")
		}
		npmrcContents = append(npmrcContents, npmrcContent)
	}
	return npmAuth{
		npmrcContent: strings.Join(npmrcContents, "\n"),
	}, nil
}

func artifactoryScopes(packageNames []string) ([]string, error) {
	scopes, err := packageScopes(packageNames)
	if err != nil {
		return nil, err
	}
	for _, packageName := range packageNames {
		if !strings.HasPrefix(packageName, "@") {
			scopes = append(scopes, defaultArtifactoryScope)
			break
		}
	}
	slices.Sort(scopes)
	return slices.Compact(scopes), nil
}
