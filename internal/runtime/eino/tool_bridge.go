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

package eino

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
)

// RuntimeTool 运行时工具接口（避免直接依赖 agent/tools 造成循环引用）
type RuntimeTool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error)
}

// RuntimeToolResult 工具执行结果
type RuntimeToolResult struct {
	Done   bool
	State  interface{}
	Output string
	Err    string
}

// RuntimeToolRegistry 运行时工具注册表接口
type RuntimeToolRegistry interface {
	List() []RuntimeTool
}

// RegistryToolBridge 将运行时工具注册表中的所有工具转为 Eino InvokableTool 列表，
// 使 Eino ADK Agent 能直接调用 Aetheris 注册的 Session 感知工具（含 Native + MCP）。
type RegistryToolBridge struct {
	registry RuntimeToolRegistry
}

// NewRegistryToolBridge 创建桥接器
func NewRegistryToolBridge(reg RuntimeToolRegistry) *RegistryToolBridge {
	return &RegistryToolBridge{registry: reg}
}

// EinoTools 将 Registry 中全部工具转为 Eino BaseTool 列表（供 ADK Agent ToolsConfig 使用）
func (b *RegistryToolBridge) EinoTools() []tool.BaseTool {
	if b.registry == nil {
		return nil
	}
	list := b.registry.List()
	out := make([]tool.BaseTool, 0, len(list))
	for _, t := range list {
		out = append(out, &registryToolAdapter{tool: t})
	}
	return out
}

// registryToolAdapter 将单个 RuntimeTool 适配为 eino tool.InvokableTool
type registryToolAdapter struct {
	tool RuntimeTool
}

var _ tool.InvokableTool = (*registryToolAdapter)(nil)

// Info 返回 Eino ToolInfo（将 Schema 映射为 Eino ParameterInfo）
func (a *registryToolAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	params := schemaMapToParams(a.tool.Schema())
	info := &schema.ToolInfo{
		Name: a.tool.Name(),
		Desc: a.tool.Description(),
	}
	if len(params) > 0 {
		info.ParamsOneOf = schema.NewParamsOneOfByParams(params)
	}
	return info, nil
}

// InvokableRun 执行工具：将 JSON 参数反序列化后调用 Tool.Execute
func (a *registryToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input map[string]any
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
			input = map[string]any{"raw": argumentsInJSON}
		}
	}
	if input == nil {
		input = make(map[string]any)
	}

	// 从 context 获取 Session（若有）
	sess := sessionFromContext(ctx)

	out, err := a.tool.Execute(ctx, sess, input, nil)
	if err != nil {
		return "", err
	}

	// 处理返回值
	switch v := out.(type) {
	case string:
		return v, nil
	case RuntimeToolResult:
		if v.Err != "" {
			return "", fmt.Errorf("%s", v.Err)
		}
		return v.Output, nil
	default:
		b, _ := json.Marshal(out)
		return string(b), nil
	}
}

// schemaMapToParams 将 map[string]any 格式的 Schema 转为 Eino ParameterInfo
func schemaMapToParams(s map[string]any) map[string]*schema.ParameterInfo {
	if len(s) == 0 {
		return nil
	}

	// 尝试解析 JSON Schema 格式 {"type":"object","properties":{...},"required":[...]}
	props, _ := s["properties"].(map[string]any)
	if props == nil {
		// 直接把顶层 key 当参数名
		params := make(map[string]*schema.ParameterInfo, len(s))
		for k, v := range s {
			params[k] = &schema.ParameterInfo{
				Type: schema.String,
				Desc: fmt.Sprint(v),
			}
		}
		return params
	}

	required := make(map[string]bool)
	if reqList, ok := s["required"].([]any); ok {
		for _, r := range reqList {
			if rs, ok := r.(string); ok {
				required[rs] = true
			}
		}
	}

	params := make(map[string]*schema.ParameterInfo, len(props))
	for name, propVal := range props {
		pi := &schema.ParameterInfo{
			Type:     schema.String,
			Required: required[name],
		}
		if pm, ok := propVal.(map[string]any); ok {
			if desc, ok := pm["description"].(string); ok {
				pi.Desc = desc
			}
			if t, ok := pm["type"].(string); ok {
				pi.Type = mapJSONTypeToEino(t)
			}
		}
		params[name] = pi
	}
	return params
}

func mapJSONTypeToEino(t string) schema.DataType {
	switch t {
	case "string":
		return schema.String
	case "integer", "int":
		return schema.Integer
	case "number", "float":
		return schema.Number
	case "boolean", "bool":
		return schema.Boolean
	case "array":
		return schema.Array
	case "object":
		return schema.Object
	default:
		return schema.String
	}
}

// --- Context-based Session passing ---

type sessionCtxKey struct{}

// WithSession 将 Session 注入 context，供工具桥接层取用
func WithSession(ctx context.Context, sess *session.Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, sess)
}

// sessionFromContext 从 context 取 Session（无则 nil）
func sessionFromContext(ctx context.Context) *session.Session {
	if v, ok := ctx.Value(sessionCtxKey{}).(*session.Session); ok {
		return v
	}
	return nil
}
