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

package conjureircli_test

import (
	"testing"

	"github.com/palantir/godel-conjure-plugin/v6/ir-gen-cli-bundler/conjureircli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLtoIR(t *testing.T) {
	for i, tc := range []struct {
		in   string
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
			`{
  "version" : 1,
  "errors" : [ ],
  "types" : [ {
    "type" : "object",
    "object" : {
      "typeName" : {
        "name" : "BooleanExample",
        "package" : "com.palantir.conjure"
      },
      "fields" : [ {
        "fieldName" : "value",
        "type" : {
          "type" : "primitive",
          "primitive" : "BOOLEAN"
        }
      } ]
    }
  } ],
  "services" : [ ],
  "extensions" : { }
}`,
		},
	} {
		got, err := conjureircli.YAMLtoIR([]byte(tc.in))
		require.NoError(t, err, "Case %d", i)
		assert.Equal(t, tc.want, string(got), "Case %d\nGot:\n%s", i, got)
	}
}
