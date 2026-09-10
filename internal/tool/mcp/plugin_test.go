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
	"os"
	"path/filepath"
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool/registry"
)

// createTestPluginDir creates a temp dir with a plugin manifest.
func createTestPluginDir(t *testing.T, name string, manifest PluginManifest) string {
	t.Helper()
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, name)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), data, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return dir // parent dir containing the plugin subdirectory
}

func TestPluginLoader_LoadValidPlugin(t *testing.T) {
	manifest := PluginManifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{
				Name:        "echo",
				Description: "Echoes input",
				Type:        "inline",
				HandlerKey:  "echo_handler",
				Properties: map[string]PropertyDef{
					"input": {Type: "string", Description: "Value to echo"},
				},
				Required: []string{"input"},
			},
		},
	}

	parentDir := createTestPluginDir(t, "test-plugin", manifest)
	loader := NewPluginLoader(parentDir)
	loader.RegisterInlineHandler("echo_handler", func(ctx context.Context, input map[string]any) (string, error) {
		val := input["input"].(string)
		return "echo: " + val, nil
	})

	plugin, err := loader.LoadPlugin(filepath.Join(parentDir, "test-plugin"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plugin.Name() != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %s", plugin.Name())
	}
	if plugin.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", plugin.Version())
	}
}

func TestPluginLoader_RegisterToRegistry(t *testing.T) {
	manifest := PluginManifest{
		Name:    "reg-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{
				Name:        "reg.echo",
				Description: "Echoes input",
				Type:        "inline",
				HandlerKey:  "echo_handler",
				Properties: map[string]PropertyDef{
					"input": {Type: "string"},
				},
				Required: []string{"input"},
			},
		},
	}

	parentDir := createTestPluginDir(t, "reg-plugin", manifest)
	loader := NewPluginLoader(parentDir)
	loader.RegisterInlineHandler("echo_handler", func(ctx context.Context, input map[string]any) (string, error) {
		return "echo: " + input["input"].(string), nil
	})

	plugin, err := loader.LoadPlugin(filepath.Join(parentDir, "reg-plugin"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reg := registry.New()
	if err := plugin.Register(reg); err != nil {
		t.Fatalf("register error: %v", err)
	}

	tool, ok := reg.Get("reg.echo")
	if !ok {
		t.Fatal("expected tool 'reg.echo' to be registered")
	}
	if tool.Description() != "Echoes input" {
		t.Errorf("description mismatch: %s", tool.Description())
	}
}

func TestPluginLoader_ExecuteRegisteredTool(t *testing.T) {
	manifest := PluginManifest{
		Name:    "exec-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{
				Name:        "exec.uppercase",
				Description: "Converts to uppercase",
				Type:        "inline",
				HandlerKey:  "uppercase_handler",
				Properties: map[string]PropertyDef{
					"text": {Type: "string"},
				},
				Required: []string{"text"},
			},
		},
	}

	parentDir := createTestPluginDir(t, "exec-plugin", manifest)
	loader := NewPluginLoader(parentDir)
	loader.RegisterInlineHandler("uppercase_handler", func(ctx context.Context, input map[string]any) (string, error) {
		text := input["text"].(string)
		return "RESULT: " + text, nil
	})

	plugin, _ := loader.LoadPlugin(filepath.Join(parentDir, "exec-plugin"))
	reg := registry.New()
	plugin.Register(reg)

	tool, _ := reg.Get("exec.uppercase")
	result, err := tool.Execute(context.Background(), map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if result.Content != "RESULT: hello" {
		t.Errorf("expected 'RESULT: hello', got %s", result.Content)
	}
}

func TestPluginLoader_BadManifest_MissingName(t *testing.T) {
	manifest := PluginManifest{
		Version: "1.0.0",
		Tools:   []ToolDef{},
	}

	parentDir := createTestPluginDir(t, "bad-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "bad-plugin"))
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestPluginLoader_BadManifest_DuplicateToolName(t *testing.T) {
	manifest := PluginManifest{
		Name:    "dup-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{Name: "dup", Type: "inline", HandlerKey: "h1"},
			{Name: "dup", Type: "inline", HandlerKey: "h2"},
		},
	}

	parentDir := createTestPluginDir(t, "dup-plugin", manifest)
	loader := NewPluginLoader(parentDir)
	loader.RegisterInlineHandler("h1", func(ctx context.Context, input map[string]any) (string, error) { return "1", nil })
	loader.RegisterInlineHandler("h2", func(ctx context.Context, input map[string]any) (string, error) { return "2", nil })

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "dup-plugin"))
	if err == nil {
		t.Error("expected error for duplicate tool name")
	}
}

func TestPluginLoader_InlineToolMissingHandlerKey(t *testing.T) {
	manifest := PluginManifest{
		Name:    "no-handler-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{Name: "tool1", Type: "inline"}, // no handler_key
		},
	}

	parentDir := createTestPluginDir(t, "no-handler-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "no-handler-plugin"))
	if err == nil {
		t.Error("expected error for inline tool without handler_key")
	}
}

func TestPluginLoader_UnregisteredHandlerKey(t *testing.T) {
	manifest := PluginManifest{
		Name:    "unregistered-handler-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{Name: "tool1", Type: "inline", HandlerKey: "nonexistent"},
		},
	}

	parentDir := createTestPluginDir(t, "unregistered-handler-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "unregistered-handler-plugin"))
	if err == nil {
		t.Error("expected error for unregistered handler key")
	}
}

func TestPluginLoader_DuplicatePluginName(t *testing.T) {
	manifest := PluginManifest{
		Name:    "same-name",
		Version: "1.0.0",
		Tools:   []ToolDef{},
	}

	parentDir := createTestPluginDir(t, "same-name", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "same-name"))
	if err != nil {
		t.Fatalf("first load error: %v", err)
	}

	_, err = loader.LoadPlugin(filepath.Join(parentDir, "same-name"))
	if err == nil {
		t.Error("expected error for duplicate plugin name")
	}
}

func TestPluginLoader_DependencyNotLoaded(t *testing.T) {
	manifest := PluginManifest{
		Name:         "dep-plugin",
		Version:      "1.0.0",
		Dependencies: []string{"nonexistent-dep"},
		Tools:        []ToolDef{},
	}

	parentDir := createTestPluginDir(t, "dep-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "dep-plugin"))
	if err == nil {
		t.Error("expected error for unmet dependency")
	}
}

func TestPluginLoader_MissingManifestFile(t *testing.T) {
	dir := t.TempDir()
	loader := NewPluginLoader(dir)
	_, err := loader.LoadPlugin(filepath.Join(dir, "nonexistent"))
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestPluginLoader_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "bad-json")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte("{invalid json"), 0644)

	loader := NewPluginLoader(dir)
	_, err := loader.LoadPlugin(pluginDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPluginLoader_CommandToolNotImplemented(t *testing.T) {
	manifest := PluginManifest{
		Name:    "cmd-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{
				Name:    "cmd.tool",
				Type:    "command",
				Command: "echo hello",
			},
		},
	}

	parentDir := createTestPluginDir(t, "cmd-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	plugin, err := loader.LoadPlugin(filepath.Join(parentDir, "cmd-plugin"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reg := registry.New()
	plugin.Register(reg)

	tool, _ := reg.Get("cmd.tool")
	_, err = tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for unimplemented command tool")
	}
}

func TestPluginLoader_CommandToolMissingCommand(t *testing.T) {
	manifest := PluginManifest{
		Name:    "cmd-missing-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{Name: "cmd.tool", Type: "command"}, // no command
		},
	}

	parentDir := createTestPluginDir(t, "cmd-missing-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "cmd-missing-plugin"))
	if err == nil {
		t.Error("expected error for command tool without command")
	}
}

func TestPluginLoader_UnsupportedToolType(t *testing.T) {
	manifest := PluginManifest{
		Name:    "bad-type-plugin",
		Version: "1.0.0",
		Tools: []ToolDef{
			{Name: "tool1", Type: "magic"},
		},
	}

	parentDir := createTestPluginDir(t, "bad-type-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	_, err := loader.LoadPlugin(filepath.Join(parentDir, "bad-type-plugin"))
	if err == nil {
		t.Error("expected error for unsupported tool type")
	}
}

func TestPluginLoader_Initialize(t *testing.T) {
	manifest := PluginManifest{
		Name:    "init-plugin",
		Version: "1.0.0",
		Tools:   []ToolDef{},
	}

	parentDir := createTestPluginDir(t, "init-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	plugin, _ := loader.LoadPlugin(filepath.Join(parentDir, "init-plugin"))

	// Nil config should error
	err := plugin.Initialize(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}

	// Non-nil config should succeed
	err = plugin.Initialize(map[string]any{"key": "value"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPluginLoader_GetLoaded(t *testing.T) {
	manifest := PluginManifest{
		Name:    "get-plugin",
		Version: "1.0.0",
		Tools:   []ToolDef{},
	}

	parentDir := createTestPluginDir(t, "get-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	loader.LoadPlugin(filepath.Join(parentDir, "get-plugin"))

	p, ok := loader.GetLoaded("get-plugin")
	if !ok {
		t.Error("expected to find loaded plugin")
	}
	if p.Name() != "get-plugin" {
		t.Errorf("name mismatch: %s", p.Name())
	}

	_, ok = loader.GetLoaded("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent plugin")
	}
}

func TestPluginLoader_CloseAll(t *testing.T) {
	manifest := PluginManifest{
		Name:    "close-plugin",
		Version: "1.0.0",
		Tools:   []ToolDef{},
	}

	parentDir := createTestPluginDir(t, "close-plugin", manifest)
	loader := NewPluginLoader(parentDir)

	loader.LoadPlugin(filepath.Join(parentDir, "close-plugin"))

	if err := loader.CloseAll(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// After close, plugin should not be in loaded
	if _, ok := loader.GetLoaded("close-plugin"); ok {
		t.Error("expected plugin to be removed after CloseAll")
	}
}

func TestPluginLoader_LoadAll_MultiplePlugins(t *testing.T) {
	parentDir := t.TempDir()

	// Create two plugin directories
	for _, name := range []string{"plugin-a", "plugin-b"} {
		pluginDir := filepath.Join(parentDir, name)
		os.MkdirAll(pluginDir, 0755)
		manifest := PluginManifest{
			Name:    name,
			Version: "1.0.0",
			Tools:   []ToolDef{},
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		os.WriteFile(filepath.Join(pluginDir, "manifest.json"), data, 0644)
	}

	loader := NewPluginLoader(parentDir)
	plugins, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestPluginLoader_YAMLManifest(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "yaml-plugin")
	os.MkdirAll(pluginDir, 0755)

	yamlContent := `name: yaml-plugin
version: "1.0.0"
description: A YAML manifest plugin
tools: []
`
	os.WriteFile(filepath.Join(pluginDir, "manifest.yaml"), []byte(yamlContent), 0644)

	loader := NewPluginLoader(dir)
	plugin, err := loader.LoadPlugin(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plugin.Name() != "yaml-plugin" {
		t.Errorf("expected 'yaml-plugin', got %s", plugin.Name())
	}
	if plugin.Description() != "A YAML manifest plugin" {
		t.Errorf("description mismatch: %s", plugin.Description())
	}
}

func TestValidateManifest_EmptyName(t *testing.T) {
	m := PluginManifest{Version: "1.0.0"}
	if err := validateManifest(m); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidateManifest_EmptyVersion(t *testing.T) {
	m := PluginManifest{Name: "test"}
	if err := validateManifest(m); err == nil {
		t.Error("expected error for empty version")
	}
}

func TestBuildSchema(t *testing.T) {
	td := ToolDef{
		Name:        "test",
		Description: "A test tool",
		Properties: map[string]PropertyDef{
			"input":  {Type: "string", Description: "Input value"},
			"number": {Type: "integer", Description: "A number"},
		},
		Required: []string{"input"},
	}

	schema := buildSchema(td)
	if schema.Type != "object" {
		t.Errorf("expected type 'object', got %s", schema.Type)
	}
	if len(schema.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(schema.Properties))
	}
	if schema.Properties["input"].Type != "string" {
		t.Errorf("input type mismatch: %s", schema.Properties["input"].Type)
	}
	if len(schema.Required) != 1 || schema.Required[0] != "input" {
		t.Errorf("required mismatch: %v", schema.Required)
	}
}
