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
	"net/url"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// npmrcConfig specifies npm registry inputs. PublishRegistry is the destination used by npm publish.
// InstallRegistry may select a different repository for dependency resolution and defaults to PublishRegistry.
type npmrcConfig struct {
	PackageNames    []string
	PublishRegistry string
	InstallRegistry string
}

// npmrc contains separate npm configuration for dependency installation and publication. Registry values are
// normalized absolute URLs without trailing slashes.
type npmrc struct {
	InstallContents string
	PublishContents string
	InstallRegistry string
	PublishRegistry string
}

// tokenCredential is an npm auth token rendered per-registry via npmAuthConfigKey.
type tokenCredential struct {
	key        string
	value      string
	alwaysAuth bool
}

type npmAuth struct {
	token    *tokenCredential
	fragment string
}

func generateNpmrc(cfg npmrcConfig, auth npmAuth) (npmrc, error) {
	publishRegistry, publishURL, err := normalizeRegistryURL(cfg.PublishRegistry)
	if err != nil {
		return npmrc{}, errors.Wrap(err, "invalid npm publish registry")
	}
	installRegistry := publishRegistry
	installURL := publishURL
	if cfg.InstallRegistry != "" {
		installRegistry, installURL, err = normalizeRegistryURL(cfg.InstallRegistry)
		if err != nil {
			return npmrc{}, errors.Wrap(err, "invalid npm install registry")
		}
	}
	// Compared after normalization, not on whether InstallRegistry was set: a caller that explicitly passes the same
	// registry for both install and publish (rather than omitting InstallRegistry) must still be treated as sharing
	// credentials between them.
	distinctInstallRegistry := installRegistry != publishRegistry
	// Credentials -- whether a directly supplied token or a provider exchange response -- are always included in the
	// publish npmrc content, and in the install npmrc content too when the two registries are the same (the
	// default). A distinct install registry always gets neither: reusing a publish token there risks disclosing it
	// to an unrelated registry, and provider responses are obtained specifically for the publish registry.
	installAuth := auth
	if distinctInstallRegistry {
		installAuth = npmAuth{}
	}
	installContents, err := renderNpmrc(installRegistry, installURL, cfg.PackageNames, installAuth.token, installAuth.fragment)
	if err != nil {
		return npmrc{}, err
	}
	publishContents, err := renderNpmrc(publishRegistry, publishURL, cfg.PackageNames, auth.token, auth.fragment)
	if err != nil {
		return npmrc{}, err
	}
	return npmrc{
		InstallContents: installContents,
		PublishContents: publishContents,
		InstallRegistry: installRegistry,
		PublishRegistry: publishRegistry,
	}, nil
}

func renderNpmrc(registry string, registryURL *url.URL, packageNames []string, token *tokenCredential, authFragment string) (string, error) {
	lines := []string{"registry=" + registry + "/"}
	scopes, err := packageScopes(packageNames)
	if err != nil {
		return "", err
	}
	for _, scope := range scopes {
		lines = append(lines, "@"+scope+":registry="+registry+"/")
	}
	if token != nil {
		lines = append(lines, npmAuthConfigKey(registryURL)+token.key+"="+token.value)
		if token.alwaysAuth {
			lines = append(lines, "always-auth=true")
		}
	}
	contents := strings.Join(lines, "\n") + "\n"
	if authFragment != "" {
		contents += authFragment
		if !strings.HasSuffix(authFragment, "\n") {
			contents += "\n"
		}
	}
	return contents, nil
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
		slashIndex := strings.Index(packageName, "/")
		if slashIndex <= 1 {
			continue
		}
		scope := packageName[1:slashIndex]
		for _, char := range scope {
			if !IsNpmScopeCharacter(char) {
				return nil, errors.Errorf("npm package %q has an invalid scope", packageName)
			}
		}
		seen[scope] = struct{}{}
	}
	scopes := make([]string, 0, len(seen))
	for scope := range seen {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes, nil
}

// IsNpmScopeCharacter reports whether char belongs to the plugin's deliberately restricted npm scope character set:
// lowercase ASCII letters, digits, and "-._~". These RFC 3986 unreserved characters can be included literally in an
// Artifactory auth URL. This is intentionally narrower than every scope npm may accept.
func IsNpmScopeCharacter(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || strings.ContainsRune("-._~", char)
}

func npmAuthConfigKey(registryURL *url.URL) string {
	registryPath := strings.TrimRight(registryURL.EscapedPath(), "/")
	return "//" + registryURL.Host + registryPath + "/:"
}

// normalizeRegistryURL validates rawURL as an absolute HTTP(S) registry URL with no embedded user info, query, or
// fragment, and returns it with any trailing slash removed.
func normalizeRegistryURL(rawURL string) (string, *url.URL, error) {
	if rawURL == "" {
		return "", nil, errors.New("registry URL must be specified")
	}
	return normalizeHTTPURL(rawURL)
}

func normalizeHTTPURL(rawURL string) (string, *url.URL, error) {
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
