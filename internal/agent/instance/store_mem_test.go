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

package instance

import (
	"context"
	"testing"
)

func TestStoreMem_GetCreateUpdate(t *testing.T) {
	ctx := context.Background()
	s := NewStoreMem()
	// Get missing returns nil
	got, err := s.Get(ctx, "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("Get missing: got %v", got)
	}
	// Create
	inst := &AgentInstance{ID: "agent-1", Name: "Test", Status: StatusIdle}
	if err := s.Create(ctx, inst); err != nil {
		t.Fatal(err)
	}
	got, err = s.Get(ctx, "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "agent-1" || got.Name != "Test" || got.Status != StatusIdle {
		t.Fatalf("Get after Create: got %+v", got)
	}
	// UpdateStatus
	if err := s.UpdateStatus(ctx, "agent-1", StatusRunning); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(ctx, "agent-1")
	if got.Status != StatusRunning {
		t.Fatalf("after UpdateStatus: got status %q", got.Status)
	}
	// ListByTenant
	list, err := s.ListByTenant(ctx, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "agent-1" {
		t.Fatalf("ListByTenant: got %+v", list)
	}
}

func TestStoreMem_CreateNilInstance(t *testing.T) {
	ctx := context.Background()
	s := NewStoreMem()

	err := s.Create(ctx, nil)
	if err != nil {
		t.Errorf("expected nil error for nil instance, got %v", err)
	}

	err = s.Create(ctx, &AgentInstance{})
	if err != nil {
		t.Errorf("expected nil error for empty ID, got %v", err)
	}
}

func TestStoreMem_Update(t *testing.T) {
	ctx := context.Background()
	s := NewStoreMem()

	// Create first
	inst := &AgentInstance{ID: "agent-1", Name: "Test", Status: StatusIdle, TenantID: "tenant-1"}
	s.Create(ctx, inst)

	// Update
	err := s.Update(ctx, &AgentInstance{ID: "agent-1", Name: "Updated", Status: StatusRunning})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(ctx, "agent-1")
	if got.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %s", got.Name)
	}
}

func TestStoreMem_UpdateNonExistent(t *testing.T) {
	ctx := context.Background()
	s := NewStoreMem()

	err := s.Update(ctx, &AgentInstance{ID: "non-existent", Name: "Test"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoreMem_UpdateCurrentJob(t *testing.T) {
	ctx := context.Background()
	s := NewStoreMem()

	s.Create(ctx, &AgentInstance{ID: "agent-1", Status: StatusIdle})

	err := s.UpdateCurrentJob(ctx, "agent-1", "job-123")
	if err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(ctx, "agent-1")
	if got.CurrentJobID != "job-123" {
		t.Errorf("expected job-123, got %s", got.CurrentJobID)
	}
}

func TestStoreMem_ListByTenant_Filter(t *testing.T) {
	ctx := context.Background()
	s := NewStoreMem()

	s.Create(ctx, &AgentInstance{ID: "agent-1", TenantID: "tenant-1"})
	s.Create(ctx, &AgentInstance{ID: "agent-2", TenantID: "tenant-2"})
	s.Create(ctx, &AgentInstance{ID: "agent-3", TenantID: "tenant-1"})

	list, _ := s.ListByTenant(ctx, "tenant-1", 10)
	if len(list) != 2 {
		t.Errorf("expected 2, got %d", len(list))
	}

	// Test limit
	list, _ = s.ListByTenant(ctx, "", 1)
	if len(list) != 1 {
		t.Errorf("expected 1, got %d", len(list))
	}
}
