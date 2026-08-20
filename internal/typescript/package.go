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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/palantir/godel-conjure-plugin/v7/ir-gen-cli-bundler/conjuretypescriptcli"
	"github.com/pkg/errors"
)

// Params configures generation of a single Conjure project's TypeScript client.
type Params struct {
	PackageName         string
	Version             string
	ProductDependencies []byte
	// NpmUserConfigPath, when non-empty, is supplied to npm through NPM_CONFIG_USERCONFIG.
	NpmUserConfigPath string

	FlavorizedAliases           bool
	NodeCompatibleModules       bool
	ReadonlyInterfaces          bool
	GenerateThrowingServices    bool
	GenerateNonThrowingServices bool
}

func (p Params) generateOptions(productDependenciesPath string) conjuretypescriptcli.GenerateOptions {
	return conjuretypescriptcli.GenerateOptions{
		PackageName:                 p.PackageName,
		PackageVersion:              p.Version,
		ProductDependenciesPath:     productDependenciesPath,
		FlavorizedAliases:           p.FlavorizedAliases,
		NodeCompatibleModules:       p.NodeCompatibleModules,
		ReadonlyInterfaces:          p.ReadonlyInterfaces,
		GenerateThrowingServices:    p.GenerateThrowingServices,
		GenerateNonThrowingServices: p.GenerateNonThrowingServices,
	}
}

// GenerateProject writes the generated TypeScript project into outDir, which must already exist.
func GenerateProject(irBytes []byte, params Params, outDir string) error {
	irPath, productDependenciesPath, cleanup, err := writeTempInputs(irBytes, params.ProductDependencies)
	if err != nil {
		return err
	}
	defer cleanup()
	return conjuretypescriptcli.Generate(irPath, outDir, params.generateOptions(productDependenciesPath))
}

// Package generates and compiles a TypeScript client, then runs npm pack. The returned path identifies the exact
// tarball produced by npm. npm's own output (both its stdout and stderr) is written to stderr.
func Package(irBytes []byte, params Params, outputDir string, stderr io.Writer) (packagePath string, rErr error) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		return "", errors.WithStack(err)
	}

	workDir, err := os.MkdirTemp("", "conjure-typescript-")
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer func() {
		if err := os.RemoveAll(workDir); rErr == nil && err != nil {
			rErr = errors.Wrap(err, "failed to remove temporary TypeScript build directory")
		}
	}()

	if err := GenerateProject(irBytes, params, workDir); err != nil {
		return "", err
	}
	if err := runNpm(workDir, params.NpmUserConfigPath, stderr, "install", "--no-package-lock", "--no-production"); err != nil {
		return "", err
	}
	if err := runNpm(workDir, params.NpmUserConfigPath, stderr, "run-script", "build"); err != nil {
		return "", err
	}
	return runNpmPack(workDir, absOutputDir, params.NpmUserConfigPath, stderr)
}

func runNpm(workDir, npmUserConfigPath string, stderr io.Writer, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = workDir
	cmd.Env = npmCommandEnv(npmUserConfigPath)
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to run npm %v", args)
	}
	return nil
}

func runNpmPack(workDir, outputDir, npmUserConfigPath string, stderr io.Writer) (string, error) {
	cmd := exec.Command("npm", "pack", "--json")
	cmd.Dir = workDir
	cmd.Env = npmCommandEnv(npmUserConfigPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "failed to run npm pack: %s", stdout.String())
	}
	var results []struct {
		Filename string `json:"filename"`
		Files    []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return "", errors.Wrapf(err, "failed to parse npm pack output: %s", stdout.String())
	}
	if len(results) != 1 || results[0].Filename == "" || filepath.Base(results[0].Filename) != results[0].Filename {
		return "", errors.Errorf("npm pack returned an invalid package filename: %s", stdout.String())
	}
	for _, file := range results[0].Files {
		if filepath.Base(filepath.Clean(file.Path)) == ".npmrc" {
			return "", errors.New("npm pack attempted to include .npmrc in the package")
		}
	}

	workPackagePath := filepath.Join(workDir, results[0].Filename)
	info, err := os.Stat(workPackagePath)
	if err != nil {
		return "", errors.Wrap(err, "npm pack did not create the reported package tarball")
	}
	if !info.Mode().IsRegular() {
		return "", errors.Errorf("npm pack output is not a regular file: %s", workPackagePath)
	}
	packagePath := filepath.Join(outputDir, results[0].Filename)
	if err := copyPackageFile(workPackagePath, packagePath); err != nil {
		return "", err
	}
	return packagePath, nil
}

func npmCommandEnv(npmUserConfigPath string) []string {
	if npmUserConfigPath == "" {
		return os.Environ()
	}
	const userConfigPrefix = "NPM_CONFIG_USERCONFIG="
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, userConfigPrefix) {
			env = append(env, entry)
		}
	}
	env = append(env, userConfigPrefix+npmUserConfigPath)
	return env
}

func copyPackageFile(sourcePath, destinationPath string) (rErr error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return errors.Wrap(err, "failed to open npm package tarball")
	}
	defer func() {
		if err := source.Close(); rErr == nil && err != nil {
			rErr = errors.Wrap(err, "failed to close npm package tarball")
		}
	}()

	tmpDestination, err := os.CreateTemp(filepath.Dir(destinationPath), ".conjure-typescript-package-")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary npm package output")
	}
	tmpDestinationPath := tmpDestination.Name()
	defer func() { _ = os.Remove(tmpDestinationPath) }()
	if _, err := io.Copy(tmpDestination, source); err != nil {
		_ = tmpDestination.Close()
		return errors.Wrap(err, "failed to copy npm package tarball")
	}
	if err := tmpDestination.Chmod(0644); err != nil {
		_ = tmpDestination.Close()
		return errors.Wrap(err, "failed to set npm package permissions")
	}
	if err := tmpDestination.Close(); err != nil {
		return errors.Wrap(err, "failed to close temporary npm package output")
	}
	if err := os.Rename(tmpDestinationPath, destinationPath); err != nil {
		return errors.Wrap(err, "failed to move npm package into output directory")
	}
	return nil
}

func writeTempInputs(irBytes, productDependencies []byte) (irPath, productDependenciesPath string, cleanup func(), _ error) {
	tmpDir, err := os.MkdirTemp("", "conjure-typescript-input-")
	if err != nil {
		return "", "", nil, errors.WithStack(err)
	}
	irPath = filepath.Join(tmpDir, "ir.json")
	if err := os.WriteFile(irPath, irBytes, 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", "", nil, errors.WithStack(err)
	}
	if productDependencies != nil {
		productDependenciesPath = filepath.Join(tmpDir, "product-dependencies.json")
		if err := os.WriteFile(productDependenciesPath, productDependencies, 0644); err != nil {
			_ = os.RemoveAll(tmpDir)
			return "", "", nil, errors.WithStack(err)
		}
	}
	return irPath, productDependenciesPath, func() { _ = os.RemoveAll(tmpDir) }, nil
}
