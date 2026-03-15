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

package compliance

import (
	"context"
	"testing"
	"time"
)

func TestRegionCode(t *testing.T) {
	tests := []struct {
		code   RegionCode
		expect string
	}{
		{RegionCode("US"), "US"},
		{RegionCode("EU"), "EU"},
		{RegionCode("CN"), "CN"},
	}

	for _, tt := range tests {
		if string(tt.code) != tt.expect {
			t.Errorf("expected %s, got %s", tt.expect, tt.code)
		}
	}
}

func TestDataCategory(t *testing.T) {
	tests := []struct {
		cat    DataCategory
		expect string
	}{
		{DataCategoryPersonal, "personal"},
		{DataCategorySensitive, "sensitive"},
		{DataCategoryFinancial, "financial"},
		{DataCategoryHealth, "health"},
		{DataCategoryGeneral, "general"},
	}

	for _, tt := range tests {
		if string(tt.cat) != tt.expect {
			t.Errorf("expected %s, got %s", tt.expect, tt.cat)
		}
	}
}

func TestResidencyPolicy(t *testing.T) {
	policy := &ResidencyPolicy{
		AllowedRegions:     []RegionCode{"US", "EU"},
		BlockedRegions:     []RegionCode{"CN"},
		TransferEncryption: true,
		DefaultRegion:      "US",
		RetentionByCategory: map[DataCategory]DataRetention{
			DataCategoryPersonal: {
				RetentionDays:      365,
				ArchiveDays:        90,
				DeleteAfterArchive: true,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if len(policy.AllowedRegions) != 2 {
		t.Errorf("expected 2 allowed regions, got %d", len(policy.AllowedRegions))
	}
	if len(policy.BlockedRegions) != 1 {
		t.Errorf("expected 1 blocked region, got %d", len(policy.BlockedRegions))
	}
	if !policy.TransferEncryption {
		t.Error("expected TransferEncryption=true")
	}
	if policy.DefaultRegion != "US" {
		t.Errorf("expected US, got %s", policy.DefaultRegion)
	}
}

func TestDataRetention(t *testing.T) {
	retention := DataRetention{
		RetentionDays:      365,
		ArchiveDays:        90,
		DeleteAfterArchive: true,
	}

	if retention.RetentionDays != 365 {
		t.Errorf("expected 365, got %d", retention.RetentionDays)
	}
	if retention.ArchiveDays != 90 {
		t.Errorf("expected 90, got %d", retention.ArchiveDays)
	}
	if !retention.DeleteAfterArchive {
		t.Error("expected DeleteAfterArchive=true")
	}
}

func TestIPRegionLookup(t *testing.T) {
	lookup := &IPRegionLookup{}

	tests := []struct {
		identifier string
		expected   RegionCode
	}{
		{"10.0.0.1", "XX"},
		{"172.16.0.1", "XX"},
		{"192.168.0.1", "XX"},
		{"8.8.8.8", "US"},
		{"1.2.3.4", "US"},
	}

	for _, tt := range tests {
		result, err := lookup.GetRegion(context.Background(), tt.identifier)
		if err != nil {
			t.Errorf("unexpected error for %s: %v", tt.identifier, err)
		}
		if result != tt.expected {
			t.Errorf("expected %s for %s, got %s", tt.expected, tt.identifier, result)
		}
	}
}

func TestNewDataResidencyController(t *testing.T) {
	defaultPolicy := &ResidencyPolicy{
		DefaultRegion:  "US",
		AllowedRegions: []RegionCode{"US", "EU"},
	}

	controller := NewDataResidencyController(defaultPolicy)
	if controller == nil {
		t.Fatal("expected non-nil controller")
	}
	if controller.defaultPolicy == nil {
		t.Error("expected non-nil defaultPolicy")
	}
	if controller.regionLookup == nil {
		t.Error("expected non-nil regionLookup")
	}
}

func TestDataResidencyController_SetPolicy(t *testing.T) {
	controller := NewDataResidencyController(&ResidencyPolicy{
		DefaultRegion: "US",
	})

	policy := &ResidencyPolicy{
		DefaultRegion:  "EU",
		AllowedRegions: []RegionCode{"EU"},
	}

	controller.SetPolicy("tenant-1", policy)

	controller.mu.RLock()
	p := controller.policies["tenant-1"]
	controller.mu.RUnlock()

	if p == nil {
		t.Fatal("expected policy to be set")
	}
	if p.DefaultRegion != "EU" {
		t.Errorf("expected EU, got %s", p.DefaultRegion)
	}
}

func TestDataResidencyController_GetPolicy(t *testing.T) {
	controller := NewDataResidencyController(&ResidencyPolicy{
		DefaultRegion: "US",
	})

	// Get non-existent policy should return default
	policy := controller.GetPolicy("unknown-tenant")
	if policy == nil {
		t.Error("expected default policy")
	}
	if policy.DefaultRegion != "US" {
		t.Errorf("expected US, got %s", policy.DefaultRegion)
	}

	// Set and get policy
	customPolicy := &ResidencyPolicy{
		DefaultRegion: "EU",
	}
	controller.SetPolicy("tenant-1", customPolicy)

	policy = controller.GetPolicy("tenant-1")
	if policy.DefaultRegion != "EU" {
		t.Errorf("expected EU, got %s", policy.DefaultRegion)
	}
}

func TestDataResidencyController_CheckStorageCompliance(t *testing.T) {
	controller := NewDataResidencyController(&ResidencyPolicy{
		DefaultRegion:      "US",
		AllowedRegions:     []RegionCode{"US", "EU"},
		TransferEncryption: true,
	})

	// Valid storage
	err := controller.CheckStorageCompliance(context.Background(), "tenant-1", "US", DataCategoryPersonal)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Blocked region
	err = controller.CheckStorageCompliance(context.Background(), "tenant-1", "CN", DataCategoryPersonal)
	if err == nil {
		t.Error("expected error for blocked region")
	}
}

func TestDataResidencyController_CheckTransferCompliance(t *testing.T) {
	controller := NewDataResidencyController(&ResidencyPolicy{
		DefaultRegion:      "US",
		AllowedRegions:     []RegionCode{"US", "EU"},
		TransferEncryption: true,
	})

	// Valid transfer between allowed regions
	err := controller.CheckTransferCompliance(context.Background(), "tenant-1", "US", "EU")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Transfer to blocked region
	err = controller.CheckTransferCompliance(context.Background(), "tenant-1", "US", "CN")
	if err == nil {
		t.Error("expected error for blocked region")
	}
}

func TestDataResidencyController_GetDataRetention(t *testing.T) {
	controller := NewDataResidencyController(nil)

	// Set up retention policy
	policy := &ResidencyPolicy{
		RetentionByCategory: map[DataCategory]DataRetention{
			DataCategoryPersonal: {
				RetentionDays: 365,
			},
		},
	}
	controller.SetPolicy("tenant-1", policy)

	retention := controller.GetDataRetention("tenant-1", DataCategoryPersonal)
	if retention.RetentionDays != 365 {
		t.Errorf("expected 365, got %d", retention.RetentionDays)
	}
}

func TestDataResidencyController_GetDefaultRegion(t *testing.T) {
	controller := NewDataResidencyController(nil)

	// Set up policy with default region
	policy := &ResidencyPolicy{
		DefaultRegion: "US",
	}
	controller.SetPolicy("tenant-1", policy)

	region := controller.GetDefaultRegion("tenant-1")
	if region != "US" {
		t.Errorf("expected US, got %s", region)
	}
}

func TestRegionCode_GetRegionName(t *testing.T) {
	tests := []struct {
		code     RegionCode
		expected string
	}{
		{"US", "United States"},
		{"EU", "European Union"},
		{"CN", "China"},
		{"XX", "XX"}, // XX is not in the map, returns itself
	}

	for _, tt := range tests {
		result := tt.code.GetRegionName()
		if result != tt.expected {
			t.Errorf("expected %s for %s, got %s", tt.expected, tt.code, result)
		}
	}
}

func TestRegionCode_IsEURegion(t *testing.T) {
	tests := []struct {
		code     RegionCode
		expected bool
	}{
		{"US", false},
		{"CN", false},
		{"DE", true},
		{"FR", true},
		{"UK", true}, // Note: uses UK, not GB
		{"EU", true},
	}

	for _, tt := range tests {
		result := tt.code.IsEURegion()
		if result != tt.expected {
			t.Errorf("expected %v for %s, got %v", tt.expected, tt.code, result)
		}
	}
}
