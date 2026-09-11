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
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// NpmConfigOptions contains publish-time npm registry and auth inputs.
type NpmConfigOptions struct {
	PackageNames    []string
	PublishRegistry string
	InstallRegistry string
	Username        string
	Password        string
	Token           string
}

// NpmConfig contains separate npmrc file contents for dependency installation and publication.
type NpmConfig struct {
	PublishRegistry string
	InstallNpmrc    string
	PublishNpmrc    string
}

type npmAuth struct {
	token        string
	npmrcContent string
}

// ValidateNpmConfig verifies the format of the registry URL, credentials, and npm package name/scope.
func ValidateNpmConfig(opts NpmConfigOptions) error {
	if _, _, err := normalizeRegistryURL(opts.PublishRegistry); err != nil {
		return errors.Wrap(err, "invalid npm publish registry")
	}
	if opts.InstallRegistry != "" {
		if _, _, err := normalizeRegistryURL(opts.InstallRegistry); err != nil {
			return errors.Wrap(err, "invalid npm install registry")
		}
	}
	if err := validateCredentials(opts); err != nil {
		return err
	}
	_, err := packageScopes(opts.PackageNames)
	return err
}

// ResolveNpmConfig prepares npm configuration for one publishing operation.
// A token is used directly or username/password credentials are exchanged with Artifactory.
func ResolveNpmConfig(opts NpmConfigOptions) (NpmConfig, error) {
	if err := ValidateNpmConfig(opts); err != nil {
		return NpmConfig{}, err
	}

	publishRegistry, publishURL, err := normalizeRegistryURL(opts.PublishRegistry)
	if err != nil {
		return NpmConfig{}, errors.Wrap(err, "invalid npm publish registry")
	}

	auth := npmAuth{
		token: opts.Token,
	}
	if auth.token == "" && opts.Username != "" {
		authReq := npmAuthRequest{
			packageNames: opts.PackageNames,
			registry:     publishRegistry,
			username:     opts.Username,
			password:     opts.Password,
		}
		auth, err = authenticateArtifactory(authReq)
		if err != nil {
			return NpmConfig{}, errors.Wrap(err, "failed to authenticate with Artifactory npm registry")
		}
	}

	installRegistry := publishRegistry
	installURL := publishURL
	if opts.InstallRegistry != "" {
		installRegistry, installURL, err = normalizeRegistryURL(opts.InstallRegistry)
		if err != nil {
			return NpmConfig{}, errors.Wrap(err, "invalid npm install registry")
		}
	}

	publishNpmrc, err := renderNpmrc(publishRegistry, publishURL, opts.PackageNames, auth)
	if err != nil {
		return NpmConfig{}, err
	}
	installAuth := auth
	if installRegistry != publishRegistry {
		installAuth = npmAuth{}
	}
	installNpmrc, err := renderNpmrc(installRegistry, installURL, opts.PackageNames, installAuth)
	if err != nil {
		return NpmConfig{}, err
	}
	return NpmConfig{
		PublishRegistry: publishRegistry,
		InstallNpmrc:    installNpmrc,
		PublishNpmrc:    publishNpmrc,
	}, nil
}

func validateCredentials(opts NpmConfigOptions) error {
	if opts.Token != "" && (opts.Username != "" || opts.Password != "") {
		return errors.New("either npm username and password or npm token may be specified, but not both")
	}
	if (opts.Username == "") != (opts.Password == "") {
		return errors.New("npm username and password must be specified together")
	}
	if strings.ContainsAny(opts.Token, "\r\n") {
		return errors.New("npm token must not contain a newline")
	}
	return nil
}

func renderNpmrc(registry string, registryURL *url.URL, packageNames []string, auth npmAuth) (string, error) {
	lines := []string{"registry=" + registry + "/"}
	scopes, err := packageScopes(packageNames)
	if err != nil {
		return "", err
	}
	for _, scope := range scopes {
		lines = append(lines, "@"+scope+":registry="+registry+"/")
	}
	if auth.token != "" {
		lines = append(lines, npmAuthConfigKey(registryURL)+"_authToken="+auth.token)
	}
	npmrc := strings.Join(lines, "\n") + "\n"
	if auth.npmrcContent != "" {
		npmrc += auth.npmrcContent
		if !strings.HasSuffix(auth.npmrcContent, "\n") {
			npmrc += "\n"
		}
	}
	return npmrc, nil
}

func packageScopes(packageNames []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, packageName := range packageNames {
		if strings.ContainsAny(packageName, "\r\n") {
			return nil, errors.New("npm package name must not contain a newline")
		}
		if !strings.HasPrefix(packageName, "@") {
			continue
		}
		scope, _, ok := strings.Cut(packageName[1:], "/")
		if !ok || scope == "" {
			return nil, errors.Errorf("npm package %q has an invalid scope", packageName)
		}
		seen[scope] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

func normalizeRegistryURL(rawURL string) (string, *url.URL, error) {
	if rawURL == "" {
		return "", nil, errors.New("registry URL must be specified")
	}
	normalized := strings.TrimRight(rawURL, "/")
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to parse URL")
	}
	if (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return "", nil, errors.New("URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", nil, errors.New("URL must not contain user information, a query, or a fragment")
	}
	return normalized, parsed, nil
}

func npmAuthConfigKey(registryURL *url.URL) string {
	registryPath := strings.TrimRight(registryURL.EscapedPath(), "/")
	return "//" + registryURL.Host + registryPath + "/:"
}
