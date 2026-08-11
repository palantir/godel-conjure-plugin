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

package conjuretypescriptcli

import (
	_ "embed" // required for go:embed directive
	"fmt"

	"github.com/palantir/godel-conjure-plugin/v7/ir-gen-cli-bundler/conjuretypescriptcli/asset"
	"github.com/palantir/pkg/clipackager"
	"github.com/pkg/errors"
)

// Version is the version of the bundled conjure-typescript generator.
const Version = asset.Version

var (
	//go:embed asset/conjure-typescript.tgz
	conjureTypeScriptCLITGZ []byte

	// CLI runner that runs the conjure-typescript CLI.
	cliRunner = clipackager.NewDefaultPackagedCLIRunner(
		"conjure-typescript",
		asset.Version,
		conjureTypeScriptCLITGZ,
		".tgz",
	)
)

// GenerateOptions configures the "generate" command of the conjure-typescript CLI. PackageName and PackageVersion
// are required by the CLI (it validates that PackageVersion is a valid SLS version).
type GenerateOptions struct {
	PackageName    string
	PackageVersion string
	// ProductDependenciesPath is an optional path to a JSON file containing the SLS service dependencies to embed in the
	// generated package.json's "sls.dependencies" block. If empty, the "--productDependencies" flag is omitted.
	ProductDependenciesPath string

	FlavorizedAliases     bool
	NodeCompatibleModules bool
	ReadonlyInterfaces    bool
	// GenerateThrowingServices defaults to true in the CLI
	GenerateThrowingServices    bool
	GenerateNonThrowingServices bool
}

// Generate invokes the "generate" command of the conjure-typescript CLI, reading the Conjure IR at irPath and writing
// the generated TypeScript project into outDir. outDir must already exist since the CLI errors if it does not.
func Generate(irPath, outDir string, opts GenerateOptions) error {
	args := []string{
		"generate",
		irPath,
		outDir,
		"--packageName=" + opts.PackageName,
		"--packageVersion=" + opts.PackageVersion,
	}
	if opts.ProductDependenciesPath != "" {
		args = append(args, "--productDependencies="+opts.ProductDependenciesPath)
	}
	if opts.FlavorizedAliases {
		args = append(args, "--flavorizedAliases")
	}
	if opts.NodeCompatibleModules {
		args = append(args, "--nodeCompatibleModules")
	}
	if opts.ReadonlyInterfaces {
		args = append(args, "--readonlyInterfaces")
	}
	// These two default to true/false respectively in the CLI, so pass explicit values to keep behavior deterministic.
	args = append(args, fmt.Sprintf("--generateThrowingServices=%t", opts.GenerateThrowingServices))
	args = append(args, fmt.Sprintf("--generateNonThrowingServices=%t", opts.GenerateNonThrowingServices))

	if cliPath, output, err := clipackager.RunPackagedCLI(cliRunner, args...); err != nil {
		return errors.Wrapf(err, "failed to execute %v\nOutput:\n%s", append([]string{cliPath}, args...), string(output))
	}
	return nil
}
