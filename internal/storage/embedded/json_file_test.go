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

package embedded

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSON_FileNotFound(t *testing.T) {
	type Data struct {
		Value string `json:"value"`
	}
	var data Data
	err := LoadJSON("/nonexistent/path/file.json", &data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should return nil for missing file
}

func TestLoadJSON_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")

	// Create empty file
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	type Data struct {
		Value string `json:"value"`
	}
	var data Data
	err := LoadJSON(path, &data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadJSON_ValidData(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.json")

	// Write JSON data
	content := `{"value":"test"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	type Data struct {
		Value string `json:"value"`
	}
	var data Data
	err := LoadJSON(path, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Value != "test" {
		t.Errorf("expected test, got %s", data.Value)
	}
}

func TestSaveJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "save.json")

	type Data struct {
		Value string `json:"value"`
	}
	data := Data{Value: "test"}

	err := SaveJSON(path, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty file")
	}
}

func TestSaveJSON_NestedDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "dir", "save.json")

	type Data struct {
		Value string `json:"value"`
	}
	data := Data{Value: "nested"}

	err := SaveJSON(path, &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(content) == "" {
		t.Error("expected non-empty file")
	}
}
