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

package conjureplugin

import (
	"io/ioutil"
	"net/http"

	"github.com/palantir/godel-conjure-plugin/v4/ir-gen-cli-bundler/conjureircli"
	"github.com/palantir/pkg/safehttp"
	"github.com/pkg/errors"
)

type IRProvider interface {
	IRBytes() ([]byte, error)
	// Generated returns true if the IR provided by this provider is generated from YAML, false otherwise.
	GeneratedFromYAML() bool
}

var _ IRProvider = &localYAMLIRProvider{}

type localYAMLIRProvider struct {
	path string
}

// NewLocalYAMLIRProvider returns an IRProvider that provides IR generated from local YAML. The provided path must be a
// path to a Conjure YAML file or a directory that contains Conjure YAML files.
func NewLocalYAMLIRProvider(path string) IRProvider {
	return &localYAMLIRProvider{
		path: path,
	}
}

func (p *localYAMLIRProvider) IRBytes() ([]byte, error) {
	return conjureircli.InputPathToIR(p.path)
}

func (p *localYAMLIRProvider) GeneratedFromYAML() bool {
	return true
}

var _ IRProvider = &urlIRProvider{}

type urlIRProvider struct {
	irURL string
}

// NewHTTPIRProvider returns an IRProvider that that provides IR downloaded from the provided URL over HTTP.
func NewHTTPIRProvider(irURL string) IRProvider {
	return &urlIRProvider{
		irURL: irURL,
	}
}

func (p *urlIRProvider) IRBytes() ([]byte, error) {
	resp, cleanup, err := safehttp.Get(http.DefaultClient, p.irURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer cleanup()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("expected response status 200 when fetching IR from remote source %s, but got %d", p.irURL, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func (p *urlIRProvider) GeneratedFromYAML() bool {
	return false
}

var _ IRProvider = &localFileIRProvider{}

type localFileIRProvider struct {
	path string
}

// NewLocalFileIRProvider returns an IRProvider that that provides IR from the local file at the specified path.
func NewLocalFileIRProvider(path string) IRProvider {
	return &localFileIRProvider{
		path: path,
	}
}

func (p *localFileIRProvider) IRBytes() ([]byte, error) {
	return ioutil.ReadFile(p.path)
}

func (p *localFileIRProvider) GeneratedFromYAML() bool {
	return false
}
