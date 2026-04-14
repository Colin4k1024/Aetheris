// Copyright 2026 Aetheris Team
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
)

// FilesystemConfig 文件系统工具配置
type FilesystemConfig struct {
	// RootDir 允许访问的根目录
	RootDir string
	// AllowedPaths 允许访问的路径列表（为空则允许 RootDir 下所有路径）
	AllowedPaths []string
	// MaxFileSize 最大文件大小（字节）
	MaxFileSize int64
	// Timeout 操作超时时间
	Timeout time.Duration
}

// FilesystemTool 文件系统 MCP 工具
// 提供安全的本地文件系统操作能力
type FilesystemTool struct {
	config *FilesystemConfig
}

// NewFilesystemTool 创建文件系统工具实例
func NewFilesystemTool(config *FilesystemConfig) *FilesystemTool {
	if config == nil {
		config = &FilesystemConfig{}
	}
	if config.RootDir == "" {
		config.RootDir = "/tmp"
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &FilesystemTool{config: config}
}

// Name 返回工具名称
func (t *FilesystemTool) Name() string {
	return "mcp-filesystem"
}

// Description 返回工具描述
func (t *FilesystemTool) Description() string {
	return "Filesystem operations tool for reading, writing, and searching files"
}

// Schema 返回参数 Schema
func (t *FilesystemTool) Schema() tool.Schema {
	return tool.Schema{
		Type: "object",
		Properties: map[string]tool.SchemaProperty{
			"action": {
				Type:        "string",
				Description: "Action to perform: read_file, write_file, list_dir, search_files, delete_file, make_dir, exists",
			},
			"path": {
				Type:        "string",
				Description: "File or directory path",
			},
			"content": {
				Type:        "string",
				Description: "File content (for write_file action)",
			},
			"pattern": {
				Type:        "string",
				Description: "Search pattern (for search_files action, supports glob pattern)",
			},
			"recursive": {
				Type:        "boolean",
				Description: "Search recursively (for search_files action)",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of results to return",
			},
			"encoding": {
				Type:        "string",
				Description: "File encoding (default: utf-8)",
			},
		},
		Required: []string{"action", "path"},
	}
}

// Execute 执行文件系统操作
func (t *FilesystemTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	action, _ := input["action"].(string)
	path, _ := input["path"].(string)

	if action == "" {
		return tool.ToolResult{Err: "action is required"}, nil
	}
	if path == "" {
		return tool.ToolResult{Err: "path is required"}, nil
	}

	// 验证路径安全性
	if err := t.validatePath(path); err != nil {
		return tool.ToolResult{Err: err.Error()}, nil
	}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	var result string
	var err error

	switch action {
	case "read_file":
		result, err = t.readFile(ctx, input)
	case "write_file":
		result, err = t.writeFile(ctx, input)
	case "list_dir":
		result, err = t.listDir(ctx, input)
	case "search_files":
		result, err = t.searchFiles(ctx, input)
	case "delete_file":
		result, err = t.deleteFile(ctx, input)
	case "make_dir":
		result, err = t.makeDir(ctx, input)
	case "exists":
		result, err = t.exists(ctx, input)
	default:
		return tool.ToolResult{Err: fmt.Sprintf("unknown action: %s", action)}, nil
	}

	if err != nil {
		return tool.ToolResult{Err: err.Error()}, nil
	}

	return tool.ToolResult{Content: result}, nil
}

// validatePath 验证路径安全性，防止目录遍历攻击
func (t *FilesystemTool) validatePath(path string) error {
	// 解析绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// 确保路径在允许的目录内
	rootDir, _ := filepath.Abs(t.config.RootDir)
	if !strings.HasPrefix(absPath, rootDir) {
		// 检查是否在允许的路径列表中
		allowed := false
		for _, allowedPath := range t.config.AllowedPaths {
			allowedAbs, _ := filepath.Abs(allowedPath)
			if strings.HasPrefix(absPath, allowedAbs) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path outside allowed directory: %s", path)
		}
	}

	return nil
}

// readFile 读取文件
func (t *FilesystemTool) readFile(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)
	encoding := "utf-8"
	if e, ok := input["encoding"].(string); ok {
		encoding = e
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("stat file failed: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", path)
	}

	if info.Size() > t.config.MaxFileSize {
		return "", fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), t.config.MaxFileSize)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file failed: %w", err)
	}

	if encoding != "utf-8" && encoding != "utf8" {
		return "", fmt.Errorf("unsupported encoding: %s (only utf-8 supported)", encoding)
	}

	result := map[string]any{
		"path":     path,
		"content":  string(content),
		"size":     info.Size(),
		"modified": info.ModTime().Format(time.RFC3339),
		"encoding": encoding,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// writeFile 写入文件
func (t *FilesystemTool) writeFile(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)
	content, _ := input["content"].(string)

	// 检查父目录是否存在
	parentDir := filepath.Dir(path)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "", fmt.Errorf("create parent directory failed: %w", err)
		}
	}

	// 写入文件
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write file failed: %w", err)
	}

	info, _ := os.Stat(path)
	result := map[string]any{
		"path":     path,
		"size":     info.Size(),
		"written":  len(content),
		"modified": info.ModTime().Format(time.RFC3339),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// listDir 列出目录
func (t *FilesystemTool) listDir(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", path)
		}
		return "", fmt.Errorf("stat directory failed: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	limit := 100
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read directory failed: %w", err)
	}

	var items []map[string]any
	count := 0
	for _, entry := range entries {
		if count >= limit {
			break
		}

		entryInfo, _ := entry.Info()
		item := map[string]any{
			"name":       entry.Name(),
			"is_dir":     entry.IsDir(),
			"is_symlink": entry.Type()&os.ModeSymlink != 0,
		}

		if entryInfo != nil {
			item["size"] = entryInfo.Size()
			item["modified"] = entryInfo.ModTime().Format(time.RFC3339)
		}

		items = append(items, item)
		count++
	}

	result := map[string]any{
		"path":  path,
		"total": len(entries),
		"items": items,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// searchFiles 搜索文件
func (t *FilesystemTool) searchFiles(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)
	pattern, _ := input["pattern"].(string)
	recursive, _ := input["recursive"].(bool)

	if pattern == "" {
		return "", errors.New("pattern is required for search_files")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", path)
		}
		return "", fmt.Errorf("stat directory failed: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", path)
	}

	limit := 100
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	var matches []map[string]any

	var walkFn filepath.WalkFunc = func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过权限错误
		}

		if len(matches) >= limit {
			return filepath.SkipDir
		}

		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return nil
		}

		if matched {
			relPath, _ := filepath.Rel(path, filePath)
			matches = append(matches, map[string]any{
				"path":     filePath,
				"relative": relPath,
				"name":     info.Name(),
				"is_dir":   info.IsDir(),
				"size":     info.Size(),
				"modified": info.ModTime().Format(time.RFC3339),
			})
		}

		return nil
	}

	if recursive {
		err = filepath.Walk(path, walkFn)
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return "", fmt.Errorf("read directory failed: %w", err)
		}
		for _, entry := range entries {
			entryInfo, _ := entry.Info()
			if entryInfo != nil {
				walkFn(filepath.Join(path, entry.Name()), entryInfo, nil)
			}
		}
		err = nil
	}

	if err != nil {
		return "", fmt.Errorf("search files failed: %w", err)
	}

	result := map[string]any{
		"path":    path,
		"pattern": pattern,
		"total":   len(matches),
		"matches": matches,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// deleteFile 删除文件或目录
func (t *FilesystemTool) deleteFile(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path not found: %s", path)
		}
		return "", fmt.Errorf("stat path failed: %w", err)
	}

	var err2 error
	if info.IsDir() {
		err2 = os.RemoveAll(path)
	} else {
		err2 = os.Remove(path)
	}

	if err2 != nil {
		return "", fmt.Errorf("delete failed: %w", err2)
	}

	result := map[string]any{
		"path":    path,
		"type":    map[bool]string{true: "directory", false: "file"}[info.IsDir()],
		"deleted": true,
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// makeDir 创建目录
func (t *FilesystemTool) makeDir(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)
	recursive := true
	if r, ok := input["recursive"].(bool); ok {
		recursive = r
	}

	var err error
	if recursive {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}

	if err != nil {
		return "", fmt.Errorf("create directory failed: %w", err)
	}

	info, _ := os.Stat(path)
	result := map[string]any{
		"path":     path,
		"created":  true,
		"is_dir":   info.IsDir(),
		"modified": info.ModTime().Format(time.RFC3339),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// exists 检查路径是否存在
func (t *FilesystemTool) exists(ctx context.Context, input map[string]any) (string, error) {
	path, _ := input["path"].(string)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result := map[string]any{
				"path":    path,
				"exists":  false,
				"is_dir":  false,
				"is_file": false,
			}
			data, _ := json.Marshal(result)
			return string(data), nil
		}
		return "", fmt.Errorf("stat path failed: %w", err)
	}

	result := map[string]any{
		"path":     path,
		"exists":   true,
		"is_dir":   info.IsDir(),
		"is_file":  !info.IsDir(),
		"size":     info.Size(),
		"modified": info.ModTime().Format(time.RFC3339),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}
