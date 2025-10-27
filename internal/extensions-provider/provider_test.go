// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package extensionsprovider_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	extensionsprovider "github.com/palantir/godel-conjure-plugin/v6/internal/extensions-provider"
	"github.com/palantir/godel-conjure-plugin/v6/internal/tempfilecreator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish(t *testing.T) {
	const (
		validButNotExtensionsAsset = `#!/bin/sh

if [ "$#" -ne 1 ]; then
    exit 1
fi

if [ "$1" = "_assetInfo" ]; then
    printf '%s\n' '{ "type": "valid type that follows the spec but is not supported yet so it should just not be called again and everything will exit OK" }'
    exit 0
fi

printf '%s\n' 'test failed: should be unreachable'
exit 1
`
	)

	var assets []string
	for _, assetContent := range [...]string{
		createExtensionsAsset(`{"foo":"bar"}`),
		createExtensionsAsset(`{"overwritten":"will get overwritten"}`),
		createExtensionsAsset(`{"overwritten":"will be present"}`),
		createExtensionsAsset(`{"work for ints":1}`),
		createExtensionsAsset(`{"work for obj":{"other":"object"}}`),
		createExtensionsAsset(`{"work for null":null}`),
		createExtensionsAsset(`{}`),
		validButNotExtensionsAsset,
	} {
		asset := tempfilecreator.MustWriteBytesToTempFile([]byte(assetContent))
		require.NoError(t, os.Chmod(asset, 0700))

		assets = append(assets, asset)
	}

	provider := extensionsprovider.New("", assets, "")

	got, err := provider([]byte{}, "", "", "")
	require.NoError(t, err)

	want := map[string]any{
		"foo":           "bar",
		"overwritten":   "will be present",
		"work for ints": json.Number("1"),
		"work for null": nil,
		"work for obj": map[string]any{
			"other": "object",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("maps are not equal:\ngot: %v\nwant:%v", got, want)
	}

	empty, err := extensionsprovider.New("", nil, "")([]byte{}, "", "", "")
	assert.NoError(t, err)
	assert.Empty(t, empty)
}

func createExtensionsAsset(assetReturnValue string) string {
	return fmt.Sprintf(`#!/bin/sh

if [ "$#" -ne 1 ]; then
    exit 1
fi

if [ "$1" = "_assetInfo" ]; then
    printf '%%s\n' '{ "type": "conjure-ir-extensions-provider" }'
    exit 0
fi

printf '%%s\n' '%s'
exit 0
`, assetReturnValue)
}
