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

package conjureplugin_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/palantir/distgo/distgo"
	"github.com/palantir/distgo/publisher"
	"github.com/palantir/distgo/publisher/artifactory"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v6/conjureplugin/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestPublish(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tmpDir, err := ioutil.TempDir(cwd, "TestPublishConjure_")
	require.NoError(t, err)
	ymlDir := path.Join(tmpDir, "yml_dir")
	err = os.Mkdir(ymlDir, 0755)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	conjureConfigYML := []byte(`
types:
  definitions:
    default-package: com.palantir.base.api
    objects:
      BaseType:
        fields:
          id: string
`)

	pluginConfigYML := []byte(`
projects:
  project-1:
    output-dir: ` + tmpDir + `/conjure
    ir-locator: ` + ymlDir + `
`)

	conjureFile := filepath.Join(ymlDir, "api.yml")
	err = ioutil.WriteFile(conjureFile, conjureConfigYML, 0644)
	require.NoError(t, err, "failed to write api file")

	var cfg config.ConjurePluginConfig
	require.NoError(t, yaml.Unmarshal(pluginConfigYML, &cfg))
	params, err := cfg.ToParams(io.Discard)
	require.NoError(t, err, "failed to parse config set")

	outputBuf := &bytes.Buffer{}
	groupID := "com.palantir.foo"
	err = conjureplugin.Publish(params, tmpDir, map[distgo.PublisherFlagName]interface{}{
		publisher.ConnectionInfoURLFlag.Name:     "http://artifactory.domain.com",
		publisher.GroupIDFlag.Name:               groupID,
		artifactory.PublisherRepositoryFlag.Name: "repo",
	}, true, outputBuf, func(_ []byte, _, _, _ string) (map[string]any, error) {
		return nil, nil
	}, &groupID)
	require.NoError(t, err, "failed to publish Conjure")

	lines := strings.Split(outputBuf.String(), "\n")
	assert.Equal(t, 3, len(lines), "Expected output to have 3 lines:\n%s", outputBuf.String())

	wantRegexp := regexp.QuoteMeta("[DRY RUN]") + " Uploading .*?" + regexp.QuoteMeta(".conjure.json") + " to " + regexp.QuoteMeta("http://artifactory.domain.com/artifactory/repo/com/palantir/foo/project-1/") + ".*?" + regexp.QuoteMeta("/project-1-") + ".*?" + regexp.QuoteMeta(".conjure.json")
	assert.Regexp(t, wantRegexp, lines[0])

	wantRegexp = regexp.QuoteMeta("[DRY RUN]") + " Uploading to " + regexp.QuoteMeta("http://artifactory.domain.com/artifactory/repo/com/palantir/foo/") + ".*?" + regexp.QuoteMeta(".pom")
	assert.Regexp(t, wantRegexp, lines[1])
}

func TestAddExtensionsToIrBytes(t *testing.T) {
	for _, tc := range [...]struct {
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
			name:               "no extensions to add, empty IR extensions field",
			inputIR:            `{"extensions":{}}`,
			providedExtensions: map[string]any{},
			expected: `{
	"extensions": {}
}`,
		},
		{
			name:               "no extensions to add, IR extensions already present",
			inputIR:            `{"extensions":{"already":"present"}}`,
			providedExtensions: map[string]any{},
			expected: `{
	"extensions": {
		"already": "present"
	}
}`,
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
			actual, err := conjureplugin.AddExtensionsToIrBytes(
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
	for _, tc := range [...]struct {
		name               string
		inputIR            string
		providedExtensions map[string]any
	}{
		{
			name:    "invalid input IR: empty string",
			inputIR: ``,
		},
		{
			name:    "invalid input IR: not a JSON object",
			inputIR: `[]`,
		},
		{
			name:    "invalid input IR: extensions does not map to an object",
			inputIR: `{"extensions":[]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := conjureplugin.AddExtensionsToIrBytes(
				[]byte(tc.inputIR),
				func(_ []byte, _, _, _ string) (map[string]any, error) {
					return nil, nil
				},
				"",
				"",
				"",
			)
			require.Error(t, err)
		})
	}
}
