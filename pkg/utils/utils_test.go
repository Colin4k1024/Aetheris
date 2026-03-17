// Copyright 2026 fanjia1024
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

package utils

import "testing"

func TestCoalesceString(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "first non-empty string",
			args:     []string{"hello", "world", "foo"},
			expected: "hello",
		},
		{
			name:     "second non-empty string",
			args:     []string{"", "world", "foo"},
			expected: "world",
		},
		{
			name:     "last non-empty string",
			args:     []string{"", "", "foo"},
			expected: "foo",
		},
		{
			name:     "all empty strings",
			args:     []string{"", "", ""},
			expected: "",
		},
		{
			name:     "single empty string",
			args:     []string{""},
			expected: "",
		},
		{
			name:     "single non-empty string",
			args:     []string{"single"},
			expected: "single",
		},
		{
			name:     "no arguments",
			args:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CoalesceString(tt.args...)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDefaultInt(t *testing.T) {
	tests := []struct {
		name       string
		v          int
		defaultVal int
		expected   int
	}{
		{
			name:       "zero value returns default",
			v:          0,
			defaultVal: 10,
			expected:   10,
		},
		{
			name:       "non-zero value returns itself",
			v:          42,
			defaultVal: 10,
			expected:   42,
		},
		{
			name:       "negative value returns itself",
			v:          -5,
			defaultVal: 10,
			expected:   -5,
		},
		{
			name:       "default zero with zero value",
			v:          0,
			defaultVal: 0,
			expected:   0,
		},
		{
			name:       "default zero with non-zero value",
			v:          5,
			defaultVal: 0,
			expected:   5,
		},
		{
			name:       "large values",
			v:          1000000,
			defaultVal: 1,
			expected:   1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultInt(tt.v, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
