package compliance

import (
	"context"
	"testing"
	"time"
)

type fakeMetricsSource struct {
	metrics Metrics
	err     error
}

func (f *fakeMetricsSource) GetMetrics(ctx context.Context, tenantID string, timeRange TimeRange) (Metrics, error) {
	if f.err != nil {
		return Metrics{}, f.err
	}
	return f.metrics, nil
}

func TestGenerateReport_DefaultControlsExposeUnsupportedScope(t *testing.T) {
	r := NewReporter(TemplateGDPR)
	report, err := r.GenerateReport(context.Background(), "tenant_1", TimeRange{})
	if err != nil {
		t.Fatalf("generate report failed: %v", err)
	}
	if report.TemplateVersion == "" {
		t.Fatalf("template version should be set")
	}
	if report.ComplianceRate >= 100 {
		t.Fatalf("default compliance rate = %v, want below 100 because unsupported controls must be explicit", report.ComplianceRate)
	}
	if len(report.UnsupportedControls) == 0 {
		t.Fatalf("unsupported controls should be explicit")
	}
	if report.Summary[string(StatusUnsupported)] == 0 {
		t.Fatalf("summary should count unsupported controls")
	}
}

func TestGenerateReport_WithMetrics(t *testing.T) {
	r := NewReporter(TemplateGDPR).WithMetricsSource(&fakeMetricsSource{
		metrics: Metrics{
			TotalChecks:      100,
			Violations:       5,
			MissingAuditLogs: 3,
			UnredactedPII:    2,
		},
	})
	report, err := r.GenerateReport(context.Background(), "tenant_1", TimeRange{})
	if err != nil {
		t.Fatalf("generate report failed: %v", err)
	}
	if report.ComplianceRate >= 95 || report.ComplianceRate <= 80 {
		t.Fatalf("unexpected compliance rate: %v", report.ComplianceRate)
	}
}

func TestGenerateReport_WithSignedEvidenceBinding(t *testing.T) {
	verifiedAt := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	r := NewReporter(TemplateHIPAA).WithEvidenceVerification(EvidenceVerification{
		PackageID:      "evidence-job-1.zip",
		JobID:          "job-1",
		RootHash:       "root-hash",
		Verified:       true,
		Signed:         true,
		SignatureValid: true,
		SignerKeyID:    "release-key",
		VerifiedAt:     verifiedAt,
	})

	report, err := r.GenerateReport(context.Background(), "tenant_1", TimeRange{})
	if err != nil {
		t.Fatalf("generate report failed: %v", err)
	}
	if report.EvidencePackageID != "evidence-job-1.zip" {
		t.Fatalf("evidence package id = %q", report.EvidencePackageID)
	}
	if !report.EvidenceVerification.SignatureValid {
		t.Fatalf("signature validity should be bound to report")
	}
	if report.ComplianceNotice == "" {
		t.Fatalf("compliance notice should be present")
	}
}

func TestGenerateReport_InvalidTimeRange(t *testing.T) {
	r := NewReporter(TemplateGDPR)
	_, err := r.GenerateReport(context.Background(), "tenant_1", TimeRange{
		Start: time.Now(),
		End:   time.Now().Add(-time.Hour),
	})
	if err == nil {
		t.Fatal("expected invalid time range error")
	}
}

func TestGenerateReport_ReturnsUnsupportedStandardError(t *testing.T) {
	r := NewReporter(ComplianceTemplate{
		Name:     "custom",
		Standard: "UNSUPPORTED",
		Version:  "1.0",
	})
	_, err := r.GenerateReport(context.Background(), "tenant_1", TimeRange{})
	if err == nil {
		t.Fatal("expected error for unsupported standard")
	}
}
