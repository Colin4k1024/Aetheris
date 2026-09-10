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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
	"github.com/Colin4k1024/Aetheris/v2/internal/tool/registry"
)

// ToolDef defines a single tool within a plugin manifest.
type ToolDef struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Type        string                 `json:"type" yaml:"type"` // "inline" or "command"
	Command     string                 `json:"command,omitempty" yaml:"command,omitempty"`
	Properties  map[string]PropertyDef `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required    []string               `json:"required,omitempty" yaml:"required,omitempty"`
	// HandlerKey identifies a registered inline handler (registered via RegisterInlineHandler)
	HandlerKey string `json:"handler_key,omitempty" yaml:"handler_key,omitempty"`
}

// PropertyDef is a simplified schema property for manifest-level tool definitions.
type PropertyDef struct {
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PluginManifest 插件清单
type PluginManifest struct {
	Name         string    `json:"name" yaml:"name"`
	Version      string    `json:"version" yaml:"version"`
	Description  string    `json:"description" yaml:"description"`
	Author       string    `json:"author" yaml:"author"`
	Tools        []ToolDef `json:"tools" yaml:"tools"`
	Dependencies []string  `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// Plugin MCP 插件接口
type Plugin interface {
	Name() string
	Version() string
	Description() string
	Register(reg *registry.Registry) error
	Initialize(config map[string]any) error
	Close() error
}

// InlineHandler is a function that implements a tool's Execute logic.
type InlineHandler func(ctx context.Context, input map[string]any) (string, error)

// PluginLoader 插件加载器
type PluginLoader struct {
	pluginDir string
	loaded    map[string]Plugin
	mu        sync.Mutex
	handlers  map[string]InlineHandler // handlerKey → handler
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(pluginDir string) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
		loaded:    make(map[string]Plugin),
		handlers:  make(map[string]InlineHandler),
	}
}

// RegisterInlineHandler registers an inline handler for tools with handler_key.
func (l *PluginLoader) RegisterInlineHandler(key string, handler InlineHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers[key] = handler
}

// LoadPlugin 从目录加载插件
func (l *PluginLoader) LoadPlugin(dir string) (Plugin, error) {
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

	// Validate manifest
	if err := validateManifest(manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	// Check for duplicate name
	l.mu.Lock()
	if _, exists := l.loaded[manifest.Name]; exists {
		l.mu.Unlock()
		return nil, fmt.Errorf("plugin %q already loaded", manifest.Name)
	}

	// Check dependencies are satisfied
	for _, dep := range manifest.Dependencies {
		if _, ok := l.loaded[dep]; !ok {
			l.mu.Unlock()
			return nil, fmt.Errorf("dependency %q not loaded; load it first", dep)
		}
	}

	// Get handlers snapshot
	handlers := make(map[string]InlineHandler)
	for k, v := range l.handlers {
		handlers[k] = v
	}
	l.mu.Unlock()

	plugin, err := l.loadFromManifest(manifest, handlers)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.loaded[manifest.Name] = plugin
	l.mu.Unlock()

	return plugin, nil
}

// LoadAll 加载目录下所有插件
func (l *PluginLoader) LoadAll() ([]Plugin, error) {
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin dir: %w", err)
	}

	var plugins []Plugin
	var loadErrors []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(l.pluginDir, entry.Name())
		plugin, err := l.LoadPlugin(dir)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("plugin %s: %w", entry.Name(), err))
			continue
		}
		plugins = append(plugins, plugin)
	}

	if len(loadErrors) > 0 && len(plugins) == 0 {
		return nil, fmt.Errorf("all plugins failed to load: %v", loadErrors)
	}

	return plugins, nil
}

// GetLoaded returns a previously loaded plugin by name.
func (l *PluginLoader) GetLoaded(name string) (Plugin, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.loaded[name]
	return p, ok
}

// CloseAll closes all loaded plugins and releases resources.
func (l *PluginLoader) CloseAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var errs []error
	for name, p := range l.loaded {
		if err := p.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", name, err))
		}
	}
	l.loaded = make(map[string]Plugin)
	if len(errs) > 0 {
		return fmt.Errorf("errors closing plugins: %v", errs)
	}
	return nil
}

// validateManifest checks required fields and tool name conflicts.
func validateManifest(m PluginManifest) error {
	if m.Name == "" {
		return fmt.Errorf("manifest name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest version is required")
	}
	seen := make(map[string]bool)
	for _, td := range m.Tools {
		if td.Name == "" {
			return fmt.Errorf("tool name is required")
		}
		if seen[td.Name] {
			return fmt.Errorf("duplicate tool name %q in manifest", td.Name)
		}
		seen[td.Name] = true
		if td.Type == "" {
			return fmt.Errorf("tool %q type is required", td.Name)
		}
		if td.Type != "inline" && td.Type != "command" {
			return fmt.Errorf("tool %q has unsupported type %q (use inline or command)", td.Name, td.Type)
		}
		if td.Type == "command" && td.Command == "" {
			return fmt.Errorf("tool %q of type command requires a command", td.Name)
		}
	}
	return nil
}

func (l *PluginLoader) loadFromManifest(manifest PluginManifest, handlers map[string]InlineHandler) (Plugin, error) {
	p := &manifestPlugin{
		name:        manifest.Name,
		version:     manifest.Version,
		description: manifest.Description,
		tools:       make([]manifestTool, 0, len(manifest.Tools)),
	}

	for _, td := range manifest.Tools {
		mt := manifestTool{
			name:        td.Name,
			description: td.Description,
			schema:      buildSchema(td),
			toolType:    td.Type,
			command:     td.Command,
		}

		if td.Type == "inline" {
			if td.HandlerKey != "" {
				handler, ok := handlers[td.HandlerKey]
				if !ok {
					return nil, fmt.Errorf("tool %q references unregistered handler key %q", td.Name, td.HandlerKey)
				}
				mt.handler = handler
			} else {
				return nil, fmt.Errorf("tool %q of type inline requires handler_key", td.Name)
			}
		}

		p.tools = append(p.tools, mt)
	}

	return p, nil
}

func buildSchema(td ToolDef) tool.Schema {
	props := make(map[string]tool.SchemaProperty)
	for name, prop := range td.Properties {
		props[name] = tool.SchemaProperty{
			Type:        prop.Type,
			Description: prop.Description,
		}
	}
	return tool.Schema{
		Type:        "object",
		Description: td.Description,
		Properties:  props,
		Required:    td.Required,
	}
}

// manifestPlugin implements Plugin from a manifest definition.
type manifestPlugin struct {
	name        string
	version     string
	description string
	tools       []manifestTool
	initialized bool
	closed      bool
}

func (p *manifestPlugin) Name() string        { return p.name }
func (p *manifestPlugin) Version() string     { return p.version }
func (p *manifestPlugin) Description() string { return p.description }

func (p *manifestPlugin) Register(reg *registry.Registry) error {
	for _, mt := range p.tools {
		reg.Register(&mt)
	}
	return nil
}

func (p *manifestPlugin) Initialize(config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config is required for plugin %s", p.name)
	}
	// Validate that required configuration is present
	// Plugins can declare required config keys via "required_config" in the manifest's config section
	// For now, accept any non-nil config
	p.initialized = true
	return nil
}

func (p *manifestPlugin) Close() error {
	p.closed = true
	return nil
}

// manifestTool implements tool.Tool from a manifest tool definition.
type manifestTool struct {
	name        string
	description string
	schema      tool.Schema
	toolType    string
	command     string
	handler     InlineHandler
}

func (t *manifestTool) Name() string        { return t.name }
func (t *manifestTool) Description() string { return t.description }
func (t *manifestTool) Schema() tool.Schema { return t.schema }

func (t *manifestTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	switch t.toolType {
	case "inline":
		if t.handler == nil {
			return tool.ToolResult{}, fmt.Errorf("tool %s has no handler", t.name)
		}
		result, err := t.handler(ctx, input)
		if err != nil {
			return tool.ToolResult{Err: err.Error()}, err
		}
		return tool.ToolResult{Content: result}, nil
	case "command":
		return tool.ToolResult{}, fmt.Errorf("command-based tool execution not yet implemented for %s", t.name)
	default:
		return tool.ToolResult{}, fmt.Errorf("unknown tool type %s for %s", t.toolType, t.name)
	}
}

// CreatePluginTemplate 创建插件模板
func CreatePluginTemplate(name, author string) PluginManifest {
	return PluginManifest{
		Name:        name,
		Version:     "1.0.0",
		Description: "A new MCP plugin",
		Author:      author,
		Tools:       []ToolDef{},
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
	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
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
