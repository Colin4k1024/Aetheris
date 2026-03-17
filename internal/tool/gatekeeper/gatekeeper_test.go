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

package gatekeeper

import (
	"testing"

	"rag-platform/internal/tool"
	"rag-platform/internal/tool/types"
)

func TestValidateRequired(t *testing.T) {
	g := New()

	schema := tool.Schema{
		Required: []string{"method", "url"},
		Properties: map[string]tool.SchemaProperty{
			"method": {Type: "string"},
			"url":    {Type: "string"},
		},
	}

	// 正常情况
	err := g.Validate("http.request", map[string]any{"method": "GET", "url": "https://example.com"}, schema)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 缺少必需参数
	err = g.Validate("http.request", map[string]any{"method": "GET"}, schema)
	if err == nil {
		t.Error("expected error for missing required parameter")
	}
	if _, ok := err.(*types.MissingParameterError); !ok {
		t.Errorf("expected MissingParameterError, got %T", err)
	}
}

func TestValidateTypes(t *testing.T) {
	g := New(WithTypeValidation(true))

	schema := tool.Schema{
		Properties: map[string]tool.SchemaProperty{
			"method":  {Type: "string"},
			"status":  {Type: "boolean"},
			"count":   {Type: "integer"},
			"headers": {Type: "object"},
		},
	}

	// 正常类型
	err := g.Validate("test", map[string]any{
		"method":  "GET",
		"status":   true,
		"count":    float64(42),
		"headers":  map[string]any{},
	}, schema)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 错误类型
	err = g.Validate("test", map[string]any{
		"method": 123, // 应该是 string
	}, schema)
	if err == nil {
		t.Error("expected error for wrong type")
	}
	if _, ok := err.(*types.InvalidTypeError); !ok {
		t.Errorf("expected InvalidTypeError, got %T", err)
	}
}

func TestValidateHTTPRequest(t *testing.T) {
	g := New(
		WithAllowedHosts([]string{"example.com", "api.example.com"}),
		WithBlockedPatterns([]string{"*.internal", "192.168.0.0/16"}),
	)

	// 正常 URL
	err := g.validateHTTPRequest(map[string]any{"url": "https://example.com/api"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 不在白名单
	err = g.validateHTTPRequest(map[string]any{"url": "https://evil.com/api"})
	if err == nil {
		t.Error("expected error for non-allowed host")
	}

	// 匹配黑名单
	err = g.validateHTTPRequest(map[string]any{"url": "https://api.internal.com"})
	if err == nil {
		t.Error("expected error for blocked pattern")
	}
}

func TestValidateFileRead(t *testing.T) {
	g := New()

	// 正常路径
	err := g.validateFileRead(map[string]any{"path": "/home/user/file.txt"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 路径遍历
	err = g.validateFileRead(map[string]any{"path": "/home/user/../../etc/passwd"})
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestValidateFileWrite(t *testing.T) {
	g := New()

	// 正常路径
	err := g.validateFileWrite(map[string]any{"path": "/home/user/file.txt"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 危险路径
	err = g.validateFileWrite(map[string]any{"path": "/etc/passwd"})
	if err == nil {
		t.Error("expected error for system directory")
	}
}

func TestValidateDBQuery(t *testing.T) {
	g := New()

	// 正常查询
	err := g.validateDBQuery(map[string]any{"query": "SELECT * FROM users WHERE id = 1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 危险查询
	err = g.validateDBQuery(map[string]any{"query": "DROP TABLE users"})
	if err == nil {
		t.Error("expected error for dangerous SQL")
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"https://api.example.com/v1", "api.example.com"},
		{"http://localhost:8080/api", "localhost:8080"},
		{"https://example.com:443/path?query=1", "example.com:443"},
	}

	for _, tt := range tests {
		result := extractHost(tt.url)
		if result != tt.expected {
			t.Errorf("extractHost(%s) = %s; want %s", tt.url, result, tt.expected)
		}
	}
}
