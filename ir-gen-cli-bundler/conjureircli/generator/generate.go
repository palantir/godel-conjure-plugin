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
	"github.com/palantir/pkg/clipackager"
)

const conjureTGZVersion = "4.51.0"

func main() {
	if err := clipackager.RunGenerate(
		"conjureircli",
		"ir-gen-cli-bundler/conjureircli/generator",
		"../asset/version.go",
		conjureTGZVersion,
		"../asset/conjure.tgz",
		"https://search.maven.org/remotecontent?filepath=com/palantir/conjure/conjure/"+conjureTGZVersion+"/conjure-"+conjureTGZVersion+".tgz",
	); err != nil {
		panic(err)
	}
}
