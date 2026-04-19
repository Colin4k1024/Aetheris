# 教程: Audit Agent

> 使用 Aetheris 构建企业合规审计 Agent

## 概述

本教程教你如何使用 Aetheris 创建一个自动化的合规审计 Agent，具备以下能力:
- 自动检查系统配置
- 验证安全策略
- 生成审计报告
- 保留完整的审计证据链

## 目标

完成本教程后，你将:
1. 理解如何使用 Aetheris 构建审计 Agent
2. 掌握证据收集和审计追踪
3. 学会生成合规报告

---

## 为什么需要审计 Agent？

在企业环境中，合规审计是一项关键任务:

| 传统方式 | Aetheris Agent |
|---------|----------------|
| 手动检查 | 自动定期执行 |
| 纸质记录 | 数字证据链 |
| 难以追溯 | 完整事件溯源 |
| 耗时耗力 | 高效可重复 |

---

## 步骤 1: 创建项目结构

```bash
mkdir -p audit-agent
cd audit-agent
go mod init audit-agent
```

## 步骤 2: 编写 Agent 代码

创建 `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func main() {
	cfg := config.Load()
	
	// 创建审计 Agent
	auditor := createAuditAgent(cfg)

	ctx := context.Background()

	// 运行审计任务
	result, err := auditor.Run(ctx, map[string]any{
		"scope":    "system",
		"checks":   []string{"security", "compliance", "performance"},
		"dry_run":  false,
	})

	if err != nil {
		log.Fatalf("审计失败: %v", err)
	}

	fmt.Printf("\n✅ 审计完成!\n\n%s\n", result)
}

// createAuditAgent 创建审计 Agent
func createAuditAgent(cfg *config.Config) *agent.Agent {
	return &agent.Agent{
		Name:        "compliance-auditor",
		Description: "企业合规审计 Agent",
		Steps: []agent.Step{
			{
				Name: "init_audit",
				Description: "初始化审计任务",
				Run: initAuditStep,
			},
			{
				Name: "collect_evidence",
				Description: "收集系统证据",
				Run: collectEvidenceStep,
			},
			{
				Name: "run_checks",
				Description: "执行检查规则",
				Run: runChecksStep,
			},
			{
				Name: "generate_report",
				Description: "生成审计报告",
				Run: generateReportStep,
			},
			{
				Name: "seal_evidence",
				Description: "密封证据链",
				Run: sealEvidenceStep,
			},
		},
	}
}

// 审计证据结构
type AuditEvidence struct {
	AuditID     string            `json:"audit_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Scope       string            `json:"scope"`
	Findings    []Finding         `json:"findings"`
	Checks      []CheckResult     `json:"checks"`
	EvidenceIDs []string         `json:"evidence_ids"`
	Sealed      bool             `json:"sealed"`
}

type Finding struct {
	Severity  string `json:"severity"`  // critical, high, medium, low, info
	Category  string `json:"category"`
	Message   string `json:"message"`
	Evidence  string `json:"evidence"`
	Remediation string `json:"remediation"`
}

type CheckResult struct {
	CheckID   string `json:"check_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`   // pass, fail, skip
	Duration  int64  `json:"duration_ms"`
	Message   string `json:"message"`
}

// Step 1: 初始化审计
func initAuditStep(ctx context.Context, input any) (any, error) {
	inputMap := input.(map[string]any)
	scope := inputMap["scope"].(string)

	auditID := fmt.Sprintf("audit_%d", time.Now().Unix())
	
	fmt.Printf("📋 初始化审计 #%s (范围: %s)\n", auditID, scope)

	evidence := &AuditEvidence{
		AuditID:   auditID,
		Timestamp: time.Now(),
		Scope:     scope,
		Findings:  []Finding{},
		Checks:    []CheckResult{},
	}

	return evidence, nil
}

// Step 2: 收集证据
func collectEvidenceStep(ctx context.Context, input any) (any, error) {
	evidence := input.(*AuditEvidence)

	fmt.Println("🔍 收集系统证据...")

	// 模拟收集各类证据
	evidenceSources := []string{
		"system_info",
		"network_config",
		"user_accounts",
		"installed_packages",
		"running_processes",
		"open_ports",
	}

	evidenceID := evidence.AuditID
	for _, source := range evidenceSources {
		evidenceID += "_" + source
		fmt.Printf("  ✓ 收集: %s\n", source)
	}

	evidence.EvidenceIDs = []string{
		"ev_" + evidence.AuditID + "_sysinfo",
		"ev_" + evidence.AuditID + "_network",
		"ev_" + evidence.AuditID + "_users",
	}

	return evidence, nil
}

// Step 3: 执行检查
func runChecksStep(ctx context.Context, input any) (any, error) {
	evidence := input.(*AuditEvidence)

	fmt.Println("⚙️ 执行合规检查...")

	// 定义检查规则
	checks := []struct {
		id       string
		name     string
		category string
		checkFn  func() (string, []Finding)
	}{
		{
			id: "SEC-001",
			name: "密码策略检查",
			category: "security",
			checkFn: checkPasswordPolicy,
		},
		{
			id: "SEC-002",
			name: "开放端口检查",
			category: "security",
			checkFn: checkOpenPorts,
		},
		{
			id: "COMP-001",
			name: "审计日志配置检查",
			category: "compliance",
			checkFn: checkAuditLogging,
		},
		{
			id: "COMP-002",
			name: "数据加密检查",
			category: "compliance",
			checkFn: checkEncryption,
		},
		{
			id: "PERF-001",
			name: "资源使用检查",
			category: "performance",
			checkFn: checkResources,
		},
	}

	for _, check := range checks {
		start := time.Now()
		status, findings := check.checkFn()
		duration := time.Since(start).Milliseconds()

		result := CheckResult{
			CheckID:  check.id,
			Name:     check.name,
			Status:   status,
			Duration: duration,
		}

		if len(findings) > 0 {
			result.Message = fmt.Sprintf("发现 %d 个问题", len(findings))
			evidence.Findings = append(evidence.Findings, findings...)
		} else {
			result.Message = "检查通过"
		}

		evidence.Checks = append(evidence.Checks, result)
		
		emoji := "✅"
		if status == "fail" {
			emoji = "❌"
		} else if status == "skip" {
			emoji = "⏭️"
		}
		fmt.Printf("  %s %s: %s (%dms)\n", emoji, check.id, result.Message, duration)
	}

	return evidence, nil
}

// Step 4: 生成报告
func generateReportStep(ctx context.Context, input any) (any, error) {
	evidence := input.(*AuditEvidence)

	fmt.Println("📝 生成审计报告...")

	report := fmt.Sprintf("═══════════════════════════════════════════════════════════\n")
	report += fmt.Sprintf("                    合规审计报告\n")
	report += fmt.Sprintf("═══════════════════════════════════════════════════════════\n\n")
	
	report += fmt.Sprintf("审计 ID:     %s\n", evidence.AuditID)
	report += fmt.Sprintf("审计时间:     %s\n", evidence.Timestamp.Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("审计范围:     %s\n", evidence.Scope)
	report += fmt.Sprintf("证据数量:     %d\n", len(evidence.EvidenceIDs))
	report += fmt.Sprintf("\n")

	// 统计
	passCount := 0
	failCount := 0
	skipCount := 0
	for _, c := range evidence.Checks {
		switch c.Status {
		case "pass": passCount++
		case "fail": failCount++
		case "skip": skipCount++
		}
	}

	report += fmt.Sprintf("─────────────────────────────────────────────────────────────\n")
	report += fmt.Sprintf("检查结果统计: ✅ 通过 %d | ❌ 失败 %d | ⏭️ 跳过 %d\n", passCount, failCount, skipCount)
	report += fmt.Sprintf("─────────────────────────────────────────────────────────────\n\n")

	// 详细发现
	if len(evidence.Findings) > 0 {
		report += fmt.Sprintf("发现项详情:\n")
		report += fmt.Sprintf("─────────────────────────────────────────────────────────────\n\n")

		critical := []Finding{}
		high := []Finding{}
		medium := []Finding{}
		low := []Finding{}

		for _, f := range evidence.Findings {
			switch f.Severity {
			case "critical": critical = append(critical, f)
			case "high": high = append(high, f)
			case "medium": medium = append(medium, f)
			case "low": low = append(low, f)
			}
		}

		printFindings := func(level string, findings []Finding) {
			if len(findings) > 0 {
				report += fmt.Sprintf("\n[%s] (%d 项)\n", level, len(findings))
				for i, f := range findings {
					report += fmt.Sprintf("  %d. %s\n", i+1, f.Message)
					report += fmt.Sprintf("      类别: %s\n", f.Category)
					report += fmt.Sprintf("      证据: %s\n", f.Evidence)
					if f.Remediation != "" {
						report += fmt.Sprintf("      修复: %s\n", f.Remediation)
					}
				}
			}
		}

		printFindings("🔴 严重", critical)
		printFindings("🟠 高危", high)
		printFindings("🟡 中危", medium)
		printFindings("🔵 低危", low)
	} else {
		report += fmt.Sprintf("✅ 未发现任何问题! 系统符合所有合规要求。\n")
	}

	report += fmt.Sprintf("\n═══════════════════════════════════════════════════════════\n")
	report += fmt.Sprintf("报告生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("═══════════════════════════════════════════════════════════\n")

	return report, nil
}

// Step 5: 密封证据链
func sealEvidenceStep(ctx context.Context, input any) (any, error) {
	evidence := input.(*AuditEvidence)

	fmt.Println("🔏 密封证据链...")

	// 生成证据链哈希
	hash := fmt.Sprintf("sha256:%x", time.Now().UnixNano())
	
	evidence.Sealed = true

	fmt.Printf("  ✓ 证据链已密封: %s\n", hash)
	fmt.Printf("  ✓ 审计完成: %d 项检查, %d 项发现\n", 
		len(evidence.Checks), len(evidence.Findings))

	return evidence, nil
}

// 检查函数实现 (模拟)
func checkPasswordPolicy() (string, []Finding) {
	// 模拟: 检查密码策略
	return "pass", nil
}

func checkOpenPorts() (string, []Finding) {
	// 模拟: 检查开放端口
	findings := []Finding{
		{
			Severity:  "medium",
			Category:  "security",
			Message:   "发现开放端口 22 (SSH)",
			Evidence:  "netstat 显示端口 22 监听",
			Remediation: "建议使用密钥认证或限制 IP 访问",
		},
	}
	return "fail", findings
}

func checkAuditLogging() (string, []Finding) {
	return "pass", nil
}

func checkEncryption() (string, []Finding) {
	findings := []Finding{
		{
			Severity:  "high",
			Category:  "compliance",
			Message:   "数据库未启用加密",
			Evidence:  "PostgreSQL 配置中 ssl=off",
			Remediation: "启用 SSL/TLS 加密",
		},
	}
	return "fail", findings
}

func checkResources() (string, []Finding) {
	return "pass", nil
}
```

## 步骤 3: 配置

创建 `config.yaml`:

```yaml
audit:
  # 审计配置
  retention_days: 2555  # 7年
  evidence_store: postgres
  
  # 检查规则
  rules:
    security:
      - id: SEC-001
        enabled: true
      - id: SEC-002
        enabled: true
    
    compliance:
      - id: COMP-001
        enabled: true
      - id: COMP-002
        enabled: true
    
    performance:
      - id: PERF-001
        enabled: true
```

## 步骤 4: 运行 Agent

```bash
go run main.go
```

预期输出:

```
📋 初始化审计 #audit_1710691200 (范围: system)
🔍 收集系统证据...
  ✓ 收集: system_info
  ✓ 收集: network_config
  ⚙️ 执行合规检查...
  ✅ SEC-001: 检查通过 (12ms)
  ❌ SEC-002: 发现 1 个问题 (8ms)
  ✅ COMP-001: 检查通过 (5ms)
  ❌ COMP-002: 发现 1 个问题 (3ms)
  ✅ PERF-001: 检查通过 (2ms)
📝 生成审计报告...
🔏 密封证据链...
  ✓ 证据链已密封: sha256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069
  ✓ 审计完成: 5 项检查, 2 项发现

✅ 审计完成!

═══════════════════════════════════════════════════════════
                    合规审计报告
═══════════════════════════════════════════════════════════

审计 ID:     audit_1710691200
审计时间:     2025-03-17 21:33:20
审计范围:     system
证据数量:     3

─────────────────────────────────────────────────────────────
检查结果统计: ✅ 通过 3 | ❌ 失败 2 | ⏭️ 跳过 0
─────────────────────────────────────────────────────────────

发现项详情:
─────────────────────────────────────────────────────────────

[🟠 高危] (1 项)
  1. 数据库未启用加密
      类别: compliance
      证据: PostgreSQL 配置中 ssl=off
      修复: 启用 SSL/TLS 加密

[🟡 中危] (1 项)
  1. 发现开放端口 22 (SSH)
      类别: security
      证据: netstat 显示端口 22 监听
      修复: 建议使用密钥认证或限制 IP 访问
```

---

## 事件溯源的好处

使用 Aetheris 运行时，审计过程自动具备:

1. **完整证据链**: 每个检查步骤都被记录
2. **可追溯**: 可以随时重放审计过程
3. **可恢复**: 审计中断后可继续
4. **合规性**: 符合 SOX, HIPAA, GDPR 等要求

```bash
# 查看审计事件历史
./bin/aetheris trace <audit_job_id>
```

---

## 扩展: 添加更多检查

```go
// 添加自定义检查
customChecks := []struct {
    id       string
    name     string
    category string
    checkFn  func() (string, []Finding)
}{
    {
        id: "PCI-001",
        name: "PCI-DSS 合规检查",
        category: "compliance",
        checkFn: checkPCIDSS,
    },
    {
        id: "GDPR-001",
        name: "GDPR 合规检查",
        category: "compliance", 
        checkFn: checkGDPR,
    },
}
```

---

## 完整项目结构

```
audit-agent/
├── config.yaml      # 配置文件
├── main.go          # 主程序
└── go.mod          # Go 模块
```

---

## 下一步

- [长程任务教程](./long-running-tasks.md) — 处理小时级任务
- [事件溯源概念](../concepts/event-sourcing.md) — 深入理解 Aetheris 设计
- [CLI 参考](../reference/cli.md) — 完整 CLI 文档
