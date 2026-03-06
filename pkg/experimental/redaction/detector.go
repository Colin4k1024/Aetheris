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

package redaction

import (
	"regexp"
	"strings"
)

// PIIType PII 类型
type PIIType string

const (
	PIITypeEmail         PIIType = "email"
	PIITypePhone         PIIType = "phone"
	PIITypeSSN           PIIType = "ssn" // Social Security Number
	PIITypeCreditCard    PIIType = "credit_card"
	PIITypeIPAddress     PIIType = "ip_address"
	PIITypeAddress       PIIType = "address"
	PIITypeDateOfBirth   PIIType = "date_of_birth"
	PIITypePassport      PIIType = "passport"
	PIITypeDriverLicense PIIType = "driver_license"
)

// PIIDetector PII 检测器
type PIIDetector struct {
	patterns map[PIIType]*regexp.Regexp
}

// NewPIIDetector 创建 PII 检测器
func NewPIIDetector() *PIIDetector {
	return &PIIDetector{
		patterns: map[PIIType]*regexp.Regexp{
			PIITypeEmail:         regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			PIITypePhone:         regexp.MustCompile(`(\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
			PIITypeSSN:           regexp.MustCompile(`\b\d{3}[-]?\d{2}[-]?\d{4}\b`),
			PIITypeCreditCard:    regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`),
			PIITypeIPAddress:     regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`),
			PIITypeAddress:       regexp.MustCompile(`\b\d{1,5}\s+[\w\s]+(?:Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Lane|Ln|Drive|Dr|Court|Ct|Place|Pl)\b`),
			PIITypeDateOfBirth:   regexp.MustCompile(`\b(?:0?[1-9]|1[0-2])[/-](?:0?[1-9]|[12][0-9]|3[01])[/-](?:19|20)\d{2}\b`),
			PIITypePassport:      regexp.MustCompile(`\b[A-Z]{1,2}[0-9]{6,9}\b`),
			PIITypeDriverLicense: regexp.MustCompile(`\b[A-Z]{1,2}[0-9]{5,8}\b`),
		},
	}
}

// Detect 检测文本中的 PII
func (d *PIIDetector) Detect(text string) []PIIDetection {
	var detections []PIIDetection

	for piiType, pattern := range d.patterns {
		matches := pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			detections = append(detections, PIIDetection{
				Type:       piiType,
				Start:      match[0],
				End:        match[1],
				Value:      text[match[0]:match[1]],
				Confidence: d.getConfidence(piiType),
			})
		}
	}

	return detections
}

// DetectInMap 检测 map 中的 PII
func (d *PIIDetector) DetectInMap(m map[string]interface{}) map[string][]PIIDetection {
	result := make(map[string][]PIIDetection)

	for key, value := range m {
		str, ok := value.(string)
		if !ok {
			continue
		}

		detections := d.Detect(str)
		if len(detections) > 0 {
			result[key] = detections
		}
	}

	return result
}

// RedactInText 对文本中的 PII 进行脱敏
func (d *PIIDetector) RedactInText(text string, mode RedactionMode) string {
	result := text

	detections := d.Detect(text)
	// 从后向前替换，避免索引偏移
	for i := len(detections) - 1; i >= 0; i-- {
		det := detections[i]
		replacement := d.getReplacement(det.Type, mode)
		result = result[:det.Start] + replacement + result[det.End:]
	}

	return result
}

// RedactInMap 对 map 中的 PII 进行脱敏
func (d *PIIDetector) RedactInMap(m map[string]interface{}, mode RedactionMode) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		switch v := value.(type) {
		case string:
			result[key] = d.RedactInText(v, mode)
		case map[string]interface{}:
			result[key] = d.RedactInMap(v, mode)
		case []interface{}:
			result[key] = d.redactSlice(v, mode)
		default:
			result[key] = value
		}
	}

	return result
}

func (d *PIIDetector) redactSlice(slice []interface{}, mode RedactionMode) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case string:
			result[i] = d.RedactInText(val, mode)
		case map[string]interface{}:
			result[i] = d.RedactInMap(val, mode)
		default:
			result[i] = val
		}
	}
	return result
}

func (d *PIIDetector) getReplacement(piiType PIIType, mode RedactionMode) string {
	switch mode {
	case RedactionModeRedact:
		return "***REDACTED***"
	case RedactionModeHash:
		// 简化实现，实际应该用 hash
		return "***HASH***"
	case RedactionModeEncrypt:
		return "***ENCRYPTED***"
	case RedactionModeRemove:
		return ""
	default:
		return "***REDACTED***"
	}
}

func (d *PIIDetector) getConfidence(piiType PIIType) float64 {
	// 根据模式复杂度返回置信度
	highConfidence := []PIIType{PIITypeSSN, PIITypeCreditCard, PIITypeEmail}
	mediumConfidence := []PIIType{PIITypePhone, PIITypeIPAddress, PIITypePassport}

	for _, t := range highConfidence {
		if t == piiType {
			return 0.9
		}
	}
	for _, t := range mediumConfidence {
		if t == piiType {
			return 0.7
		}
	}
	return 0.5
}

// PIIDetection PII 检测结果
type PIIDetection struct {
	Type       PIIType `json:"type"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// GetBuiltInFields 获取内置的 PII 字段名
func GetBuiltInFields() map[string]PIIType {
	return map[string]PIIType{
		"email":           PIITypeEmail,
		"phone":           PIITypePhone,
		"telephone":       PIITypePhone,
		"mobile":          PIITypePhone,
		"ssn":             PIITypeSSN,
		"social_security": PIITypeSSN,
		"credit_card":     PIITypeCreditCard,
		"card_number":     PIITypeCreditCard,
		"ip_address":      PIITypeIPAddress,
		"ip":              PIITypeIPAddress,
		"address":         PIITypeAddress,
		"home_address":    PIITypeAddress,
		"date_of_birth":   PIITypeDateOfBirth,
		"dob":             PIITypeDateOfBirth,
		"birth_date":      PIITypeDateOfBirth,
		"passport":        PIITypePassport,
		"driver_license":  PIITypeDriverLicense,
		"license":         PIITypeDriverLicense,
	}
}

// AutoDetectPolicy 自动检测并脱敏的策略
type AutoDetectPolicy struct {
	EnabledTypes []PIIType          // 启用的 PII 类型
	Mode         RedactionMode      // 脱敏模式
	FieldMapping map[string]PIIType // 字段名到 PII 类型的映射
}

// NewAutoDetectPolicy 创建自动检测策略
func NewAutoDetectPolicy(mode RedactionMode) *AutoDetectPolicy {
	return &AutoDetectPolicy{
		EnabledTypes: []PIIType{
			PIITypeEmail,
			PIITypePhone,
			PIITypeSSN,
			PIITypeCreditCard,
		},
		Mode:         mode,
		FieldMapping: GetBuiltInFields(),
	}
}

// Apply 对数据应用自动脱敏
func (p *AutoDetectPolicy) Apply(data map[string]interface{}) map[string]interface{} {
	detector := NewPIIDetector()

	result := make(map[string]interface{})
	for key, value := range data {
		// 先检查字段名是否匹配
		if piiType, ok := p.FieldMapping[strings.ToLower(key)]; ok {
			if p.isEnabled(piiType) {
				switch v := value.(type) {
				case string:
					result[key] = detector.RedactInText(v, p.Mode)
				case map[string]interface{}:
					result[key] = p.Apply(v)
				default:
					result[key] = value
				}
				continue
			}
		}

		// 文本内容检测
		if str, ok := value.(string); ok {
			detected := detector.Detect(str)
			if len(detected) > 0 {
				result[key] = detector.RedactInText(str, p.Mode)
				continue
			}
		}

		// 递归处理嵌套结构
		if nested, ok := value.(map[string]interface{}); ok {
			result[key] = p.Apply(nested)
			continue
		}

		result[key] = value
	}

	return result
}

func (p *AutoDetectPolicy) isEnabled(piiType PIIType) bool {
	for _, t := range p.EnabledTypes {
		if t == piiType {
			return true
		}
	}
	return false
}
