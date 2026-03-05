// Copyright 2026 Aetheris
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

package sla

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SLAReport SLA 报告
type SLAReport struct {
	TenantID     string         `json:"tenant_id"`
	PeriodStart  time.Time      `json:"period_start"`
	PeriodEnd    time.Time      `json:"period_end"`
	TotalJobs    int64          `json:"total_jobs"`
	CompletedJobs int64         `json:"completed_jobs"`
	FailedJobs   int64          `json:"failed_jobs"`
	BreachedJobs int64          `json:"breached_jobs"`
	SLOs         []SLOReportItem `json:"slos"`
	GeneratedAt  time.Time      `json:"generated_at"`
}

// SLOReportItem SLO 报告项
type SLOReportItem struct {
	Name        string  `json:"name"`
	Type        SLOType `json:"type"`
	Target      float64 `json:"target"`
	Actual      float64 `json:"actual"`
	Status      string  `json:"status"` // met, breached, unknown
	Breaches    int     `json:"breaches"`
}

// Reporter SLA 报告生成器
type Reporter struct {
	monitor *Monitor
}

// NewReporter 创建报告生成器
func NewReporter(monitor *Monitor) *Reporter {
	return &Reporter{monitor: monitor}
}

// GenerateReport 生成 SLA 报告
func (r *Reporter) GenerateReport(ctx context.Context, tenantID string, period time.Duration) (*SLAReport, error) {
	report := &SLAReport{
		TenantID:    tenantID,
		PeriodStart: time.Now().Add(-period),
		PeriodEnd:   time.Now(),
		GeneratedAt: time.Now(),
	}

	r.monitor.mu.RLock()
	contract, hasContract := r.monitor.contracts[tenantID]
	r.monitor.mu.RUnlock()

	if !hasContract {
		return nil, fmt.Errorf("no SLA contract for tenant %s", tenantID)
	}

	// TODO: 从实际测量数据中获取
	// 这里模拟数据
	report.TotalJobs = 100
	report.CompletedJobs = 95
	report.FailedJobs = 3
	report.BreachedJobs = 2

	// 生成 SLO 报告项
	for _, slo := range contract.SLOs {
		item := SLOReportItem{
			Name:   slo.Name,
			Type:   slo.Type,
			Target: slo.Target,
		}

		// 根据类型计算实际值
		switch slo.Type {
		case SLOTypeAvailability:
			item.Actual = float64(report.CompletedJobs) / float64(report.TotalJobs) * 100
			if item.Actual >= slo.Target {
				item.Status = "met"
			} else {
				item.Status = "breached"
				item.Breaches = 1
			}
		case SLOTypeErrorRate:
			item.Actual = float64(report.FailedJobs) / float64(report.TotalJobs) * 100
			if item.Actual <= slo.Target {
				item.Status = "met"
			} else {
				item.Status = "breached"
				item.Breaches = 1
			}
		default:
			item.Status = "unknown"
		}

		report.SLOs = append(report.SLOs, item)
	}

	return report, nil
}

// FormatJSON JSON 格式报告
func (r *Reporter) FormatJSON(report *SLAReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// FormatSummary 文本格式摘要
func (r *Reporter) FormatSummary(report *SLAReport) string {
	summary := fmt.Sprintf("SLA Report for Tenant: %s\n", report.TenantID)
	summary += fmt.Sprintf("Period: %s - %s\n", report.PeriodStart.Format(time.RFC3339), report.PeriodEnd.Format(time.RFC3339))
	summary += fmt.Sprintf("Total Jobs: %d\n", report.TotalJobs)
	summary += fmt.Sprintf("Completed: %d\n", report.CompletedJobs)
	summary += fmt.Sprintf("Failed: %d\n", report.FailedJobs)
	summary += fmt.Sprintf("Breached: %d\n", report.BreachedJobs)
	summary += "\nSLO Status:\n"

	for _, slo := range report.SLOs {
		statusIcon := "✓"
		if slo.Status != "met" {
			statusIcon = "✗"
		}
		summary += fmt.Sprintf("  %s %s: %.2f%% (target: %.2f%%)\n", statusIcon, slo.Name, slo.Actual, slo.Target)
	}

	return summary
}
