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
)

// Client 视觉模型接口。
//
// 当前状态：接口已定义，StubClient 仅供测试使用。
// 生产环境需要实现具体的视觉模型 adapter（如 OpenAI Vision、Claude Vision 等）。
type Client interface {
	// Describe 描述图像内容
	Describe(ctx context.Context, imageURLOrBase64 string) (string, error)
	// Name 返回模型名称
	Name() string
}

// StubClient 是一个测试占位实现，所有调用返回固定值。
// 不要在生产环境使用。
type StubClient struct{}

// Describe 返回占位文本
func (s *StubClient) Describe(ctx context.Context, imageURLOrBase64 string) (string, error) {
	return "vision stub: not implemented", nil
}

// Name 返回 "stub"
func (s *StubClient) Name() string {
	return "stub"
}
