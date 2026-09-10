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

package embedding

import (
	"context"
	"hash/fnv"
)

// Embedder 向量化接口。生产实现调用真实 embedding provider（如 OpenAI）；
// 测试或 demo 使用 NewEmbedder 获得的 mockEmbedder，后者产生确定性非零向量。
type Embedder interface {
	// Embed 对文本做向量化，返回与 texts 一一对应的向量
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	// Model 返回模型名称
	Model() string
	// Dimension 返回向量维度
	Dimension() int
}

// embedderBase 提供 Model/Dimension 的公共实现
type embedderBase struct {
	model     string
	dimension int
}

func (e *embedderBase) Model() string {
	return e.model
}

func (e *embedderBase) Dimension() int {
	return e.dimension
}

// mockEmbedder 产生确定性非零向量，仅供测试和 demo 使用。
// 不同文本产生不同向量，同一文本产生相同向量，保证入库→查询可区分。
type mockEmbedder struct {
	embedderBase
}

// NewEmbedder 创建 mock Embedder（确定性非零向量，仅供测试/demo）。
// 生产路径应使用 NewOpenAIEmbedder 或 NewEmbedderFromConfig。
func NewEmbedder(model string, dimension int) Embedder {
	if dimension <= 0 {
		dimension = 1536
	}
	return &mockEmbedder{embedderBase{model: model, dimension: dimension}}
}

// Embed 产生确定性非零向量
func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if m == nil || len(texts) == 0 {
		return nil, nil
	}
	dim := m.dimension
	if dim <= 0 {
		dim = 1536
	}
	out := make([][]float64, len(texts))
	for i, text := range texts {
		out[i] = mockVector(text, dim)
	}
	return out, nil
}

// mockVector 根据文本生成确定性非零向量
func mockVector(text string, dim int) []float64 {
	v := make([]float64, dim)
	h := fnv.New32a()
	_, _ = h.Write([]byte(text))
	seed := uint32(h.Sum32())
	for i := 0; i < dim; i++ {
		seed = seed*1103515245 + 12345
		v[i] = float64(seed%1000) / 1000.0
	}
	// 保证至少一个非零值
	if v[0] == 0 {
		v[0] = 0.001
	}
	return v
}
