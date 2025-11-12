// Copyright (c) 2024 Six-Thirty Labs, Inc.
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

package opa

import "testing"

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name          string
		inputPath     string
		expectedDot   string
		expectedSlash string
	}{
		{
			name:          "slash format",
			inputPath:     "rbac/allow",
			expectedDot:   "rbac.allow",
			expectedSlash: "rbac/allow",
		},
		{
			name:          "dot format",
			inputPath:     "rbac.allow",
			expectedDot:   "rbac.allow",
			expectedSlash: "rbac/allow",
		},
		{
			name:          "nested slash format",
			inputPath:     "authz/rules/allow",
			expectedDot:   "authz.rules.allow",
			expectedSlash: "authz/rules/allow",
		},
		{
			name:          "nested dot format",
			inputPath:     "authz.rules.allow",
			expectedDot:   "authz.rules.allow",
			expectedSlash: "authz/rules/allow",
		},
		{
			name:          "single segment",
			inputPath:     "allow",
			expectedDot:   "allow",
			expectedSlash: "allow",
		},
		{
			name:          "empty path",
			inputPath:     "",
			expectedDot:   "",
			expectedSlash: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dotPath, slashPath := normalizePath(tt.inputPath)

			if dotPath != tt.expectedDot {
				t.Errorf("dotPath: got %q, want %q", dotPath, tt.expectedDot)
			}

			if slashPath != tt.expectedSlash {
				t.Errorf("slashPath: got %q, want %q", slashPath, tt.expectedSlash)
			}
		})
	}
}
