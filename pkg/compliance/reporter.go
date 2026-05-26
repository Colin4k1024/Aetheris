// Copyright 2026 fanjia1024
// Compliance report generator (3.0-M4)

package compliance

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Reporter 合规报告生成器
type Reporter struct {
	template             ComplianceTemplate
	source               MetricsSource
	evidenceVerification EvidenceVerification
}

// MetricsSource 合规指标来源。
type MetricsSource interface {
	GetMetrics(ctx context.Context, tenantID string, timeRange TimeRange) (Metrics, error)
}

// Metrics 合规统计指标。
type Metrics struct {
	TotalChecks      int
	Violations       int
	MissingAuditLogs int
	UnredactedPII    int
}

// NewReporter 创建报告生成器
func NewReporter(template ComplianceTemplate) *Reporter {
	return &Reporter{template: template}
}

// WithMetricsSource 设置指标来源。
func (r *Reporter) WithMetricsSource(source MetricsSource) *Reporter {
	r.source = source
	return r
}

// WithEvidenceVerification binds a report to a previously verified signed evidence package.
func (r *Reporter) WithEvidenceVerification(verification EvidenceVerification) *Reporter {
	r.evidenceVerification = verification
	return r
}

// GenerateReport 生成合规报告
func (r *Reporter) GenerateReport(ctx context.Context, tenantID string, timeRange TimeRange) (*ComplianceReport, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if !timeRange.Start.IsZero() && !timeRange.End.IsZero() && timeRange.Start.After(timeRange.End) {
		return nil, fmt.Errorf("invalid time range: start is after end")
	}

	report, err := r.generateControlReport(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if report == nil {
		report = &ComplianceReport{
			TenantID: tenantID,
			Standard: r.template.Standard,
			Summary:  map[string]int{},
		}
	}

	report.TenantID = tenantID
	report.Standard = r.template.Standard
	report.TemplateVersion = r.template.Version
	report.TemplateName = r.template.Name
	report.TimeRange = timeRange
	report.GeneratedAt = time.Now().UTC()
	report.EvidencePackageID = r.evidenceVerification.PackageID
	report.EvidenceVerification = r.evidenceVerification
	report.ComplianceNotice = "Aetheris reports summarize runtime evidence only; they are not legal compliance certifications."
	if report.Summary == nil {
		report.Summary = map[string]int{}
	}

	if r.source != nil {
		metrics, err := r.source.GetMetrics(ctx, tenantID, timeRange)
		if err != nil {
			return nil, err
		}
		report.ComplianceRate = calculateComplianceRate(metrics)
	}

	return report, nil
}

func (r *Reporter) generateControlReport(ctx context.Context, tenantID string) (*ComplianceReport, error) {
	factory := &FrameworkFactory{}
	framework, err := factory.CreateFramework(Standard(r.template.Standard))
	if err != nil {
		return nil, nil
	}
	checker := NewChecker()
	checker.RegisterFramework(framework)
	return checker.CheckTenant(ctx, tenantID, Standard(r.template.Standard))
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ComplianceReport 合规报告
type ComplianceReport struct {
	TenantID             string               `json:"tenant_id"`
	Standard             string               `json:"standard"`
	TemplateName         string               `json:"template_name,omitempty"`
	TemplateVersion      string               `json:"template_version,omitempty"`
	TimeRange            TimeRange            `json:"time_range,omitempty"`
	ComplianceRate       float64              `json:"compliance_rate"`
	GeneratedAt          time.Time            `json:"generated_at,omitempty"`
	Controls             []ControlStatus      `json:"controls,omitempty"`
	Summary              map[string]int       `json:"summary,omitempty"`
	UnsupportedControls  []UnsupportedControl `json:"unsupported_controls,omitempty"`
	EvidencePackageID    string               `json:"evidence_package_id,omitempty"`
	EvidenceVerification EvidenceVerification `json:"evidence_verification,omitempty"`
	ComplianceNotice     string               `json:"compliance_notice,omitempty"`
}

// EvidenceVerification is an auditor-facing summary of the evidence package
// verification result used as input for a report.
type EvidenceVerification struct {
	PackageID      string    `json:"package_id,omitempty"`
	JobID          string    `json:"job_id,omitempty"`
	RootHash       string    `json:"root_hash,omitempty"`
	Verified       bool      `json:"verified"`
	Signed         bool      `json:"signed"`
	SignatureValid bool      `json:"signature_valid"`
	SignerKeyID    string    `json:"signer_key_id,omitempty"`
	VerifiedAt     time.Time `json:"verified_at,omitempty"`
	Errors         []string  `json:"errors,omitempty"`
}

// UnsupportedControl identifies controls that Aetheris cannot certify from
// runtime evidence alone.
type UnsupportedControl struct {
	ControlID string `json:"control_id"`
	Reason    string `json:"reason"`
}

func calculateComplianceRate(m Metrics) float64 {
	if m.TotalChecks <= 0 {
		return 100
	}
	weightedViolations := float64(m.Violations) +
		1.0*float64(m.MissingAuditLogs) +
		1.5*float64(m.UnredactedPII)
	rate := 100 * (1 - weightedViolations/float64(m.TotalChecks))
	if rate < 0 {
		rate = 0
	}
	if rate > 100 {
		rate = 100
	}
	return math.Round(rate*10) / 10
}
