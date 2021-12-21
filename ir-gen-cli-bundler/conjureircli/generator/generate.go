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
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/palantir/godel-conjure-plugin/v6/ir-gen-cli-bundler/conjureircli/internal"
)

const conjureTgzPath = "../internal/conjure.tgz"

var conjureURL = fmt.Sprintf(
	"https://search.maven.org/remotecontent?filepath=com/palantir/conjure/conjure/%s/conjure-%s.tgz",
	internal.Version, internal.Version)

func main() {
	if err := downloadFile(conjureTgzPath, conjureURL); err != nil {
		panic(err)
	}
}

func downloadFile(filepath string, url string) error {
	if _, err := os.Stat(filepath); err == nil {
		hash := sha256.New()
		existing, err := os.OpenFile(filepath, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		if _, err := io.Copy(hash, existing); err != nil {
			return err
		}
		if sha := fmt.Sprintf("%x", hash.Sum(nil)); sha == internal.SHA256 {
			// existing file up to date
			return nil
		}
	}
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

	hash := sha256.New()
	reader := io.TeeReader(resp.Body, hash)
	if _, err := io.Copy(out, reader); err != nil {
		return err
	}

	if sha := fmt.Sprintf("%x", hash.Sum(nil)); sha != internal.SHA256 {
		return fmt.Errorf("unexpected download sha256 %s", sha)
	}
	return nil
}
