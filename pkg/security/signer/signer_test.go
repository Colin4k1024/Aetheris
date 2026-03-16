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

package signer

import (
	"strconv"
	"testing"
	"time"
)

func TestHeaderConstants(t *testing.T) {
	if HeaderSignature != "X-Signature" {
		t.Errorf("expected X-Signature, got %s", HeaderSignature)
	}
	if HeaderTimestamp != "X-Timestamp" {
		t.Errorf("expected X-Timestamp, got %s", HeaderTimestamp)
	}
	if HeaderAccessKey != "X-Access-Key" {
		t.Errorf("expected X-Access-Key, got %s", HeaderAccessKey)
	}
	if HeaderSignedHeaders != "X-Signed-Headers" {
		t.Errorf("expected X-Signed-Headers, got %s", HeaderSignedHeaders)
	}
}

func TestAlgorithmConstants(t *testing.T) {
	if AlgorithmHMACSHA256 != "hmac-sha256" {
		t.Errorf("expected hmac-sha256, got %s", AlgorithmHMACSHA256)
	}
	if AlgorithmHMACSHA512 != "hmac-sha512" {
		t.Errorf("expected hmac-sha512, got %s", AlgorithmHMACSHA512)
	}
}

func TestNew(t *testing.T) {
	signer := New("test-secret")
	if signer == nil {
		t.Fatal("expected non-nil signer")
	}
	if string(signer.secretKey) != "test-secret" {
		t.Errorf("expected test-secret, got %s", string(signer.secretKey))
	}
	if signer.algorithm != AlgorithmHMACSHA256 {
		t.Errorf("expected HMAC-SHA256, got %s", signer.algorithm)
	}
	if signer.clockSkew != 5*time.Minute {
		t.Errorf("expected 5 minutes, got %v", signer.clockSkew)
	}
}

func TestWithAlgorithm(t *testing.T) {
	opt := WithAlgorithm(AlgorithmHMACSHA512)
	signer := &Signer{
		secretKey: []byte("test"),
		algorithm: AlgorithmHMACSHA256,
	}
	opt(signer)
	if signer.algorithm != AlgorithmHMACSHA512 {
		t.Errorf("expected hmac-sha512, got %s", signer.algorithm)
	}
}

func TestWithClockSkew(t *testing.T) {
	opt := WithClockSkew(10 * time.Minute)
	signer := &Signer{
		secretKey: []byte("test"),
		clockSkew: 5 * time.Minute,
	}
	opt(signer)
	if signer.clockSkew != 10*time.Minute {
		t.Errorf("expected 10 minutes, got %v", signer.clockSkew)
	}
}

func TestWithSignedHeaders(t *testing.T) {
	headers := []string{"host", "content-type", "x-custom"}
	opt := WithSignedHeaders(headers)
	signer := &Signer{
		secretKey:     []byte("test"),
		signedHeaders: []string{"host"},
	}
	opt(signer)
	if len(signer.signedHeaders) != 3 {
		t.Errorf("expected 3 headers, got %d", len(signer.signedHeaders))
	}
}

func TestSignRequest(t *testing.T) {
	signer := New("test-secret")
	signature := signer.SignRequest("GET", "/api/test", "1234567890", "body")
	if signature == "" {
		t.Error("expected non-empty signature")
	}
}

func TestSignRequest_Deterministic(t *testing.T) {
	signer := New("test-secret")
	sig1 := signer.SignRequest("GET", "/api/test", "1234567890", "body")
	sig2 := signer.SignRequest("GET", "/api/test", "1234567890", "body")
	if sig1 != sig2 {
		t.Error("expected signatures to be deterministic")
	}
}

func TestSignRequest_DifferentInputs(t *testing.T) {
	signer := New("test-secret")
	sig1 := signer.SignRequest("GET", "/api/test", "1234567890", "body")
	sig2 := signer.SignRequest("POST", "/api/test", "1234567890", "body")
	if sig1 == sig2 {
		t.Error("expected different methods to produce different signatures")
	}
}

func TestSignRequest_DifferentBody(t *testing.T) {
	signer := New("test-secret")
	sig1 := signer.SignRequest("GET", "/api/test", "1234567890", "body1")
	sig2 := signer.SignRequest("GET", "/api/test", "1234567890", "body2")
	if sig1 == sig2 {
		t.Error("expected different bodies to produce different signatures")
	}
}

func TestSignRequestWithHeaders(t *testing.T) {
	signer := New("test-secret", WithSignedHeaders([]string{"host", "content-type"}))
	headers := map[string]string{
		"host":         "example.com",
		"content-type": "application/json",
	}
	signature := signer.SignRequestWithHeaders("GET", "/api/test", "1234567890", "body", headers)
	if signature == "" {
		t.Error("expected non-empty signature")
	}
}

func TestBuildStringToSign(t *testing.T) {
	signer := New("test-secret")
	result := signer.buildStringToSign("GET", "/api/test", "1234567890", "body")
	expected := "GET\n/api/test\n1234567890\nbody"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBuildStringToSignWithHeaders(t *testing.T) {
	signer := New("test-secret", WithSignedHeaders([]string{"host", "content-type"}))
	headers := map[string]string{
		"host":         "example.com",
		"content-type": "application/json",
	}
	result := signer.buildStringToSignWithHeaders("POST", "/api/test", "1234567890", "body", headers)
	// Should contain the method, path, timestamp, headers, and body
	if len(result) == 0 {
		t.Error("expected non-empty string")
	}
}

func TestSign(t *testing.T) {
	signer := New("test-secret")
	result := signer.sign("test-string-to-sign")
	if result == "" {
		t.Error("expected non-empty signature")
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	signer := New("test-secret")
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := signer.SignRequest("GET", "/api/test", timestamp, "body")
	err := signer.VerifySignature("GET", "/api/test", timestamp, "body", signature)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	signer := New("test-secret")
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	err := signer.VerifySignature("GET", "/api/test", timestamp, "body", "invalid-signature")
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestVerifySignature_ExpiredTimestamp(t *testing.T) {
	signer := New("test-secret", WithClockSkew(1*time.Second))
	// Use a timestamp from 10 seconds ago
	timestamp := strconv.FormatInt(time.Now().Unix()-10, 10)
	signature := signer.SignRequest("GET", "/api/test", timestamp, "body")
	err := signer.VerifySignature("GET", "/api/test", timestamp, "body", signature)
	if err == nil {
		t.Error("expected error for expired timestamp")
	}
}

func TestVerifySignatureWithHeaders_Valid(t *testing.T) {
	signer := New("test-secret", WithSignedHeaders([]string{"host", "content-type"}))
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	headers := map[string]string{
		"host":         "example.com",
		"content-type": "application/json",
	}
	signature := signer.SignRequestWithHeaders("GET", "/api/test", timestamp, "body", headers)
	err := signer.VerifySignatureWithHeaders("GET", "/api/test", timestamp, "body", headers, signature)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGenerateTimestamp(t *testing.T) {
	signer := New("test-secret")
	ts := signer.GenerateTimestamp()
	_, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		t.Errorf("expected valid timestamp, got %v", err)
	}
}

func TestGetSignedHeaders(t *testing.T) {
	signer := New("test-secret", WithSignedHeaders([]string{"host", "content-type"}))
	headers := signer.GetSignedHeaders()
	if headers != "host,content-type" {
		t.Errorf("expected host,content-type, got %s", headers)
	}
}

func TestNewFromConfig(t *testing.T) {
	cfg := Config{
		SecretKey:     "my-secret",
		Algorithm:     AlgorithmHMACSHA512,
		ClockSkew:     10 * time.Minute,
		SignedHeaders: []string{"host"},
	}
	signer := NewFromConfig(cfg)
	if string(signer.secretKey) != "my-secret" {
		t.Errorf("expected my-secret, got %s", string(signer.secretKey))
	}
	if signer.algorithm != AlgorithmHMACSHA512 {
		t.Errorf("expected hmac-sha512, got %s", signer.algorithm)
	}
	if signer.clockSkew != 10*time.Minute {
		t.Errorf("expected 10 minutes, got %v", signer.clockSkew)
	}
}
