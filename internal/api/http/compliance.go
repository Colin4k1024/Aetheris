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

package http

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"rag-platform/pkg/compliance"
)

// ComplianceTemplates 获取合规模板列表（3.0-M4）
// GET /api/compliance/templates
func (h *Handler) ComplianceTemplates(c context.Context, ctx *app.RequestContext) {
	templates := compliance.ListTemplates()

	result := make([]map[string]interface{}, 0, len(templates))
	for _, t := range templates {
		result = append(result, map[string]interface{}{
			"name":           t.Name,
			"standard":       t.Standard,
			"retention_days": t.RetentionDays,
			"export_format":  t.ExportFormat,
		})
	}

	ctx.JSON(consts.StatusOK, map[string]interface{}{
		"templates": result,
	})
}

// ComplianceApply 应用合规策略（3.0-M4）
// POST /api/compliance/apply
func (h *Handler) ComplianceApply(c context.Context, ctx *app.RequestContext) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Standard string `json:"standard"` // GDPR, SOX, HIPAA
	}

	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.TenantID == "" {
		ctx.JSON(consts.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
		return
	}

	// 使用 compliance Checker 检查合规性
	checker := compliance.NewChecker()

	var framework *compliance.ComplianceFramework
	switch req.Standard {
	case "GDPR", "SOX", "HIPAA":
		factory := &compliance.FrameworkFactory{}
		f, _ := factory.CreateFramework(compliance.Standard(req.Standard))
		framework = f
	default:
		ctx.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid standard, must be GDPR, SOX, or HIPAA"})
		return
	}

	if framework != nil {
		checker.RegisterFramework(framework)
	}

	report, err := checker.CheckTenant(c, req.TenantID, compliance.Standard(req.Standard))
	if err != nil {
		ctx.JSON(consts.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("compliance check failed: %v", err),
		})
		return
	}

	ctx.JSON(consts.StatusOK, map[string]interface{}{
		"tenant_id":         req.TenantID,
		"standard":          req.Standard,
		"compliant":         report.IsCompliant(),
		"compliance_rate":   report.ComplianceRate,
		"critical_findings": report.GetCriticalFindings(),
	})
}

// ComplianceReport 生成合规报告（3.0-M4）
// POST /api/compliance/report
func (h *Handler) ComplianceReport(c context.Context, ctx *app.RequestContext) {
	var req struct {
		TenantID string `json:"tenant_id"`
		Standard string `json:"standard"` // GDPR, SOX, HIPAA
		Start    string `json:"start"`
		End      string `json:"end"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(consts.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.TenantID == "" || req.Standard == "" {
		ctx.JSON(consts.StatusBadRequest, map[string]string{"error": "tenant_id and standard are required"})
		return
	}

	// 解析时间范围
	var timeRange compliance.TimeRange
	now := time.Now()
	timeRange.End = now

	if req.Start != "" {
		if t, err := time.Parse("2006-01-02", req.Start); err == nil {
			timeRange.Start = t
		}
	}
	if req.End != "" {
		if t, err := time.Parse("2006-01-02", req.End); err == nil {
			timeRange.End = t
		}
	}
	if timeRange.Start.IsZero() {
		timeRange.Start = now.AddDate(0, -1, 0) // 默认最近一个月
	}

	// 获取模板并生成报告
	template := compliance.GetTemplate(req.Standard)
	if template == nil {
		ctx.JSON(consts.StatusBadRequest, map[string]string{"error": "template not found"})
		return
	}

	reporter := compliance.NewReporter(*template)
	report, err := reporter.GenerateReport(c, req.TenantID, timeRange)
	if err != nil {
		ctx.JSON(consts.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("generate report failed: %v", err),
		})
		return
	}

	ctx.JSON(consts.StatusOK, map[string]interface{}{
		"tenant_id":       req.TenantID,
		"standard":        req.Standard,
		"time_range":      timeRange,
		"compliant":       report.IsCompliant(),
		"compliance_rate": report.ComplianceRate,
		"generated_at":    time.Now().Format(time.RFC3339),
	})
}
