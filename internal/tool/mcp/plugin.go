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

package mcp

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

// PluginManifest 插件清单
type PluginManifest struct {
	Name        string   `json:"name" yaml:"name"`
	Version     string   `json:"version" yaml:"version"`
	Description string   `json:"description" yaml:"description"`
	Author      string   `json:"author" yaml:"author"`
	Tools       []string `json:"tools" yaml:"tools"`
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// Plugin MCP 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string
	// Version 返回插件版本
	Version() string
	// Description 返回插件描述
	Description() string
	// Register 注册工具到 registry
	Register(registry interface{}) error
	// Initialize 初始化插件
	Initialize(config map[string]any) error
}

// PluginLoader 插件加载器
type PluginLoader struct {
	pluginDir string
	loaded    map[string]Plugin
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(pluginDir string) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
		loaded:    make(map[string]Plugin),
	}
}

// LoadPlugin 从目录加载插件
func (l *PluginLoader) LoadPlugin(dir string) (Plugin, error) {
	// 读取 manifest
	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		manifestPath = filepath.Join(dir, "manifest.yaml")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest
	if strings.HasSuffix(manifestPath, ".yaml") {
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
	}

	// 加载插件代码（这里简化处理，实际应该是动态加载）
	plugin, err := l.loadFromManifest(manifest)
	if err != nil {
		return nil, err
	}

	l.loaded[manifest.Name] = plugin
	return plugin, nil
}

// LoadAll 加载目录下所有插件
func (l *PluginLoader) LoadAll() ([]Plugin, error) {
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin dir: %w", err)
	}

	var plugins []Plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(l.pluginDir, entry.Name())
		plugin, err := l.LoadPlugin(dir)
		if err != nil {
			// 记录错误但继续加载其他插件
			fmt.Printf("failed to load plugin %s: %v\n", entry.Name(), err)
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func (l *PluginLoader) loadFromManifest(manifest PluginManifest) (Plugin, error) {
	// 这里应该实现动态加载
	// 简化版本：返回一个包装器
	return &staticPlugin{
		name:        manifest.Name,
		version:     manifest.Version,
		description: manifest.Description,
		tools:       manifest.Tools,
	}, nil
}

// staticPlugin 静态插件实现（示例）
type staticPlugin struct {
	name        string
	version     string
	description string
	tools       []string
}

func (p *staticPlugin) Name() string        { return p.name }
func (p *staticPlugin) Version() string     { return p.version }
func (p *staticPlugin) Description() string { return p.description }

func (p *staticPlugin) Register(registry interface{}) error {
	// 注册工具
	return nil
}

func (p *staticPlugin) Initialize(config map[string]any) error {
	return nil
}

// CreatePluginTemplate 创建插件模板
func CreatePluginTemplate(name, author string) PluginManifest {
	return PluginManifest{
		Name:        name,
		Version:     "1.0.0",
		Description: "A new MCP plugin",
		Author:      author,
		Tools:       []string{},
	}
}

// WritePluginTemplate 写入插件模板文件
func WritePluginTemplate(dir, name, author string) error {
	manifest := CreatePluginTemplate(name, author)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// 创建示例工具文件
	exampleTool := `package main

import (
	"context"
	"rag-platform/internal/tool"
)

// ExampleTool 示例工具
type ExampleTool struct{}

func NewExampleTool() *ExampleTool {
	return &ExampleTool{}
}

func (t *ExampleTool) Name() string {
	return "example.action"
}

func (t *ExampleTool) Description() string {
	return "An example tool"
}

func (t *ExampleTool) Schema() tool.Schema {
	return tool.Schema{
		Type:        "object",
		Description: "Input parameters",
		Properties: map[string]tool.SchemaProperty{
			"input": {Type: "string", Description: "Input value"},
		},
		Required: []string{"input"},
	}
}

func (t *ExampleTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	inputVal, _ := input["input"].(string)
	return tool.ToolResult{Content: "processed: " + inputVal}, nil
}
`

	if err := os.WriteFile(filepath.Join(dir, "tool.go"), []byte(exampleTool), 0644); err != nil {
		return fmt.Errorf("failed to write tool.go: %w", err)
	}

	return nil
}
