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

package typescript

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateArtifactory(t *testing.T) {
	t.Run("multiple scopes", func(t *testing.T) {
		var paths []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "user", username)
			assert.Equal(t, "password", password)
			_, _ = fmt.Fprintf(w, "//auth.example/%s=credentials\n", filepath.Base(r.URL.Path))
		}))
		defer server.Close()

		auth, err := authenticateArtifactory(npmAuthRequest{
			// 2 packages under the alpha scope and 1 that should default to the palantir scope
			packageNames: []string{"@alpha/one", "unscoped", "@alpha/two"},
			registry:     server.URL,
			username:     "user",
			password:     "password",
		})
		require.NoError(t, err)

		assert.Equal(t, []string{"/auth/alpha", "/auth/palantir"}, paths)
		assert.Contains(t, auth.npmrcContent, "//auth.example/alpha=credentials")
		assert.Contains(t, auth.npmrcContent, "//auth.example/palantir=credentials")
	})

	t.Run("401", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(strings.Repeat("body should not be read", 1000)))
		}))
		defer server.Close()

		_, err := authenticateArtifactory(npmAuthRequest{
			packageNames: []string{"@alpha/one"},
			registry:     server.URL,
			username:     "user",
			password:     "password",
		})
		require.ErrorContains(t, err, "status 401")
	})
}
