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

package conjurebackcompatcli_test

import (
	"io/ioutil"
	"testing"

	"github.com/palantir/godel-conjure-plugin/v5/backcompat-cli-bundler/conjurebackcompatcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConjureBackCompat(t *testing.T) {
	for i, tc := range []struct {
		old  string
		new  string
		want string
	}{
		{
			`
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      BooleanExample: { fields: { value: boolean } }
`,
			`
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      BooleanExample: { fields: { value: boolean } }
`,
			`{
  "results" : [ ]
}
`,
		},
	} {
		isCompatible, out, err := conjurebackcompatcli.CheckBackcompatYaml([]byte(tc.old), []byte(tc.new))
		require.NoError(t, err, "Case %d", i)
		assert.True(t, isCompatible)
		assert.Equal(t, tc.want, string(out), "Case %d\nGot:\n%s", i, out)
	}
}

func TestConjureBackCompat2(t *testing.T) {
	for i, tc := range []struct {
		old  string
		new  string
		want string
	}{
		{
			`
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      BooleanExample: { fields: { value: boolean } }
`,
			`
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      BooleanExample: { fields: { value: boolean } }
`,
			`{
  "results" : [ ]
}`,
		},
		{
			`
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      Example: { fields: { value1: boolean, value2: boolean } }
`,
			`
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      BooleanExample: { fields: { value1: boolean } }
`,
			`{
  "results" : [ ]
}`,
		},
	} {
		old, err := ioutil.ReadFile("/Users/hsaraogi/gowork/src/github.com/palantir/godel-conjure-plugin/backcompat-cli-bundler/test_ressources/skylab1.yml")
		require.NoError(t, err)
		new, err := ioutil.ReadFile("/Users/hsaraogi/gowork/src/github.com/palantir/godel-conjure-plugin/backcompat-cli-bundler/test_ressources/skylab2.yml")
		require.NoError(t, err)

		isCompatible, out, err := conjurebackcompatcli.CheckBackcompatYaml(old, new)
		require.NoError(t, err, "Case %d", i)
		assert.False(t, isCompatible)
		assert.Equal(t, tc.want, string(out), "Case %d\nGot:\n%s", i, out)
	}
}
