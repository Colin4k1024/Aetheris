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
	"fmt"
	"time"
)

// Standard 合规标准
type Standard string

const (
	StandardSOC2     Standard = "SOC2"
	StandardGDPR     Standard = "GDPR"
	StandardHIPAA    Standard = "HIPAA"
	StandardISO27001 Standard = "ISO27001"
)

// ControlCategory 控制类别
type ControlCategory string

const (
	// SOC 2 控制类别
	ControlCategoryCC1 ControlCategory = "CC1" // 控制环境
	ControlCategoryCC2 ControlCategory = "CC2" // 通信与信息
	ControlCategoryCC3 ControlCategory = "CC3" // 风险评估
	ControlCategoryCC4 ControlCategory = "CC4" // 监控活动
	ControlCategoryCC5 ControlCategory = "CC5" // 控制活动
	ControlCategoryCC6 ControlCategory = "CC6" // 逻辑与物理访问控制
	ControlCategoryCC7 ControlCategory = "CC7" // 系统操作
	ControlCategoryCC8 ControlCategory = "CC8" // 变更管理
	ControlCategoryCC9 ControlCategory = "CC9" // 风险缓解

	// GDPR 控制类别
	ControlCategoryGDPR1 ControlCategory = "GDPR1" // 法律基础
	ControlCategoryGDPR2 ControlCategory = "GDPR2" // 数据主体权利
	ControlCategoryGDPR3 ControlCategory = "GDPR3" // 数据保护

	// HIPAA 控制类别
	ControlCategoryHIPAA1 ControlCategory = "HIPAA1" // 行政保障
	ControlCategoryHIPAA2 ControlCategory = "HIPAA2" // 物理保障
	ControlCategoryHIPAA3 ControlCategory = "HIPAA3" // 技术保障
)

// Control 控制项
type Control struct {
	ID            string          `json:"id"`
	Category      ControlCategory `json:"category"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Standard      Standard        `json:"standard"`
	Implemented   bool            `json:"implemented"`
	Automated     bool            `json:"automated"`
	LastCheckedAt *time.Time      `json:"last_checked_at"`
	EvidencePaths []string        `json:"evidence_paths"`
}

// ControlStatus 控制状态
type ControlStatus struct {
	ControlID     string    `json:"control_id"`
	Status        Status    `json:"status"` // Compliant, NonCompliant, NotApplicable
	LastCheckTime time.Time `json:"last_check_time"`
	Findings      []Finding `json:"findings"`
	Evidence      []string  `json:"evidence"`
}

// Status 合规状态
type Status string

const (
	StatusCompliant     Status = "compliant"
	StatusNonCompliant  Status = "non_compliant"
	StatusNotApplicable Status = "not_applicable"
	StatusPending       Status = "pending"
)

// Finding 发现项
type Finding struct {
	Severity  Severity  `json:"severity"`
	Message   string    `json:"message"`
	Resource  string    `json:"resource"`
	Timestamp time.Time `json:"timestamp"`
}

// Severity 严重程度
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ComplianceFramework 合规框架
type ComplianceFramework struct {
	Standard        Standard                  `json:"standard"`
	Controls        map[string]*Control       `json:"controls"`
	ControlStatuses map[string]*ControlStatus `json:"control_statuses"`
}

// NewFramework 创建新框架
func NewFramework(standard Standard) *ComplianceFramework {
	return &ComplianceFramework{
		Standard:        standard,
		Controls:        make(map[string]*Control),
		ControlStatuses: make(map[string]*ControlStatus),
	}
}

// AddControl 添加控制项
func (f *ComplianceFramework) AddControl(control *Control) {
	f.Controls[control.ID] = control
}

// GetControl 获取控制项
func (f *ComplianceFramework) GetControl(id string) *Control {
	return f.Controls[id]
}

// UpdateControlStatus 更新控制状态
func (f *ComplianceFramework) UpdateControlStatus(controlID string, status *ControlStatus) {
	f.ControlStatuses[controlID] = status
	if control, ok := f.Controls[controlID]; ok {
		now := time.Now()
		control.LastCheckedAt = &now
		control.Implemented = status.Status == StatusCompliant
	}
}

// GetComplianceRate 获取合规率
func (f *ComplianceFramework) GetComplianceRate() float64 {
	if len(f.Controls) == 0 {
		return 0
	}

	var compliant, total int
	for _, control := range f.Controls {
		if status, ok := f.ControlStatuses[control.ID]; ok {
			if status.Status == StatusCompliant {
				compliant++
			}
			total++
		}
	}

	if total == 0 {
		return 0
	}

	return float64(compliant) / float64(total) * 100
}

// GetControlsByCategory 获取指定类别的控制项
func (f *ComplianceFramework) GetControlsByCategory(category ControlCategory) []*Control {
	var result []*Control
	for _, control := range f.Controls {
		if control.Category == category {
			result = append(result, control)
		}
	}
	return result
}

// FrameworkFactory 创建框架工厂
type FrameworkFactory struct{}

// CreateFramework 创建指定标准的框架
func (f *FrameworkFactory) CreateFramework(standard Standard) (*ComplianceFramework, error) {
	framework := NewFramework(standard)

	switch standard {
	case StandardSOC2:
		f.addSOC2Controls(framework)
	case StandardGDPR:
		f.addGDPRControls(framework)
	case StandardHIPAA:
		f.addHIPAAControls(framework)
	default:
		return nil, fmt.Errorf("unsupported standard: %s", standard)
	}

	return framework, nil
}

// addSOC2Controls 添加 SOC 2 控制项
func (f *FrameworkFactory) addSOC2Controls(framework *ComplianceFramework) {
	controls := []*Control{
		// CC1 - 控制环境
		{ID: "CC1.1", Category: ControlCategoryCC1, Name: "Integrity and Ethical Values", Description: "实体展示对诚信和道德价值观的承诺"},
		{ID: "CC1.2", Category: ControlCategoryCC1, Name: "Board Oversight", Description: "董事会或审计委员会对内部控制的监督"},
		{ID: "CC1.3", Category: ControlCategoryCC1, Name: "Organizational Structure", Description: "组织结构和责任分配"},

		// CC2 - 通信与信息
		{ID: "CC2.1", Category: ControlCategoryCC2, Name: "Information Communication", Description: "与相关方沟通内部控制责任"},
		{ID: "CC2.2", Category: ControlCategoryCC2, Name: "Internal Communication", Description: "内部沟通控制信息"},

		// CC3 - 风险评估
		{ID: "CC3.1", Category: ControlCategoryCC3, Name: "Risk Identification", Description: "识别和分析风险"},
		{ID: "CC3.2", Category: ControlCategoryCC3, Name: "Risk Analysis", Description: "分析欺诈风险和非法财务报告风险"},

		// CC4 - 监控活动
		{ID: "CC4.1", Category: ControlCategoryCC4, Name: "Monitoring", Description: "选择和开发持续和/或单独评估"},
		{ID: "CC4.2", Category: ControlCategoryCC4, Name: "Evaluation", Description: "评估并沟通内部控制的缺陷"},

		// CC5 - 控制活动
		{ID: "CC5.1", Category: ControlCategoryCC5, Name: "Control Activities", Description: "选择和发展控制活动"},
		{ID: "CC5.2", Category: ControlCategoryCC5, Name: "Technology Controls", Description: "通过技术部署控制活动"},

		// CC6 - 逻辑与物理访问控制
		{ID: "CC6.1", Category: ControlCategoryCC6, Name: "Logical Access", Description: "逻辑访问安全"},
		{ID: "CC6.2", Category: ControlCategoryCC6, Name: "Access Provisioning", Description: "访问配置和移除"},
		{ID: "CC6.3", Category: ControlCategoryCC6, Name: "Physical Access", Description: "物理访问安全"},

		// CC7 - 系统操作
		{ID: "CC7.1", Category: ControlCategoryCC7, Name: "System Operations", Description: "系统操作管理"},
		{ID: "CC7.2", Category: ControlCategoryCC7, Name: "Change Management", Description: "变更管理流程"},

		// CC8 - 变更管理
		{ID: "CC8.1", Category: ControlCategoryCC8, Name: "Change Management", Description: "变更管理流程"},

		// CC9 - 风险缓解
		{ID: "CC9.1", Category: ControlCategoryCC9, Name: "Risk Mitigation", Description: "风险缓解流程"},
	}

	for _, c := range controls {
		framework.AddControl(c)
	}
}

// addGDPRControls 添加 GDPR 控制项
func (f *FrameworkFactory) addGDPRControls(framework *ComplianceFramework) {
	controls := []*Control{
		{ID: "GDPR.1", Category: ControlCategoryGDPR1, Name: "Lawful Basis", Description: "处理的法律基础"},
		{ID: "GDPR.2", Category: ControlCategoryGDPR1, Name: "Consent", Description: "数据主体同意"},
		{ID: "GDPR.3", Category: ControlCategoryGDPR2, Name: "Access Right", Description: "数据访问权"},
		{ID: "GDPR.4", Category: ControlCategoryGDPR2, Name: "Rectification", Description: "数据更正权"},
		{ID: "GDPR.5", Category: ControlCategoryGDPR2, Name: "Erasure", Description: "被遗忘权"},
		{ID: "GDPR.6", Category: ControlCategoryGDPR2, Name: "Portability", Description: "数据可携带权"},
		{ID: "GDPR.7", Category: ControlCategoryGDPR3, Name: "Data Protection", Description: "数据保护措施"},
		{ID: "GDPR.8", Category: ControlCategoryGDPR3, Name: "Breach Notification", Description: "数据泄露通知"},
	}

	for _, c := range controls {
		framework.AddControl(c)
	}
}

// addHIPAAControls 添加 HIPAA 控制项
func (f *FrameworkFactory) addHIPAAControls(framework *ComplianceFramework) {
	controls := []*Control{
		// 行政保障
		{ID: "HIPAA.1", Category: ControlCategoryHIPAA1, Name: "Security Management", Description: "安全管理制度"},
		{ID: "HIPAA.2", Category: ControlCategoryHIPAA1, Name: "Workforce Security", Description: "员工安全"},
		{ID: "HIPAA.3", Category: ControlCategoryHIPAA1, Name: "Information Access", Description: "信息访问管理"},

		// 物理保障
		{ID: "HIPAA.4", Category: ControlCategoryHIPAA2, Name: "Facility Access", Description: "设施访问控制"},
		{ID: "HIPAA.5", Category: ControlCategoryHIPAA2, Name: "Workstation Use", Description: "工作站使用政策"},

		// 技术保障
		{ID: "HIPAA.6", Category: ControlCategoryHIPAA3, Name: "Access Control", Description: "访问控制"},
		{ID: "HIPAA.7", Category: ControlCategoryHIPAA3, Name: "Audit Controls", Description: "审计控制"},
		{ID: "HIPAA.8", Category: ControlCategoryHIPAA3, Name: "Transmission Security", Description: "传输安全"},
	}

	for _, c := range controls {
		framework.AddControl(c)
	}
}
