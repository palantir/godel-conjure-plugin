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

package conjurebackcompatcli

import (
	"fmt"
	"github.com/palantir/godel-conjure-plugin/v5/commons"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	conjurebackcompatcli_internal "github.com/palantir/godel-conjure-plugin/v5/backcompat-cli-bundler/conjurebackcompatcli/internal"
	"github.com/palantir/godel-conjure-plugin/v5/ir-gen-cli-bundler/conjureircli"
	"github.com/pkg/errors"
)

const ConjureBackcompatJarPath = "conjure-backcompat.jar"
const OldIRPath = "old-ir.json"


func CheckBackcompat(current []byte, groupName, repoName, conjureIRDir string) (isCompatible bool, rBytes []byte, rErr error) {
	old, err := getLastTagIR(groupName, repoName, conjureIRDir)
	if err != nil {
		return false, nil, err
	}
	return CheckBackcompatYaml(old, current)
}

func CheckBackcompatYaml(old []byte, new []byte) (isCompatible bool, rBytes []byte, rErr error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to create temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); rErr == nil && err != nil {
			rErr = errors.Wrapf(err, "failed to remove temporary directory")
		}
	}()
	oldIRBytes, err := conjureircli.YAMLtoIR(old)
	if err != nil {
		return false, nil, errors.Wrapf(err, "Failed to convert old yaml bytes to IR")
	}
	oldIRPath := path.Join(tmpDir, "old-ir.json")
	if err := ioutil.WriteFile(oldIRPath, oldIRBytes, 0644); err != nil {
		return false, nil, errors.WithStack(err)
	}

	newIRBytes, err := conjureircli.YAMLtoIR(new)
	if err != nil {
		return false, nil, errors.Wrapf(err, "Failed to convert new yaml bytes to IR")
	}
	newIRPath := path.Join(tmpDir, "new-ir.json")
	if err := ioutil.WriteFile(newIRPath, newIRBytes, 0644); err != nil {
		return false, nil, errors.WithStack(err)
	}

	return CheckBackcompatPaths(oldIRPath, newIRPath)
}

func CheckBackcompatPaths(oldPath string, newPath string) (isCompatible bool, rBytes []byte, rErr error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to create temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); rErr == nil && err != nil {
			rErr = errors.Wrapf(err, "failed to remove temporary directory")
		}
	}()
	return IsCompatible(oldPath, newPath)
}

func IsCompatible(oldPath, newPath string) (isCompatible bool, rBytes []byte, rErr error) {
	cliPath, err := cliCmdPath()
	if err != nil {
		return false, nil, err
	}
	cmd := exec.Command("java",
		"-jar",
		cliPath,
		fmt.Sprintf("--old=%s", oldPath),
		fmt.Sprintf("--proposed=%s", newPath))
	output, err := cmd.CombinedOutput()
	if err == nil {
		return true, output, nil
	} else if err.Error() == "exit status 1" {
		return false, output, nil
	} else {
		return false, nil, errors.Wrapf(err, "failed to execute %v\nOutput:\n%s", cmd.Args, string(output))
	}
}

func getLastTagIR(groupName, repoName, conjureIRDir string) ([]byte, error) {
	defer func() {
		_ = os.Remove(OldIRPath)
	}()
	version, err := getLastTag(conjureIRDir)
	if err != nil {
		return nil, err
	}

	group := strings.Replace(groupName, ".", "/", -1)
	artifactPath := strings.Join([]string{group, repoName, version}, "/")
	artifactName := fmt.Sprintf("%s-%s%s", repoName, version, ".conjure.json")
	url := fmt.Sprintf("%s/%s", "https://artifactory.palantir.build/artifactory/internal-conjure-release", path.Join(artifactPath, artifactName))

	if err := commons.DownloadFile(OldIRPath, url); err != nil {
		return nil, errors.Wrapf(err, "failed to download old IR\n url:%s", url)
	}
	return ioutil.ReadFile(OldIRPath)
}

func getLastTag(projectDir string) (string, error) {
	cmd := exec.Command("git", "describe", "--abbrev=0", "--tags")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get last tag %v\nOutput:\n%s\nDir:%s", cmd.Args, string(output), projectDir)
	}
	return string(output), nil
}

func cliCmdPath() (string, error) {
	cacheDirPath := path.Join(os.TempDir(), "__conjurebackcompatcli")
	dstPath := path.Join(cacheDirPath, fmt.Sprintf("conjure-backcompat-cli-%v.jar", conjurebackcompatcli_internal.Version))
	if err := ensureCLIExists(dstPath); err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin", "linux":
		return dstPath, nil
	default:
		return "", errors.Errorf("OS %s not supported", runtime.GOOS)
	}
}

func ensureCLIExists(dstPath string) error {
	if fi, err := os.Stat(dstPath); err == nil && !fi.IsDir() {
		// destination already exists
		return nil
	}

	// expand asset into destination
	jarBytes, err := conjurebackcompatcli_internal.Asset(ConjureBackcompatJarPath)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = os.Mkdir(path.Dir(dstPath), 0777)
	return errors.WithStack(ioutil.WriteFile(dstPath, jarBytes, 0777))
}
