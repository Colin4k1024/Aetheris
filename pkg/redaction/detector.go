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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
	patterns   map[PIIType]*regexp.Regexp
	hashSalt   string // Hash 模式的 salt
	encryptKey []byte // Encrypt 模式的 AES 密钥（32 字节）
}

// NewPIIDetector 创建 PII 检测器（无密钥，Hash/Encrypt 模式降级为 Redact）
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

// SetHashSalt 设置 Hash 模式的 salt
func (d *PIIDetector) SetHashSalt(salt string) {
	d.hashSalt = salt
}

// SetEncryptKey 设置 Encrypt 模式的 AES 密钥（必须 32 字节）
func (d *PIIDetector) SetEncryptKey(key []byte) {
	d.encryptKey = key
}

// NewPIIDetectorWithKeys 创建带密钥的 PII 检测器
func NewPIIDetectorWithKeys(hashSalt string, encryptKey []byte) *PIIDetector {
	d := NewPIIDetector()
	d.hashSalt = hashSalt
	d.encryptKey = encryptKey
	return d
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
		replacement := d.getReplacement(det.Value, det.Type, mode)
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

// getReplacement 根据模式对原始值执行真实变换，返回替换字符串。
// Hash 模式：SHA256(value + salt) → hash:hex；不同输入产生不同摘要。
// Encrypt 模式：AES-256-GCM 加密 → enc:hex；缺密钥时降级为 Redact，不声称加密成功。
func (d *PIIDetector) getReplacement(value string, piiType PIIType, mode RedactionMode) string {
	switch mode {
	case RedactionModeRedact:
		return "***REDACTED***"
	case RedactionModeHash:
		return d.hashValue(value)
	case RedactionModeEncrypt:
		encrypted, err := d.encryptValue(value)
		if err != nil {
			// 缺密钥时降级为 Redact，不声称加密成功
			return "***REDACTED***"
		}
		return encrypted
	case RedactionModeRemove:
		return ""
	default:
		return "***REDACTED***"
	}
}

// hashValue 计算 SHA256(value + salt)，返回 hash:hex 格式
func (d *PIIDetector) hashValue(value string) string {
	h := sha256.New()
	h.Write([]byte(value))
	if d.hashSalt != "" {
		h.Write([]byte(d.hashSalt))
	}
	return "hash:" + hex.EncodeToString(h.Sum(nil))
}

// encryptValue 使用 AES-256-GCM 加密，返回 enc:hex 格式
func (d *PIIDetector) encryptValue(value string) (string, error) {
	if len(d.encryptKey) == 0 {
		return "", fmt.Errorf("encryption key not configured")
	}

	block, err := aes.NewCipher(d.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return "enc:" + hex.EncodeToString(ciphertext), nil
}

// DecryptValue 解密 enc:hex 格式的密文，返回原始值
func (d *PIIDetector) DecryptValue(encrypted string) (string, error) {
	if len(d.encryptKey) == 0 {
		return "", fmt.Errorf("encryption key not configured")
	}

	if !strings.HasPrefix(encrypted, "enc:") {
		return "", fmt.Errorf("not an encrypted value (missing enc: prefix)")
	}

	data, err := hex.DecodeString(encrypted[4:])
	if err != nil {
		return "", fmt.Errorf("hex decode failed: %w", err)
	}

	block, err := aes.NewCipher(d.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed (possibly tampered): %w", err)
	}

	return string(plaintext), nil
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
