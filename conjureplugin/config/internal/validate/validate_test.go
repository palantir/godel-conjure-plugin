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

package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateProjectName(t *testing.T) {
	for _, tc := range []struct {
		name        string
		projectName string
		wantError   string
	}{
		{
			name:        "valid project name",
			projectName: "my-project",
			wantError:   "",
		},
		{
			name:        "valid project name with underscores",
			projectName: "my_project",
			wantError:   "",
		},
		{
			name:        "valid project name with numbers",
			projectName: "project123",
			wantError:   "",
		},
		{
			name:        "valid project name with mixed characters",
			projectName: "my-project_v2",
			wantError:   "",
		},
		{
			name:        "invalid project name with forward slash",
			projectName: "my/project",
			wantError:   `project name "my/project" cannot contain path separators (/ or \)`,
		},
		{
			name:        "invalid project name with backslash",
			projectName: "my\\project",
			wantError:   `project name "my\\project" cannot contain path separators (/ or \)`,
		},
		{
			name:        "invalid project name with multiple forward slashes",
			projectName: "my/project/foo",
			wantError:   `project name "my/project/foo" cannot contain path separators (/ or \)`,
		},
		{
			name:        "invalid project name is dot",
			projectName: ".",
			wantError:   `project name cannot be "."`,
		},
		{
			name:        "invalid project name is double dot",
			projectName: "..",
			wantError:   `project name cannot be ".."`,
		},
		{
			name:        "valid project name starting with dot",
			projectName: ".hidden-project",
			wantError:   "",
		},
		{
			name:        "valid project name with spaces",
			projectName: "my project",
			wantError:   "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProjectName(tc.projectName)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
		})
	}
}
