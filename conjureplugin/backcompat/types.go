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

package backcompat

// Input represents the JSON input sent to the backcompat asset.
type Input struct {
	Type                   string                `json:"type"`
	CheckBackCompat        *CheckBackCompatInput `json:"checkBackCompat,omitempty"`
	AcceptBackCompatBreaks *AcceptBreaksInput    `json:"acceptBackCompatBreaks,omitempty"`
}

// CheckBackCompatInput contains the inputs for checking backcompat.
type CheckBackCompatInput struct {
	CurrentIR       string         `json:"currentIR"`
	Project         string         `json:"project"`
	GroupID         string         `json:"groupId"`
	ProjectConfig   map[string]any `json:"projectConfig"`
	GodelProjectDir string         `json:"godelProjectDir"`
}

// AcceptBreaksInput contains the inputs for accepting backcompat breaks.
type AcceptBreaksInput struct {
	CurrentIR       string         `json:"currentIR"`
	Project         string         `json:"project"`
	GroupID         string         `json:"groupId"`
	ProjectConfig   map[string]any `json:"projectConfig"`
	GodelProjectDir string         `json:"godelProjectDir"`
}
