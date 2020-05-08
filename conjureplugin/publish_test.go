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
	"github.com/palantir/godel-conjure-plugin/v5/conjureplugin"
	"github.com/palantir/godel-conjure-plugin/v5/conjureplugin/config"
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
	params, err := cfg.ToParams()
	require.NoError(t, err, "failed to parse config set")

	outputBuf := &bytes.Buffer{}
	err = conjureplugin.Publish(params, tmpDir, map[distgo.PublisherFlagName]interface{}{
		publisher.ConnectionInfoURLFlag.Name:     "http://artifactory.domain.com",
		publisher.GroupIDFlag.Name:               "com.palantir.foo",
		artifactory.PublisherRepositoryFlag.Name: "repo",
	}, true, outputBuf)
	require.NoError(t, err, "failed to publish Conjure")

	lines := strings.Split(outputBuf.String(), "\n")
	assert.Equal(t, 3, len(lines), "Expected output to have 3 lines:\n%s", outputBuf.String())

	wantRegexp := regexp.QuoteMeta("[DRY RUN]") + " Uploading .*?" + regexp.QuoteMeta(".conjure.json") + " to " + regexp.QuoteMeta("http://artifactory.domain.com/artifactory/repo/com/palantir/foo/project-1/") + ".*?" + regexp.QuoteMeta("/project-1-") + ".*?" + regexp.QuoteMeta(".conjure.json")
	assert.Regexp(t, wantRegexp, lines[0])

	wantRegexp = regexp.QuoteMeta("[DRY RUN]") + " Uploading to " + regexp.QuoteMeta("http://artifactory.domain.com/artifactory/repo/com/palantir/foo/") + ".*?" + regexp.QuoteMeta(".pom")
	assert.Regexp(t, wantRegexp, lines[1])
}
