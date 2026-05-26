package http

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestComplianceReport_BindsSignedEvidenceAndExposesUnsupportedControls(t *testing.T) {
	h := NewHandler(nil, nil)
	s := server.Default(server.WithHostPorts(":0"))
	s.POST("/api/compliance/report", h.ComplianceReport)

	body := []byte(`{
		"tenant_id":"tenant-1",
		"standard":"GDPR",
		"evidence_package_id":"evidence-job-1.zip",
		"evidence_verification":{
			"job_id":"job-1",
			"root_hash":"root-hash",
			"verified":true,
			"signed":true,
			"signature_valid":true,
			"signer_key_id":"release-key"
		}
	}`)
	w := ut.PerformRequest(s.Engine, "POST", "/api/compliance/report", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 200 {
		t.Fatalf("status = %d, want 200; body=%s", got, w.Result().Body())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Result().Body(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["template_version"] == "" {
		t.Fatalf("template_version should be present: %s", w.Result().Body())
	}
	if resp["evidence_package_id"] != "evidence-job-1.zip" {
		t.Fatalf("evidence_package_id = %v", resp["evidence_package_id"])
	}
	if resp["compliance_notice"] == "" {
		t.Fatalf("compliance_notice should be present")
	}
	unsupported, ok := resp["unsupported_controls"].([]interface{})
	if !ok || len(unsupported) == 0 {
		t.Fatalf("unsupported_controls should be non-empty: %s", w.Result().Body())
	}
	verification, ok := resp["evidence_verification"].(map[string]interface{})
	if !ok {
		t.Fatalf("evidence_verification should be an object")
	}
	if verification["signature_valid"] != true {
		t.Fatalf("signature_valid should be true: %v", verification)
	}
}

func TestComplianceReport_RequiresSignedEvidenceVerification(t *testing.T) {
	h := NewHandler(nil, nil)
	s := server.Default(server.WithHostPorts(":0"))
	s.POST("/api/compliance/report", h.ComplianceReport)

	body := []byte(`{"tenant_id":"tenant-1","standard":"GDPR"}`)
	w := ut.PerformRequest(s.Engine, "POST", "/api/compliance/report", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 400 {
		t.Fatalf("status = %d, want 400", got)
	}
	if !bytes.Contains(w.Result().Body(), []byte("evidence_package_id is required")) {
		t.Fatalf("expected evidence package validation error: %s", w.Result().Body())
	}
}

func TestComplianceReport_RejectsMismatchedEvidencePackageID(t *testing.T) {
	h := NewHandler(nil, nil)
	s := server.Default(server.WithHostPorts(":0"))
	s.POST("/api/compliance/report", h.ComplianceReport)

	body := []byte(`{
		"tenant_id":"tenant-1",
		"standard":"GDPR",
		"evidence_package_id":"evidence-job-1.zip",
		"evidence_verification":{
			"package_id":"different-evidence.zip",
			"root_hash":"root-hash",
			"verified":true,
			"signed":true,
			"signature_valid":true
		}
	}`)
	w := ut.PerformRequest(s.Engine, "POST", "/api/compliance/report", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 400 {
		t.Fatalf("status = %d, want 400; body=%s", got, w.Result().Body())
	}
	if !bytes.Contains(w.Result().Body(), []byte("evidence_verification.package_id must match evidence_package_id")) {
		t.Fatalf("expected package id mismatch error: %s", w.Result().Body())
	}
}
