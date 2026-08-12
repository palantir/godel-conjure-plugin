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
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type couchDBTokenFetcher func(registry, username, password string) (string, error)

type couchDBPublisher struct {
	fetchToken couchDBTokenFetcher
}

func (couchDBPublisher) groupKey(string) (string, error) {
	return "", nil
}

func (p couchDBPublisher) prepare(req npmConfigRequest) (npmConfigFiles, error) {
	return prepareGeneratedNpmConfig(req, p.authenticate)
}

func (p couchDBPublisher) authenticate(req npmConfigRequest) (npmAuth, error) {
	token, err := p.fetchToken(req.publishRegistry, req.username, req.password)
	if err != nil {
		return npmAuth{}, err
	}
	if strings.ContainsAny(token, "\r\n") {
		return npmAuth{}, errors.New("npm authentication token must not contain a newline")
	}
	return npmAuth{token: &tokenCredential{key: "_authToken", value: token, alwaysAuth: true}}, nil
}

func fetchCouchDBToken(registry, username, password string) (string, error) {
	requestBody, err := json.Marshal(struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}{Name: username, Password: password})
	if err != nil {
		return "", errors.Wrap(err, "failed to create CouchDB npm authentication request")
	}
	authURL := registry + "/-/user/org.couchdb.user:" + url.PathEscape(username)
	req, err := http.NewRequest(http.MethodPut, authURL, bytes.NewReader(requestBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to create CouchDB npm authentication request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := npmAuthHTTPClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "CouchDB npm authentication request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", errors.Errorf("CouchDB npm authentication request failed with status %d", resp.StatusCode)
	}
	contents, err := readNpmAuthResponse(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read CouchDB npm authentication response")
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(contents, &response); err != nil {
		return "", errors.Wrap(err, "failed to parse CouchDB npm authentication response")
	}
	if response.Token == "" {
		return "", errors.New("CouchDB npm authentication response did not contain a token")
	}
	return response.Token, nil
}
