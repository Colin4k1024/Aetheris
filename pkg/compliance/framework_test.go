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
	"testing"
	"time"
)

func TestStandard(t *testing.T) {
	standards := []Standard{
		StandardSOC2,
		StandardGDPR,
		StandardHIPAA,
		StandardISO27001,
	}
	for _, s := range standards {
		if s == "" {
			t.Error("Standard should not be empty")
		}
	}
}

func TestControlCategory(t *testing.T) {
	categories := []ControlCategory{
		ControlCategoryCC1,
		ControlCategoryCC2,
		ControlCategoryCC3,
		ControlCategoryCC4,
		ControlCategoryCC5,
		ControlCategoryCC6,
		ControlCategoryCC7,
		ControlCategoryCC8,
		ControlCategoryCC9,
		ControlCategoryGDPR1,
		ControlCategoryGDPR2,
		ControlCategoryGDPR3,
		ControlCategoryHIPAA1,
		ControlCategoryHIPAA2,
		ControlCategoryHIPAA3,
	}
	for _, c := range categories {
		if c == "" {
			t.Error("ControlCategory should not be empty")
		}
	}
}

func TestStatus(t *testing.T) {
	statuses := []Status{
		StatusCompliant,
		StatusNonCompliant,
		StatusNotApplicable,
		StatusPending,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("Status should not be empty")
		}
	}
}

func TestSeverity(t *testing.T) {
	severities := []Severity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
		SeverityInfo,
	}
	for _, s := range severities {
		if s == "" {
			t.Error("Severity should not be empty")
		}
	}
}

func TestControl(t *testing.T) {
	control := Control{
		ID:            "CC1.1",
		Category:      ControlCategoryCC1,
		Name:          "Test Control",
		Description:   "Test description",
		Standard:      StandardSOC2,
		Implemented:   true,
		Automated:     false,
		EvidencePaths: []string{"/path/to/evidence"},
	}
	if control.ID != "CC1.1" {
		t.Errorf("expected CC1.1, got %s", control.ID)
	}
	if control.Category != ControlCategoryCC1 {
		t.Errorf("expected CC1, got %s", control.Category)
	}
	if !control.Implemented {
		t.Error("Expected Implemented to be true")
	}
}

func TestControlStatus(t *testing.T) {
	now := time.Now()
	status := ControlStatus{
		ControlID:     "CC1.1",
		Status:        StatusCompliant,
		LastCheckTime: now,
		Findings:      []Finding{},
		Evidence:      []string{"/path"},
	}
	if status.ControlID != "CC1.1" {
		t.Errorf("expected CC1.1, got %s", status.ControlID)
	}
	if status.Status != StatusCompliant {
		t.Errorf("expected Compliant, got %s", status.Status)
	}
}

func TestFinding(t *testing.T) {
	now := time.Now()
	finding := Finding{
		Severity:  SeverityHigh,
		Message:   "Test finding",
		Resource:  "/path/to/resource",
		Timestamp: now,
	}
	if finding.Severity != SeverityHigh {
		t.Errorf("expected High, got %s", finding.Severity)
	}
	if finding.Message != "Test finding" {
		t.Errorf("expected message, got %s", finding.Message)
	}
}

func TestComplianceFramework_New(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	if framework == nil {
		t.Fatal("expected non-nil framework")
	}
	if framework.Standard != StandardSOC2 {
		t.Errorf("expected SOC2, got %s", framework.Standard)
	}
	if framework.Controls == nil {
		t.Error("expected non-nil Controls")
	}
	if framework.ControlStatuses == nil {
		t.Error("expected non-nil ControlStatuses")
	}
}

func TestComplianceFramework_AddControl(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	control := &Control{
		ID:       "CC1.1",
		Category: ControlCategoryCC1,
		Name:     "Test",
	}
	framework.AddControl(control)
	if framework.Controls["CC1.1"] == nil {
		t.Error("expected control to be added")
	}
}

func TestComplianceFramework_GetControl(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	control := &Control{
		ID:       "CC1.1",
		Category: ControlCategoryCC1,
		Name:     "Test",
	}
	framework.AddControl(control)
	retrieved := framework.GetControl("CC1.1")
	if retrieved == nil {
		t.Error("expected to retrieve control")
	}
	if retrieved.ID != "CC1.1" {
		t.Errorf("expected CC1.1, got %s", retrieved.ID)
	}
}

func TestComplianceFramework_GetControl_NotFound(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	retrieved := framework.GetControl("nonexistent")
	if retrieved != nil {
		t.Error("expected nil for nonexistent control")
	}
}

func TestComplianceFramework_UpdateControlStatus(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	control := &Control{
		ID:          "CC1.1",
		Category:    ControlCategoryCC1,
		Name:        "Test",
		Implemented: false,
	}
	framework.AddControl(control)

	status := &ControlStatus{
		ControlID: "CC1.1",
		Status:    StatusCompliant,
	}
	framework.UpdateControlStatus("CC1.1", status)

	// Control should now be implemented
	if !control.Implemented {
		t.Error("Expected control to be marked as implemented")
	}
	if control.LastCheckedAt == nil {
		t.Error("Expected LastCheckedAt to be set")
	}
}

func TestComplianceFramework_GetComplianceRate_Empty(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	rate := framework.GetComplianceRate()
	if rate != 0 {
		t.Errorf("expected 0, got %f", rate)
	}
}

func TestComplianceFramework_GetComplianceRate(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	framework.AddControl(&Control{ID: "CC1.1", Category: ControlCategoryCC1})
	framework.AddControl(&Control{ID: "CC1.2", Category: ControlCategoryCC1})

	framework.UpdateControlStatus("CC1.1", &ControlStatus{ControlID: "CC1.1", Status: StatusCompliant})
	framework.UpdateControlStatus("CC1.2", &ControlStatus{ControlID: "CC1.2", Status: StatusNonCompliant})

	rate := framework.GetComplianceRate()
	if rate != 50 {
		t.Errorf("expected 50, got %f", rate)
	}
}

func TestComplianceFramework_GetControlsByCategory(t *testing.T) {
	framework := NewFramework(StandardSOC2)
	framework.AddControl(&Control{ID: "CC1.1", Category: ControlCategoryCC1})
	framework.AddControl(&Control{ID: "CC2.1", Category: ControlCategoryCC2})
	framework.AddControl(&Control{ID: "CC1.2", Category: ControlCategoryCC1})

	controls := framework.GetControlsByCategory(ControlCategoryCC1)
	if len(controls) != 2 {
		t.Errorf("expected 2 controls, got %d", len(controls))
	}
}

func TestFrameworkFactory_CreateFramework_SOC2(t *testing.T) {
	factory := FrameworkFactory{}
	framework, err := factory.CreateFramework(StandardSOC2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if framework == nil {
		t.Fatal("expected non-nil framework")
	}
	if len(framework.Controls) == 0 {
		t.Error("expected controls to be added")
	}
}

func TestFrameworkFactory_CreateFramework_GDPR(t *testing.T) {
	factory := FrameworkFactory{}
	framework, err := factory.CreateFramework(StandardGDPR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if framework == nil {
		t.Fatal("expected non-nil framework")
	}
}

func TestFrameworkFactory_CreateFramework_HIPAA(t *testing.T) {
	factory := FrameworkFactory{}
	framework, err := factory.CreateFramework(StandardHIPAA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if framework == nil {
		t.Fatal("expected non-nil framework")
	}
}

func TestFrameworkFactory_CreateFramework_Unsupported(t *testing.T) {
	factory := FrameworkFactory{}
	_, err := factory.CreateFramework("UNSUPPORTED")
	if err == nil {
		t.Fatal("expected error for unsupported standard")
	}
}
