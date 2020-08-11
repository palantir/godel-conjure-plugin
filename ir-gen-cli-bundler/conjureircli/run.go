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

package conjureircli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/mholt/archiver"
	conjureircli_internal "github.com/palantir/godel-conjure-plugin/v5/ir-gen-cli-bundler/conjureircli/internal"
	"github.com/pkg/errors"
)

func YAMLtoIR(in []byte) (rBytes []byte, rErr error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); rErr == nil && err != nil {
			rErr = errors.Wrapf(err, "failed to remove temporary directory")
		}
	}()

	inPath := path.Join(tmpDir, "in.yml")
	if err := ioutil.WriteFile(inPath, in, 0644); err != nil {
		return nil, errors.WithStack(err)
	}
	return InputPathToIR(inPath)
}

func InputPathToIR(inPath string) (rBytes []byte, rErr error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); rErr == nil && err != nil {
			rErr = errors.Wrapf(err, "failed to remove temporary directory")
		}
	}()

	outPath := path.Join(tmpDir, "out.json")
	if err := Run(inPath, outPath); err != nil {
		return nil, err
	}
	irBytes, err := ioutil.ReadFile(outPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return irBytes, nil
}

func Run(inPath, outPath string) error {
	cliPath, err := cliCmdPath()
	if err != nil {
		return err
	}
	if err := ensureCLIExists(cliPath); err != nil {
		return err
	}
	cmd := exec.Command(cliPath, "compile", inPath, outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to execute %v\nOutput:\n%s", cmd.Args, string(output))
	}
	return nil
}

// cliUnpackDir is the directory into which the tarball is unpacked
var cliUnpackDir = path.Join(os.TempDir(), "_conjureircli")

// cliArchiveDir is the top-level directory of the unpacked archive
var cliArchiveDir = path.Join(cliUnpackDir, fmt.Sprintf("conjure-%v", conjureircli_internal.Version))

// cliCmdPath is the path to the conjure compiler executable
func cliCmdPath() (string, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		return path.Join(cliArchiveDir, "bin", "conjure"), nil
	default:
		return "", errors.Errorf("OS %s not supported", runtime.GOOS)
	}
}

// ensureCLIExists installs the conjure compiler if it does not already exist or it appears malformed.
func ensureCLIExists(cliPath string) error {
	if checkCliExists(cliPath) == nil {
		// destination already exists
		return nil
	}

	// destination does not exist or is malformed, remove the archive dir just in case of a previous bad install
	if err := os.RemoveAll(cliArchiveDir); err != nil {
		return errors.Wrap(err, "failed to remove destination dir before unpacking cli archive")
	}

	// expand asset into destination
	tgzBytes, err := conjureircli_internal.Asset("conjure.tgz")
	if err != nil {
		return errors.WithStack(err)
	}
	if err := archiver.TarGz.Read(bytes.NewReader(tgzBytes), cliUnpackDir); err != nil {
		return errors.WithStack(err)
	}

	// check that we can now find the cli
	if err := checkCliExists(cliPath); err != nil {
		return errors.Wrap(err, "failed to stat cli file after unpacking; please comment on godel-conjure-plugin#84 and retry")
	}

	return nil
}

// checkCliExists returns an error if the cliPath is not a regular file with nonzero size.
func checkCliExists(cliPath string) error {
	fi, err := os.Stat(cliPath)
	switch {
	case err != nil:
		return err
	case fi.Size() == 0:
		return errors.New("file was empty")
	case !fi.Mode().IsRegular():
		return fmt.Errorf("file mode %s was unexpected", fi.Mode().String())
	}
	return nil
}
