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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// PublishPackage publishes the packagePath tarball to registry using "npm publish".
func PublishPackage(packagePath, registry, npmUserConfigPath string, stdout io.Writer) (rErr error) {
	// npm requires the trailing slash when computing the nerf-darted cred lookup key, otherwise the last path
	// segment will be stripped and will fail to authenticate.
	registry = strings.TrimRight(registry, "/") + "/"

	absPackagePath, err := filepath.Abs(packagePath)
	if err != nil {
		return errors.Wrap(err, "failed to resolve npm package path")
	}
	packageInfo, err := os.Stat(absPackagePath)
	if err != nil {
		return errors.Wrap(err, "failed to inspect npm package tarball")
	}
	if !packageInfo.Mode().IsRegular() {
		return errors.Errorf("npm package tarball is not a regular file: %s", absPackagePath)
	}

	workDir, err := os.MkdirTemp("", "conjure-typescript-publish-")
	if err != nil {
		return errors.Wrap(err, "failed to create npm publish working directory")
	}
	defer func() {
		if err := os.RemoveAll(workDir); rErr == nil && err != nil {
			rErr = errors.Wrap(err, "failed to remove npm publish working directory")
		}
	}()

	cmd := exec.Command("npm", "publish", absPackagePath, "--registry", registry)
	cmd.Dir = workDir
	cmd.Env = npmCommandEnv(npmUserConfigPath)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to publish npm package %s to %s", absPackagePath, registry)
	}
	return nil
}
