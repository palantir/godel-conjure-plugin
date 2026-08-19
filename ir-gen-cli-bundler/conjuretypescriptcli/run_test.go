// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package conjuretypescriptcli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/godel-conjure-plugin/v7/ir-gen-cli-bundler/conjureircli"
	"github.com/palantir/godel-conjure-plugin/v7/ir-gen-cli-bundler/conjuretypescriptcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	const yml = `
types:
  definitions:
    default-package: com.palantir.conjure
    objects:
      BooleanExample: { fields: { value: boolean } }
`
	irBytes, err := conjureircli.YAMLtoIR([]byte(yml))
	require.NoError(t, err)

	tmpDir := t.TempDir()
	irPath := filepath.Join(tmpDir, "ir.json")
	require.NoError(t, os.WriteFile(irPath, irBytes, 0644))

	outDir := filepath.Join(tmpDir, "out")
	require.NoError(t, os.Mkdir(outDir, 0755))

	require.NoError(t, conjuretypescriptcli.Generate(irPath, outDir, conjuretypescriptcli.GenerateOptions{
		PackageName:              "@palantir/test-api",
		PackageVersion:           "1.2.3",
		GenerateThrowingServices: true,
	}))

	pkgJSONBytes, err := os.ReadFile(filepath.Join(outDir, "package.json"))
	require.NoError(t, err, "package.json should be generated")
	var pkgJSON struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(pkgJSONBytes, &pkgJSON))
	assert.Equal(t, "@palantir/test-api", pkgJSON.Name)
	assert.Equal(t, "1.2.3", pkgJSON.Version)

	assert.FileExists(t, filepath.Join(outDir, "tsconfig.json"))
	assert.FileExists(t, filepath.Join(outDir, ".npmignore"))
	assert.FileExists(t, filepath.Join(outDir, "index.ts"))
}
