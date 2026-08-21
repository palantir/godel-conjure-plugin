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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/palantir/distgo/distgo"
	"github.com/palantir/distgo/pkg/git"
	gitversioner "github.com/palantir/distgo/projectversioner/git"
	"github.com/palantir/godel-conjure-plugin/v7/internal/extensionsprovider"
	"github.com/palantir/godel-conjure-plugin/v7/internal/typescript"
	"github.com/palantir/godel-conjure-plugin/v7/ir-gen-cli-bundler/conjuretypescriptcli"
	"github.com/pkg/errors"
)

const typeScriptNpmDistID = distgo.DistID("npm")

// PackageTypeScriptOptions configures generation of npm package tarballs for projects with TypeScript configuration.
type PackageTypeScriptOptions struct {
	// OutputDir to write npm package tarballs. If empty, a new temporary directory is created and used.
	OutputDir      string
	PackageVersion string
	// NpmUserConfigPath, when non-empty, selects an npm user configuration file for install, build, and pack. It is
	// intended for callers that manage npm configuration, including temporary credentials, outside the package tree.
	NpmUserConfigPath string
}

// TypeScriptPackage describes a generated npm package tarball.
type TypeScriptPackage struct {
	ProjectName string
	PackageName string
	Version     string
	Path        string
}

// PackageTypeScript builds an npm package tarball for every project with TypeScript configuration. If an extensions
// provider is supplied, it enriches each IR using the raw Git project version before product dependencies are read.
// cliGroupID overrides a project's configured group ID when invoking the provider. npm's own output is written to stderr.
func PackageTypeScript(
	params ConjureProjectParams,
	projectDir string,
	opts PackageTypeScriptOptions,
	stdout io.Writer,
	stderr io.Writer,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string,
) ([]TypeScriptPackage, error) {
	return packageTypeScript(params, projectDir, opts, stdout, stderr, typescript.Package, extensionsProvider, cliGroupID)
}

type typeScriptPackager func(irBytes []byte, params typescript.Params, outputDir string, stderr io.Writer) (string, error)

func packageTypeScript(
	params ConjureProjectParams,
	projectDir string,
	opts PackageTypeScriptOptions,
	stdout io.Writer,
	stderr io.Writer,
	packager typeScriptPackager,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string,
) ([]TypeScriptPackage, error) {
	return packageTypeScriptWithVersioner(params, projectDir, opts, stdout, stderr, packager, extensionsProvider, cliGroupID,
		func(projectDir string) (string, error) {
			return gitversioner.New().ProjectVersion(projectDir)
		})
}

type projectVersioner func(projectDir string) (string, error)

func packageTypeScriptWithVersioner(
	params ConjureProjectParams,
	projectDir string,
	opts PackageTypeScriptOptions,
	stdout io.Writer,
	stderr io.Writer,
	packager typeScriptPackager,
	extensionsProvider extensionsprovider.ExtensionsProvider,
	cliGroupID string,
	versioner projectVersioner,
) ([]TypeScriptPackage, error) {
	if !hasTypeScriptProjects(params) {
		return nil, nil
	}
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve project directory")
	}
	packageRoot := opts.OutputDir
	switch {
	case packageRoot == "":
		packageRoot, err = os.MkdirTemp("", "conjure-typescript-")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temporary output directory")
		}
	case !filepath.IsAbs(packageRoot):
		packageRoot = filepath.Join(projectDir, packageRoot)
	}

	gitVersion := ""
	if opts.PackageVersion == "" || extensionsProvider != nil {
		gitVersion, err = versioner(projectDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to determine project version")
		}
		if gitVersion == git.Unspecified {
			return nil, errors.New("unable to determine project version from Git")
		}
	}

	var packages []TypeScriptPackage
	for _, param := range params {
		if param.TypeScript == nil {
			continue
		}
		typeScriptParam := param.TypeScript
		packageVersion := opts.PackageVersion
		if packageVersion == "" {
			var err error
			packageVersion, err = npmPackageVersion(typeScriptParam.NpmVersionScheme, gitVersion)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to determine npm package version for project %q", param.ProjectName)
			}
		}

		projectOutputDir, err := projectPackageOutputDir(packageRoot, param.ProjectName, packageVersion)
		if err != nil {
			return nil, err
		}
		irBytes, err := param.IRProvider.IRBytes()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to obtain IR for project %q", param.ProjectName)
		}
		if extensionsProvider != nil {
			groupID := param.GroupID
			if cliGroupID != "" {
				groupID = cliGroupID
			}
			irBytes, err = addExtensionsToIRBytes(irBytes, extensionsProvider, groupID, param.ProjectName, gitVersion)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to resolve IR extensions for project %q", param.ProjectName)
			}
		}
		productDependencies, err := productDependenciesFromIR(irBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read product dependencies from IR for project %q", param.ProjectName)
		}

		_, _ = fmt.Fprintf(stdout, "Packaging TypeScript client for %q (%s@%s)\n", param.ProjectName, typeScriptParam.PackageName, packageVersion)
		packagePath, err := packager(irBytes, typescript.Params{
			PackageName:                 typeScriptParam.PackageName,
			Version:                     packageVersion,
			ProductDependencies:         productDependencies,
			NpmUserConfigPath:           opts.NpmUserConfigPath,
			FlavorizedAliases:           typeScriptParam.FlavorizedAliases,
			NodeCompatibleModules:       typeScriptParam.NodeCompatibleModules,
			ReadonlyInterfaces:          typeScriptParam.ReadonlyInterfaces,
			GenerateThrowingServices:    typeScriptParam.GenerateThrowingServices,
			GenerateNonThrowingServices: typeScriptParam.GenerateNonThrowingServices,
		}, projectOutputDir, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to package TypeScript client for project %q", param.ProjectName)
		}
		_, _ = fmt.Fprintf(stdout, "Created npm package %s\n", packagePath)
		packages = append(packages, TypeScriptPackage{
			ProjectName: param.ProjectName,
			PackageName: typeScriptParam.PackageName,
			Version:     packageVersion,
			Path:        packagePath,
		})
	}
	return packages, nil
}

func hasTypeScriptProjects(params ConjureProjectParams) bool {
	for _, param := range params {
		if param.TypeScript != nil {
			return true
		}
	}
	return false
}

func productDependenciesFromIR(irBytes []byte) ([]byte, error) {
	var ir struct {
		Extensions map[string]json.RawMessage `json:"extensions"`
	}
	if err := json.Unmarshal(irBytes, &ir); err != nil {
		return nil, errors.Wrap(err, "failed to parse Conjure IR as JSON")
	}
	productDependencies, ok := ir.Extensions["recommended-product-dependencies"]
	if !ok {
		return nil, nil
	}
	var dependencies []json.RawMessage
	if err := json.Unmarshal(productDependencies, &dependencies); err != nil {
		return nil, errors.Wrap(err, "recommended-product-dependencies must be an array")
	}
	if dependencies == nil {
		return nil, errors.New("recommended-product-dependencies must be an array")
	}
	return productDependencies, nil
}

func projectPackageOutputDir(root, projectName, packageVersion string) (string, error) {
	if !isPathSegment(projectName) {
		return "", errors.Errorf("invalid Conjure project name for package output: %q", projectName)
	}
	if !isPathSegment(packageVersion) {
		return "", errors.Errorf("invalid npm package version for package output: %q", packageVersion)
	}
	return distgo.ProductDistOutputDir(
		distgo.ProjectInfo{
			ProjectDir: root,
			Version:    packageVersion,
		},
		distgo.ProductOutputInfo{
			ID:              distgo.ProductID(projectName),
			DistOutputInfos: &distgo.DistOutputInfos{},
		},
		typeScriptNpmDistID,
	), nil
}

func isPathSegment(value string) bool {
	return value != "." && filepath.IsLocal(value) && filepath.Base(value) == value
}

func npmPackageVersion(scheme NpmVersionScheme, gitVersion string) (string, error) {
	switch scheme {
	case "", NpmVersionSchemeGit:
		return gitVersion, nil
	case NpmVersionSchemeGeneratorMajor:
		generatorMajor, err := majorVersion(conjuretypescriptcli.Version)
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse bundled generator version %q", conjuretypescriptcli.Version)
		}
		return fmt.Sprintf("%d0%s", generatorMajor, gitVersion), nil
	default:
		return "", errors.Errorf("unknown npm version scheme %q", scheme)
	}
}

func majorVersion(version string) (int, error) {
	major, _, _ := strings.Cut(version, ".")
	return strconv.Atoi(major)
}
