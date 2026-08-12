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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const maxNpmAuthResponseBytes = 1024 * 1024

var npmAuthHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

const (
	// NpmPublisherProviderArtifactory uses Artifactory's scoped npm authentication endpoint.
	NpmPublisherProviderArtifactory = "artifactory"
	// NpmPublisherProviderCouchDB uses the standard npm CouchDB token exchange.
	NpmPublisherProviderCouchDB = "couchdb"
	// NpmPublisherProviderNpmrc uses an npmrc file supplied by the caller.
	NpmPublisherProviderNpmrc = "npmrc"
	// DefaultNpmPublisherProvider is used when a TypeScript project does not select a provider.
	DefaultNpmPublisherProvider = NpmPublisherProviderArtifactory
)

type npmPublisher interface {
	groupKey(packageName string) (string, error)
	prepare(npmConfigRequest) (npmConfigFiles, error)
}

type npmPublisherDefinition struct {
	publisher     npmPublisher
	usesNpmrcFile bool
}

type npmConfigRequest struct {
	packageNames    []string
	publishRegistry string
	installRegistry string
	username        string
	password        string
	token           string
	npmrcFile       string
}

type npmConfigFiles struct {
	installPath string
	publishPath string
	close       func() error
}

func prepareGeneratedNpmConfig(req npmConfigRequest, authenticate func(npmConfigRequest) (npmAuth, error)) (npmConfigFiles, error) {
	if err := validateNpmCredentials(req); err != nil {
		return npmConfigFiles{}, err
	}

	var auth npmAuth
	switch {
	case req.token != "":
		auth.token = &tokenCredential{key: "_authToken", value: req.token, alwaysAuth: true}
	case req.username != "":
		var err error
		auth, err = authenticate(req)
		if err != nil {
			return npmConfigFiles{}, err
		}
	}

	npmrc, err := generateNpmrc(npmrcConfig{
		PackageNames:    req.packageNames,
		PublishRegistry: req.publishRegistry,
		InstallRegistry: req.installRegistry,
	}, auth)
	if err != nil {
		return npmConfigFiles{}, err
	}
	return writeTemporaryNpmrcFiles(npmrc)
}

var npmPublisherFactories = map[string]func() npmPublisherDefinition{
	NpmPublisherProviderArtifactory: func() npmPublisherDefinition {
		return npmPublisherDefinition{publisher: artifactoryPublisher{fetchAuth: fetchArtifactoryNpmAuth}}
	},
	NpmPublisherProviderCouchDB: func() npmPublisherDefinition {
		return npmPublisherDefinition{publisher: couchDBPublisher{fetchToken: fetchCouchDBToken}}
	},
	NpmPublisherProviderNpmrc: func() npmPublisherDefinition {
		return npmPublisherDefinition{publisher: npmrcPublisher{}, usesNpmrcFile: true}
	},
}

// IsNpmPublisherProvider reports whether name identifies a supported npm publisher provider.
func IsNpmPublisherProvider(name string) bool {
	_, ok := npmPublisherFactories[name]
	return ok
}

func npmPublisherFor(name string) (npmPublisherDefinition, error) {
	factory, ok := npmPublisherFactories[name]
	if !ok {
		return npmPublisherDefinition{}, errors.Errorf("unsupported npm publisher provider %q", name)
	}
	return factory(), nil
}

func validateNpmCredentials(req npmConfigRequest) error {
	if req.token != "" && (req.username != "" || req.password != "") {
		return errors.New("either npm username and password or npm token may be specified, but not both")
	}
	if (req.username == "") != (req.password == "") {
		return errors.New("npm username and password must be specified together")
	}
	if strings.ContainsAny(req.token, "\r\n") {
		return errors.New("npm token must not contain a newline")
	}
	return nil
}

func resolveNpmrcFile(npmrcFile string) (string, error) {
	absPath, err := filepath.Abs(npmrcFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve npmrc file")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", errors.Wrap(err, "npmrc file does not exist")
	}
	if !info.Mode().IsRegular() {
		return "", errors.Errorf("npmrc file is not a regular file: %s", absPath)
	}
	if info.Size() == 0 {
		return "", errors.Errorf("npmrc file must not be empty: %s", absPath)
	}
	return absPath, nil
}

func writeTemporaryNpmrcFiles(config npmrc) (npmConfigFiles, error) {
	tmpDir, err := os.MkdirTemp("", "conjure-typescript-npm-config-")
	if err != nil {
		return npmConfigFiles{}, errors.Wrap(err, "failed to create temporary npm configuration directory")
	}
	cleanup := func() error {
		if err := os.RemoveAll(tmpDir); err != nil {
			return errors.Wrap(err, "failed to remove temporary npm configuration directory")
		}
		return nil
	}
	installPath := filepath.Join(tmpDir, "install.npmrc")
	publishPath := filepath.Join(tmpDir, "publish.npmrc")
	if err := os.WriteFile(installPath, []byte(config.InstallContents), 0600); err != nil {
		_ = cleanup()
		return npmConfigFiles{}, errors.Wrap(err, "failed to write temporary npm install configuration")
	}
	if err := os.WriteFile(publishPath, []byte(config.PublishContents), 0600); err != nil {
		_ = cleanup()
		return npmConfigFiles{}, errors.Wrap(err, "failed to write temporary npm publish configuration")
	}
	return npmConfigFiles{
		installPath: installPath,
		publishPath: publishPath,
		close:       cleanup,
	}, nil
}

func readNpmAuthResponse(body io.Reader) ([]byte, error) {
	contents, err := io.ReadAll(io.LimitReader(body, maxNpmAuthResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(contents) > maxNpmAuthResponseBytes {
		return nil, errors.New("npm authentication response is too large")
	}
	return contents, nil
}
