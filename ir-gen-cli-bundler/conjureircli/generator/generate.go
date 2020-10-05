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

//go:generate go run $GOFILE

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	"github.com/go-bindata/go-bindata"
)

const conjureVersion = "4.9.0"

func main() {
	versionFilePath := "../internal/version.go"
	newVersionFileContent := fmt.Sprintf(`// This is a generated file: do not edit by hand.
// To update this file, run the generator in conjureircli/generator.
package conjureircli_internal

const Version = "%s"
`, conjureVersion)

	// version file exists and is in desired state: assume that all generated content is in desired state
	if currVersionFileContent, err := ioutil.ReadFile(versionFilePath); err == nil && string(currVersionFileContent) == newVersionFileContent {
		return
	}

	conjureTgzPath := "conjure.tgz"
	defer func() {
		_ = os.Remove(conjureTgzPath)
	}()

	if err := downloadFile(conjureTgzPath, fmt.Sprintf("https://palantir.bintray.com/releases/com/palantir/conjure/conjure/%s/conjure-%s.tgz", conjureVersion, conjureVersion)); err != nil {
		panic(err)
	}

	if err := bindata.Translate(&bindata.Config{
		Input: []bindata.InputConfig{
			{
				Path: ".",
			},
		},
		Ignore: []*regexp.Regexp{
			regexp.MustCompile(`.*\.go`),
		},
		NoCompress: true,
		Output:     "../internal/bindata.go",
		Package:    "conjureircli_internal",
	}); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(versionFilePath, []byte(newVersionFileContent), 0644); err != nil {
		panic(err)
	}
}

func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}
