// Copyright 2026 fanjia1024

package compliance

import (
	"testing"
)

// TestGetTemplate 测试获取模板
func TestGetTemplate(t *testing.T) {
	template := GetTemplate("GDPR")
	if template == nil {
		t.Fatal("GDPR template should exist")
	}

	if template.Standard != "GDPR" {
		t.Errorf("expected standard GDPR, got %s", template.Standard)
	}

	if template.RetentionDays != 90 {
		t.Errorf("expected 90 days retention, got %d", template.RetentionDays)
	}
	if template.Version == "" {
		t.Error("expected template version")
	}
	if template.ExportFormat != "json" {
		t.Errorf("expected json export format, got %s", template.ExportFormat)
	}
}

// TestListTemplates 测试列出所有模板
func TestListTemplates(t *testing.T) {
	templates := ListTemplates()

	if len(templates) < 3 {
		t.Errorf("expected at least 3 templates, got %d", len(templates))
	}

	hasGDPR := false
	for _, tmpl := range templates {
		if tmpl.Name == "GDPR" {
			hasGDPR = true
		}
	}

	if !hasGDPR {
		t.Error("should have GDPR template")
	}
}
