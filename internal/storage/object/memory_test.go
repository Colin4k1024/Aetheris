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

package object

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestMemoryStore_Put_Get_Delete(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	data := bytes.NewReader([]byte("hello"))
	if err := s.Put(ctx, "p1", data, 5, nil); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, err := s.Get(ctx, "p1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if string(b) != "hello" {
		t.Errorf("Get: got %q", string(b))
	}
	if err := s.Delete(ctx, "p1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "p1"); err == nil {
		t.Error("Get after Delete should error")
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_, err := s.Get(ctx, "missing")
	if err == nil {
		t.Error("Get missing should error")
	}
}

func TestMemoryStore_List(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	// Put some objects
	_ = s.Put(ctx, "dir1/file1.txt", bytes.NewReader([]byte("a")), 0, nil)
	_ = s.Put(ctx, "dir1/file2.txt", bytes.NewReader([]byte("b")), 0, nil)
	_ = s.Put(ctx, "dir2/file3.txt", bytes.NewReader([]byte("c")), 0, nil)

	// List
	objs, err := s.List(ctx, "dir1/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(objs) != 2 {
		t.Errorf("expected 2 objects, got %d", len(objs))
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	_ = s.Put(ctx, "exists.txt", bytes.NewReader([]byte("a")), 0, nil)

	exists, err := s.Exists(ctx, "exists.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Error("expected exists=true")
	}

	exists, err = s.Exists(ctx, "missing.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("expected exists=false")
	}
}

func TestMemoryStore_GetMetadata(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	// GetMetadata for missing object should error
	_, err := s.GetMetadata(ctx, "missing.txt")
	if err == nil {
		t.Error("expected error for missing object")
	}
}

func TestMemoryStore_Close(t *testing.T) {
	s := NewMemoryStore()
	err := s.Close()
	if err != nil {
		t.Errorf("Close: %v", err)
	}
}
