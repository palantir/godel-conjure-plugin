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

package tempfilecreator

import (
	"errors"
	"os"
)

// WriteBytesToTempFile writes the provided bytes to a new temporary file.
// Returns the closed file path on success, or an error if any operation fails.
// On error, attempts to remove the temp file.
func WriteBytesToTempFile(bytes []byte) (_ string, rErr error) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer func() {
		if rErr != nil {
			rErr = errors.Join(rErr, os.Remove(file.Name()))
		}
	}()
	defer func() { rErr = errors.Join(rErr, file.Close()) }()

	if _, err = file.Write(bytes); err != nil {
		return "", err
	}

	return file.Name(), nil
}

// MustWriteBytesToTempFile is a helper that panics if WriteBytesToTempFile fails.
// Returns the temp file path on success.
// Panics on failure.
func MustWriteBytesToTempFile(bytes []byte) string {
	if path, err := WriteBytesToTempFile(bytes); err != nil {
		panic(err)
	} else {
		return path
	}
}
