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

package eino

import (
	"os"
	"testing"
)

func TestContextMode_String(t *testing.T) {
	tests := []struct {
		mode     ContextMode
		expected string
	}{
		{ContextModeInline, "inline"},
		{ContextModeFork, "fork"},
		{ContextModeIsolate, "isolate"},
	}
	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.mode)
		}
	}
}

func TestFrontMatter(t *testing.T) {
	fm := FrontMatter{
		Name:        "test-skill",
		Description: "A test skill",
		Context:     ContextModeInline,
		Agent:       "react",
		Model:       "gpt-4",
	}
	if fm.Name != "test-skill" {
		t.Errorf("expected test-skill, got %s", fm.Name)
	}
	if fm.Context != ContextModeInline {
		t.Errorf("expected inline, got %s", fm.Context)
	}
}

func TestSkill(t *testing.T) {
	skill := Skill{
		FrontMatter: FrontMatter{
			Name:        "test",
			Description: "desc",
		},
		Content:       "skill content",
		BaseDirectory: "/path/to/skill",
	}
	if skill.Name != "test" {
		t.Errorf("expected test, got %s", skill.Name)
	}
	if skill.Content != "skill content" {
		t.Errorf("expected skill content, got %s", skill.Content)
	}
}

type mockFileSystemBackend struct {
	files map[string][]byte
	dirs  map[string][]os.DirEntry
}

func (m *mockFileSystemBackend) ReadFile(ctx interface{}, path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, &mockError{"file not found"}
}

func (m *mockFileSystemBackend) ReadDir(ctx interface{}, path string) ([]os.DirEntry, error) {
	if entries, ok := m.dirs[path]; ok {
		return entries, nil
	}
	return nil, nil
}

func (m *mockFileSystemBackend) IsDir(ctx interface{}, path string) (bool, error) {
	return false, nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
