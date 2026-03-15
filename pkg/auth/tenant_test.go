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

package auth

import (
	"testing"
	"time"
)

func TestTenantStatusConstants(t *testing.T) {
	tests := []struct {
		status   TenantStatus
		expected string
	}{
		{TenantStatusActive, "active"},
		{TenantStatusSuspended, "suspended"},
		{TenantStatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestTenant(t *testing.T) {
	now := time.Now()
	tenant := Tenant{
		ID:        "tenant-1",
		Name:      "Test Tenant",
		Status:    TenantStatusActive,
		Quota:     DefaultTenantQuota(),
		Metadata:  map[string]string{"key": "value"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if tenant.ID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", tenant.ID)
	}
	if tenant.Name != "Test Tenant" {
		t.Errorf("expected Test Tenant, got %s", tenant.Name)
	}
	if tenant.Status != TenantStatusActive {
		t.Errorf("expected active, got %s", tenant.Status)
	}
	if tenant.Quota.MaxJobs != 0 {
		t.Errorf("expected 0, got %d", tenant.Quota.MaxJobs)
	}
	if tenant.Quota.MaxExports != 100 {
		t.Errorf("expected 100, got %d", tenant.Quota.MaxExports)
	}
	if tenant.Quota.MaxAgents != 100 {
		t.Errorf("expected 100, got %d", tenant.Quota.MaxAgents)
	}
}

func TestTenantQuota(t *testing.T) {
	quota := TenantQuota{
		MaxJobs:    100,
		MaxStorage: 1024 * 1024 * 1024,
		MaxExports: 50,
		MaxAgents:  50,
	}

	if quota.MaxJobs != 100 {
		t.Errorf("expected 100, got %d", quota.MaxJobs)
	}
	if quota.MaxStorage != 1024*1024*1024 {
		t.Errorf("expected 1073741824, got %d", quota.MaxStorage)
	}
	if quota.MaxExports != 50 {
		t.Errorf("expected 50, got %d", quota.MaxExports)
	}
	if quota.MaxAgents != 50 {
		t.Errorf("expected 50, got %d", quota.MaxAgents)
	}
}

func TestDefaultTenantQuota(t *testing.T) {
	quota := DefaultTenantQuota()

	if quota.MaxJobs != 0 {
		t.Errorf("expected MaxJobs=0, got %d", quota.MaxJobs)
	}
	if quota.MaxStorage != 0 {
		t.Errorf("expected MaxStorage=0, got %d", quota.MaxStorage)
	}
	if quota.MaxExports != 100 {
		t.Errorf("expected MaxExports=100, got %d", quota.MaxExports)
	}
	if quota.MaxAgents != 100 {
		t.Errorf("expected MaxAgents=100, got %d", quota.MaxAgents)
	}
}
