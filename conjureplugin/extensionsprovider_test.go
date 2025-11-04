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

package conjureplugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddExtensionsToIRBytes(t *testing.T) {
	for _, tc := range []struct {
		name               string
		inputIR            string
		providedExtensions map[string]any
		expected           string
	}{
		{
			name:               "no extensions to add, empty IR",
			inputIR:            `{}`,
			providedExtensions: map[string]any{},
			expected:           `{}`,
		},
		{
			name:               "add extension to empty IR",
			inputIR:            `{}`,
			providedExtensions: map[string]any{"hello": "world"},
			expected: `{
	"extensions": {
		"hello": "world"
	}
}`,
		},
		{
			name:               "no extensions to add, empty IR extensions field returns original IR",
			inputIR:            `{"extensions":{}}`,
			providedExtensions: map[string]any{},
			expected:           `{"extensions":{}}`,
		},
		{
			name:               "no extensions to add, IR extensions already present returns original IR",
			inputIR:            `{"extensions":{"already":"present"}}`,
			providedExtensions: map[string]any{},
			expected:           `{"extensions":{"already":"present"}}`,
		},
		{
			name:               "add extension to existing IR extensions",
			inputIR:            `{"extensions":{"already":"present"}}`,
			providedExtensions: map[string]any{"new": "value"},
			expected: `{
	"extensions": {
		"already": "present",
		"new": "value"
	}
}`,
		},
		{
			name:               "add extension to existing IR extensions; other IR fields are preserved",
			inputIR:            `{"a":"b","c":"d","extensions":{"already":"present"}}`,
			providedExtensions: map[string]any{"new": "value"},
			expected: `{
	"a": "b",
	"c": "d",
	"extensions": {
		"already": "present",
		"new": "value"
	}
}`,
		},
		{
			name:               "overwrite existing extensions",
			inputIR:            `{"extensions":{"value":"old"}}`,
			providedExtensions: map[string]any{"value": "new"},
			expected: `{
	"extensions": {
		"value": "new"
	}
}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := addExtensionsToIRBytes(
				[]byte(tc.inputIR),
				func(_ []byte, _, _, _ string) (map[string]any, error) {
					return tc.providedExtensions, nil
				},
				"",
				"",
				"",
			)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(actual))
		})
	}
	for _, tc := range []struct {
		name               string
		inputIR            string
		providedExtensions map[string]any
		wantErr            string
	}{
		{
			name:    "invalid input IR: empty string",
			inputIR: ``,
			wantErr: `EOF`,
		},
		{
			name:    "invalid input IR: not a JSON object",
			inputIR: `[]`,
			wantErr: `json: cannot unmarshal array into Go value of type map[string]interface {}`,
		},
		{
			name:    "invalid input IR: extensions does not map to an object",
			inputIR: `{"extensions":[]}`,
			wantErr: `the provided Conjure IR has an "extensions" field that is not a map, which is a violation of the Conjure spec; see https://github.com/palantir/conjure/blob/master/docs/spec/intermediate_representation.md#extensions for details`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := addExtensionsToIRBytes(
				[]byte(tc.inputIR),
				func(_ []byte, _, _, _ string) (map[string]any, error) {
					return map[string]any{"key": "value"}, nil
				},
				"",
				"",
				"",
			)
			assert.EqualError(t, err, tc.wantErr)
		})
	}
}
