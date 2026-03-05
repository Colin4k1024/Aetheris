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

// Package signer provides API request signing functionality.
package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
)

const (
	// HeaderSignature 签名 Header 名称
	HeaderSignature = "X-Signature"
	// HeaderTimestamp 时间戳 Header 名称
	HeaderTimestamp = "X-Timestamp"
	// HeaderAccessKey Access Key Header 名称
	HeaderAccessKey = "X-Access-Key"
	// HeaderSignedHeaders 已签名的 Header 列表
	HeaderSignedHeaders = "X-Signed-Headers"
)

// Algorithm 签名算法
type Algorithm string

const (
	AlgorithmHMACSHA256 Algorithm = "hmac-sha256"
	AlgorithmHMACSHA512 Algorithm = "hmac-sha512"
)

// Signer API 请求签名器
type Signer struct {
	secretKey     []byte
	algorithm     Algorithm
	clockSkew     time.Duration
	signedHeaders []string
}

// Option 配置选项
type Option func(*Signer)

// WithAlgorithm 设置签名算法
func WithAlgorithm(algo Algorithm) Option {
	return func(s *Signer) {
		s.algorithm = algo
	}
}

// WithClockSkew 设置允许的时间偏差
func WithClockSkew(skew time.Duration) Option {
	return func(s *Signer) {
		s.clockSkew = skew
	}
}

// WithSignedHeaders 设置需要签名的 Header
func WithSignedHeaders(headers []string) Option {
	return func(s *Signer) {
		s.signedHeaders = headers
	}
}

// New 创建签名器
func New(secretKey string, opts ...Option) *Signer {
	s := &Signer{
		secretKey:     []byte(secretKey),
		algorithm:     AlgorithmHMACSHA256,
		clockSkew:     5 * time.Minute,
		signedHeaders: []string{"host", "content-type"},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SignRequest 对请求进行签名
// method: HTTP 方法 (GET, POST, etc.)
// path: 请求路径
// timestamp: 时间戳 (Unix 秒)
// body: 请求体
func (s *Signer) SignRequest(method, path, timestamp, body string) string {
	stringToSign := s.buildStringToSign(method, path, timestamp, body)
	return s.sign(stringToSign)
}

// SignRequestWithHeaders 对请求进行签名（包含额外 Header）
func (s *Signer) SignRequestWithHeaders(method, path, timestamp, body string, headers map[string]string) string {
	stringToSign := s.buildStringToSignWithHeaders(method, path, timestamp, body, headers)
	return s.sign(stringToSign)
}

// buildStringToSign 构建待签名字符串
func (s *Signer) buildStringToSign(method, path, timestamp, body string) string {
	parts := []string{
		strings.ToUpper(method),
		path,
		timestamp,
		body,
	}
	return strings.Join(parts, "\n")
}

// buildStringToSignWithHeaders 构建待签名字符串（包含 Header）
func (s *Signer) buildStringToSignWithHeaders(method, path, timestamp, body string, headers map[string]string) string {
	var signedHeaders []string
	for _, header := range s.signedHeaders {
		if value, ok := headers[strings.ToLower(header)]; ok {
			signedHeaders = append(signedHeaders, fmt.Sprintf("%s:%s", strings.ToLower(header), value))
		}
	}

	parts := []string{
		strings.ToUpper(method),
		path,
		timestamp,
		strings.Join(signedHeaders, "\n"),
		body,
	}
	return strings.Join(parts, "\n")
}

// sign 生成签名
func (s *Signer) sign(stringToSign string) string {
	var h hash.Hash

	switch s.algorithm {
	case AlgorithmHMACSHA512:
		h = hmac.New(sha512.New, s.secretKey)
	default:
		h = hmac.New(sha256.New, s.secretKey)
	}

	h.Write([]byte(stringToSign))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证请求签名
func (s *Signer) VerifySignature(method, path, timestamp, body, signature string) error {
	// 验证时间戳
	if err := s.verifyTimestamp(timestamp); err != nil {
		return err
	}

	// 计算期望的签名
	expected := s.SignRequest(method, path, timestamp, body)

	// 比较签名
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// VerifySignatureWithHeaders 验证请求签名（包含 Header）
func (s *Signer) VerifySignatureWithHeaders(method, path, timestamp, body string, headers map[string]string, signature string) error {
	// 验证时间戳
	if err := s.verifyTimestamp(timestamp); err != nil {
		return err
	}

	// 计算期望的签名
	expected := s.SignRequestWithHeaders(method, path, timestamp, body, headers)

	// 比较签名
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// verifyTimestamp 验证时间戳是否在允许范围内
func (s *Signer) verifyTimestamp(timestamp string) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format")
	}

	now := time.Now().Unix()
	diff := now - ts
	if diff < 0 {
		diff = -diff
	}

	if diff > int64(s.clockSkew.Seconds()) {
		return fmt.Errorf("timestamp expired or too far in future")
	}

	return nil
}

// GenerateTimestamp 生成时间戳
func (s *Signer) GenerateTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// GetSignedHeaders 获取需要签名的 Header 列表
func (s *Signer) GetSignedHeaders() string {
	return strings.Join(s.signedHeaders, ",")
}

// Config 签名配置
type Config struct {
	SecretKey     string
	Algorithm     Algorithm
	ClockSkew     time.Duration
	SignedHeaders []string
}

// NewFromConfig 从配置创建签名器
func NewFromConfig(cfg Config) *Signer {
	opts := []Option{
		WithAlgorithm(cfg.Algorithm),
		WithClockSkew(cfg.ClockSkew),
	}
	if len(cfg.SignedHeaders) > 0 {
		opts = append(opts, WithSignedHeaders(cfg.SignedHeaders))
	}
	return New(cfg.SecretKey, opts...)
}
