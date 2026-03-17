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
	"testing"
)

func TestNewContextManager(t *testing.T) {
	cm := NewContextManager()
	if cm == nil {
		t.Fatal("expected non-nil ContextManager")
	}
	if cm.runners == nil {
		t.Error("expected non-nil runners map")
	}
}

func TestContextManager_RegisterAndGetRunner(t *testing.T) {
	cm := NewContextManager()

	// GetRunner should fail for non-existent runner
	_, err := cm.GetRunner("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent runner")
	}
}

func TestContextManager_ExecuteTool_NotFound(t *testing.T) {
	cm := NewContextManager()

	// ExecuteTool should fail for non-existent runner
	_, err := cm.ExecuteTool(nil, "nonexistent", "tool", "input")
	if err == nil {
		t.Error("expected error for non-existent runner")
	}
}
