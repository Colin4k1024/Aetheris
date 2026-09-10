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

package vision

import (
	"context"
	"fmt"
	"strings"
)

// Client 视觉模型接口。
//
// 当前状态：接口已定义，OpenAIVisionClient 已实现真实视觉适配。
// StubClient 仅供测试使用，不应在生产环境注册。
type Client interface {
	// Describe 描述图像内容
	Describe(ctx context.Context, imageURLOrBase64 string) (string, error)
	// Name 返回模型名称
	Name() string
}

// StubClient 是一个测试占位实现，所有调用返回固定值。
// 仅限测试使用；生产环境必须使用真实 provider 如 OpenAIVisionClient。
type StubClient struct{}

// Describe 返回占位文本
func (s *StubClient) Describe(ctx context.Context, imageURLOrBase64 string) (string, error) {
	return "vision stub: not implemented", nil
}

// Name 返回 "stub"
func (s *StubClient) Name() string {
	return "stub"
}

// Config holds the configuration for creating a vision client.
type Config struct {
	Provider string // "openai" (future: "claude", "gemini", etc.)
	APIKey   string
	BaseURL  string
	Model    string
}

// NewClientFromConfig creates a vision client based on the provider type.
// Returns an error for unknown or unconfigured providers.
func NewClientFromConfig(config Config) (Client, error) {
	switch strings.ToLower(config.Provider) {
	case "openai":
		return NewOpenAIVisionClient(config.Model, config.APIKey, config.BaseURL)
	case "stub":
		return &StubClient{}, nil
	default:
		return nil, fmt.Errorf("unknown vision provider: %s", config.Provider)
	}
}
