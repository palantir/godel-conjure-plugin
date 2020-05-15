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
{
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
  "services" : [ ]
}`,
            `
{
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
  "services" : [ ]
}`,
            true,
            `{
  "results" : [ ]
}
`,
        },
        {
            `
{
  "version" : 1,
  "errors" : [ ],
  "types" : [ {
    "type" : "object",
    "object" : {
      "typeName" : {
        "name" : "Example",
        "package" : "com.palantir.test"
      },
      "fields" : [ {
        "fieldName" : "value1",
        "type" : {
          "type" : "primitive",
          "primitive" : "BOOLEAN"
        }
      }, {
        "fieldName" : "value2",
        "type" : {
          "type" : "primitive",
          "primitive" : "BOOLEAN"
        }
      } ]
    }
  } ],
  "services" : [ {
    "serviceName" : {
      "name" : "Test",
      "package" : "com.palantir.test"
    },
    "endpoints" : [ {
      "endpointName" : "get",
      "httpMethod" : "GET",
      "httpPath" : "/test/v1/some",
      "args" : [ ],
      "returns" : {
        "type" : "reference",
        "reference" : {
          "name" : "Example",
          "package" : "com.palantir.test"
        }
      },
      "markers" : [ ]
    } ]
  } ]
}`,
            `
{
  "version" : 1,
  "errors" : [ ],
  "types" : [ {
    "type" : "object",
    "object" : {
      "typeName" : {
        "name" : "Example",
        "package" : "com.palantir.test"
      },
      "fields" : [ {
        "fieldName" : "value1",
        "type" : {
          "type" : "primitive",
          "primitive" : "BOOLEAN"
        }
      } ]
    }
  } ],
  "services" : [ {
    "serviceName" : {
      "name" : "Test",
      "package" : "com.palantir.test"
    },
    "endpoints" : [ {
      "endpointName" : "get",
      "httpMethod" : "GET",
      "httpPath" : "/test/v1/some",
      "args" : [ ],
      "returns" : {
        "type" : "reference",
        "reference" : {
          "name" : "Example",
          "package" : "com.palantir.test"
        }
      },
      "markers" : [ ]
    } ]
  } ]
}`,
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
        isCompatible, out, err := conjurebackcompatcli.CheckBackcompatIRs([]byte(tc.old), []byte(tc.new))
        require.NoError(t, err, "Case %d", i)
        assert.Equal(t, tc.isCompatible, isCompatible)
        assert.Equal(t, tc.want, string(out), "Case %d\nGot:\n%s", i, out)
    }
}
