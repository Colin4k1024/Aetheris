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

// Package audit provides audit log signing and verification.
package audit

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ed25519"
)

// Algorithm 签名算法
type Algorithm string

const (
	AlgorithmECDSA   Algorithm = "ecdsa"
	AlgorithmRSA     Algorithm = "rsa"
	AlgorithmEd25519 Algorithm = "ed25519"
)

// SigningKey 签名密钥
type SigningKey struct {
	Algorithm  Algorithm
	PrivateKey crypto.PrivateKey
	PublicKey  crypto.PublicKey
	KeyID      string // 密钥标识符
	CreatedAt  time.Time
}

// Signer 审计日志签名器
type Signer struct {
	key          *SigningKey
	previousHash string
}

// SignedAuditEntry 带签名的审计条目
type SignedAuditEntry struct {
	EntryID      string            `json:"entry_id"`
	TenantID     string            `json:"tenant_id"`
	UserID       string            `json:"user_id"`
	Action       string            `json:"action"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Success      bool              `json:"success"`
	DurationMS   int64             `json:"duration_ms"`
	Timestamp    time.Time         `json:"timestamp"`
	PreviousHash string            `json:"previous_hash"`
	Signature    string            `json:"signature"`
	KeyID        string            `json:"key_id"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Config 签名器配置
type Config struct {
	Algorithm         Algorithm // ecdsa, rsa, ed25519
	KeyFile           string    // 私钥文件路径
	KeyPassphrase     string    // 私钥密码（可选）
	GenerateIfMissing bool      // 密钥不存在时生成
}

// NewSigner 创建签名器
func NewSigner(cfg Config) (*Signer, error) {
	var key *SigningKey
	var err error

	if cfg.KeyFile != "" {
		key, err = loadKey(cfg.KeyFile, cfg.KeyPassphrase)
		if err != nil && !cfg.GenerateIfMissing {
			return nil, err
		}
	}

	// 如果没有密钥，生成新的
	if key == nil {
		key, err = generateKey(cfg.Algorithm)
		if err != nil {
			return nil, err
		}
	}

	return &Signer{
		key:          key,
		previousHash: "genesis",
	}, nil
}

// Sign 对审计条目签名
func (s *Signer) Sign(entry *SignedAuditEntry) error {
	// 生成唯一 ID
	entry.EntryID = generateEntryID()

	// 设置时间戳
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// 设置前一个哈希
	entry.PreviousHash = s.previousHash

	// 设置密钥 ID
	entry.KeyID = s.key.KeyID

	// 计算内容哈希
	content := entry.ContentToSign()
	hash := s.hashContent(content)

	// 签名
	signature, err := s.sign(hash)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	entry.Signature = signature

	// 更新 previousHash
	s.previousHash = hash

	return nil
}

// Verify 验证审计条目
func (s *Signer) Verify(entry *SignedAuditEntry) error {
	if entry.Signature == "" {
		return fmt.Errorf("empty signature")
	}

	// 验证密钥 ID
	if entry.KeyID != s.key.KeyID {
		return fmt.Errorf("key ID mismatch")
	}

	// 计算期望的签名
	content := entry.ContentToSign()
	hash := s.hashContent(content)

	// 验证签名
	return s.verify(hash, entry.Signature)
}

// VerifyChain 验证审计链的完整性
func (s *Signer) VerifyChain(entries []SignedAuditEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// 第一个条目的 previousHash 应该是 "genesis"
	if entries[0].PreviousHash != "genesis" {
		return fmt.Errorf("first entry must have genesis previous hash")
	}

	var previousHash string
	for i, entry := range entries {
		// 验证前一个哈希
		if i == 0 {
			if entry.PreviousHash != "genesis" {
				return fmt.Errorf("entry %d: expected genesis hash", i)
			}
		} else {
			if entry.PreviousHash != previousHash {
				return fmt.Errorf("entry %d: previous hash mismatch", i)
			}
		}

		// 验证签名
		content := entry.ContentToSign()
		hash := s.hashContent(content)
		if err := s.verify(hash, entry.Signature); err != nil {
			return fmt.Errorf("entry %d: %w", i, err)
		}

		previousHash = hash
	}

	return nil
}

// ContentToSign 获取待签名内容
func (e *SignedAuditEntry) ContentToSign() string {
	parts := []string{
		e.EntryID,
		e.TenantID,
		e.UserID,
		e.Action,
		e.ResourceType,
		e.ResourceID,
		fmt.Sprintf("%t", e.Success),
		fmt.Sprintf("%d", e.DurationMS),
		e.Timestamp.Format(time.RFC3339Nano),
		e.PreviousHash,
	}
	return strings.Join(parts, "|")
}

// hashContent 计算内容哈希
func (s *Signer) hashContent(content string) string {
	var hash []byte
	switch s.key.Algorithm {
	case AlgorithmRSA:
		h := sha512.Sum512([]byte(content))
		hash = h[:]
	default:
		h := sha256.Sum256([]byte(content))
		hash = h[:]
	}
	return base64.StdEncoding.EncodeToString(hash)
}

// sign 计算签名
func (s *Signer) sign(hash string) (string, error) {
	h := sha256.Sum256([]byte(hash))

	var sig []byte
	var err error

	switch key := s.key.PrivateKey.(type) {
	case *ecdsa.PrivateKey:
		sig, err = ecdsa.SignASN1(rand.Reader, key, h[:])
	case *rsa.PrivateKey:
		sig, err = rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	case ed25519.PrivateKey:
		sig = ed25519.Sign(key, h[:])
	default:
		return "", fmt.Errorf("unsupported key type")
	}

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// verify 验证签名
func (s *Signer) verify(hash, signature string) error {
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding")
	}

	h := sha256.Sum256([]byte(hash))

	switch key := s.key.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(key, h[:], sig) {
			return fmt.Errorf("signature verification failed")
		}
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, h[:], sig); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	case ed25519.PublicKey:
		if !ed25519.Verify(key, h[:], sig) {
			return fmt.Errorf("signature verification failed")
		}
	default:
		return fmt.Errorf("unsupported key type")
	}

	return nil
}

// generateKey 生成密钥
func generateKey(algo Algorithm) (*SigningKey, error) {
	var privateKey crypto.PrivateKey
	var publicKey crypto.PublicKey

	switch algo {
	case AlgorithmECDSA:
		pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		privateKey = pk
		publicKey = &pk.PublicKey
	case AlgorithmRSA:
		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		privateKey = pk
		publicKey = &pk.PublicKey
	case AlgorithmEd25519:
		pk, pub, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		privateKey = pk
		publicKey = pub
	default:
		// 默认使用 ECDSA
		pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		privateKey = pk
		publicKey = &pk.PublicKey
	}

	keyID := generateKeyID(publicKey)

	return &SigningKey{
		Algorithm:  algo,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      keyID,
		CreatedAt:  time.Now(),
	}, nil
}

// loadKey 从文件加载密钥
func loadKey(keyFile, passphrase string) (*SigningKey, error) {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM file")
	}

	var privateKey crypto.PrivateKey

	switch block.Type {
	case "EC PRIVATE KEY":
		pk, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		privateKey = pk
	case "RSA PRIVATE KEY":
		pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		privateKey = pk
	case "PRIVATE KEY":
		pk, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		privateKey = pk
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	var publicKey crypto.PublicKey
	switch k := privateKey.(type) {
	case *ecdsa.PrivateKey:
		publicKey = &k.PublicKey
	case *rsa.PrivateKey:
		publicKey = &k.PublicKey
	case ed25519.PrivateKey:
		publicKey = k.Public().(ed25519.PublicKey)
	}

	keyID := generateKeyID(publicKey)

	return &SigningKey{
		Algorithm:  getKeyAlgorithm(privateKey),
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      keyID,
		CreatedAt:  time.Now(),
	}, nil
}

// SaveKey 保存密钥到文件
func SaveKey(key *SigningKey, keyFile string) error {
	var block *pem.Block

	switch k := key.PrivateKey.(type) {
	case *ecdsa.PrivateKey:
		data, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return err
		}
		block = &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}
	case *rsa.PrivateKey:
		data := x509.MarshalPKCS1PrivateKey(k)
		block = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: data}
	case ed25519.PrivateKey:
		data, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return err
		}
		block = &pem.Block{Type: "PRIVATE KEY", Bytes: data}
	default:
		return fmt.Errorf("unsupported key type")
	}

	return os.WriteFile(keyFile, pem.EncodeToMemory(block), 0600)
}

// GetPublicKey 获取公钥
func (s *Signer) GetPublicKey() crypto.PublicKey {
	return s.key.PublicKey
}

// GetKeyID 获取密钥 ID
func (s *Signer) GetKeyID() string {
	return s.key.KeyID
}

func generateEntryID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func generateKeyID(publicKey crypto.PublicKey) string {
	data, _ := json.Marshal(publicKey)
	h := sha256.Sum256(data)
	return base64.URLEncoding.EncodeToString(h[:8])
}

func getKeyAlgorithm(privateKey crypto.PrivateKey) Algorithm {
	switch privateKey.(type) {
	case *ecdsa.PrivateKey:
		return AlgorithmECDSA
	case *rsa.PrivateKey:
		return AlgorithmRSA
	case ed25519.PrivateKey:
		return AlgorithmEd25519
	default:
		return AlgorithmECDSA
	}
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// MarshalJSON 实现 JSON 序列化
func (e *SignedAuditEntry) MarshalJSON() ([]byte, error) {
	type Alias SignedAuditEntry
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(e),
		Timestamp: e.Timestamp.Format(time.RFC3339),
	})
}

// UnmarshalJSON 实现 JSON 反序列化
func (e *SignedAuditEntry) UnmarshalJSON(data []byte) error {
	type Alias SignedAuditEntry
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Timestamp != "" {
		e.Timestamp, _ = time.Parse(time.RFC3339, aux.Timestamp)
	}
	return nil
}
