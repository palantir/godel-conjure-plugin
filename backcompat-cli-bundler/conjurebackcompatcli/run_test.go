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
    "testing"

    "github.com/palantir/godel-conjure-plugin/v5/backcompat-cli-bundler/conjurebackcompatcli"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestConjureBackCompat(t *testing.T) {
    for i, tc := range []struct {
        old          string
        new          string
        isCompatible bool
        want         string
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
            true,
            `{
  "results" : [ ]
}
`,
        },
        {
            `
types:
 definitions:
   default-package: com.palantir.test
   objects:
     Example: { fields: { value1: boolean, value2: boolean } }
services:
  Test:
    name: test
    package: com.palantir.test
    base-path: "/test/v1"
    endpoints:
      get:
       http: GET /some
       returns: Example
`,
            `
types:
 definitions:
   default-package: com.palantir.test
   objects:
     Example: { fields: { value1: boolean } }
services:
  Test:
    name: test
    package: com.palantir.test
    base-path: "/test/v1"
    endpoints:
      get:
       http: GET /some
       returns: Example
`,
            false,
            `{
  "results" : [ {
    "service" : "com.palantir.test.Test",
    "endpoint" : "get",
    "totalChecks" : 10,
    "failures" : [ [ "Checking endpoints 'get' -> 'get'", "Checking return types: com.palantir.test.Example -> com.palantir.test.Example", "Unwrapping named types: com.palantir.test.Example -> com.palantir.test.Example", "Missing 'value2' in proposed (type: boolean)" ] ]
  } ]
}
`,
        },
    } {
        isCompatible, out, err := conjurebackcompatcli.CheckBackcompatYaml([]byte(tc.old), []byte(tc.new))
        require.NoError(t, err, "Case %d", i)
        assert.Equal(t, tc.isCompatible, isCompatible)
        assert.Equal(t, tc.want, string(out), "Case %d\nGot:\n%s", i, out)
    }
}
