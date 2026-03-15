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

// Package eino provides eino integration for Aetheris
package eino

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ContextMode represents how skill content is processed
type ContextMode string

const (
	// ContextModeInline - skill content returned directly as tool result
	ContextModeInline ContextMode = "inline"
	// ContextModeFork - create new agent with copied history
	ContextModeFork ContextMode = "fork"
	// ContextModeIsolate - create new agent with isolated context
	ContextModeIsolate ContextMode = "isolate"
)

// FrontMatter represents the metadata in SKILL.md yaml frontmatter
type FrontMatter struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Context     ContextMode `yaml:"context"`
	Agent       string      `yaml:"agent"`
	Model       string      `yaml:"model"`
}

// Skill represents a complete skill with metadata and content
type Skill struct {
	FrontMatter
	Content       string
	BaseDirectory string
}

// SkillBackend defines the interface for skill storage
type SkillBackend interface {
	List(ctx context.Context) ([]FrontMatter, error)
	Get(ctx context.Context, name string) (Skill, error)
}

// FileSystemBackend defines the interface for file system operations
type FileSystemBackend interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	ReadDir(ctx context.Context, path string) ([]os.DirEntry, error)
	IsDir(ctx context.Context, path string) (bool, error)
}

// LocalFileBackendConfig is the configuration for local file system backend
type LocalFileBackendConfig struct {
	BaseDir string
}

// localFileBackend implements FileSystemBackend using local filesystem
type localFileBackend struct {
	baseDir string
}

// NewLocalFileBackend creates a new local filesystem backend
func NewLocalFileBackend(ctx context.Context, config *LocalFileBackendConfig) (FileSystemBackend, error) {
	if config == nil || config.BaseDir == "" {
		return nil, fmt.Errorf("BaseDir is required")
	}
	return &localFileBackend{baseDir: config.BaseDir}, nil
}

func (b *localFileBackend) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (b *localFileBackend) ReadDir(ctx context.Context, path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (b *localFileBackend) IsDir(ctx context.Context, path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// SkillBackendFromFilesystemConfig is the configuration for filesystem-based skill backend
type SkillBackendFromFilesystemConfig struct {
	Backend FileSystemBackend
	BaseDir string
}

// filesystemSkillBackend implements SkillBackend using filesystem
type filesystemSkillBackend struct {
	backend FileSystemBackend
	baseDir string
}

// NewSkillBackendFromFilesystem creates a new skill backend from filesystem
func NewSkillBackendFromFilesystem(ctx context.Context, backend FileSystemBackend, baseDir string) (SkillBackend, error) {
	if backend == nil {
		return nil, fmt.Errorf("Backend is required")
	}
	if baseDir == "" {
		return nil, fmt.Errorf("BaseDir is required")
	}
	return &filesystemSkillBackend{
		backend: backend,
		baseDir: baseDir,
	}, nil
}

// List returns all available skills (only frontmatter metadata)
func (b *filesystemSkillBackend) List(ctx context.Context) ([]FrontMatter, error) {
	entries, err := b.backend.ReadDir(ctx, b.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []FrontMatter
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillDir := filepath.Join(b.baseDir, skillName)
		skillFile := filepath.Join(skillDir, "SKILL.md")

		isDir, err := b.backend.IsDir(ctx, skillDir)
		if err != nil || !isDir {
			continue
		}

		// Try to read SKILL.md
		content, err := b.backend.ReadFile(ctx, skillFile)
		if err != nil {
			continue // Skip directories without SKILL.md
		}

		frontmatter, err := parseSkillFrontMatter(string(content))
		if err != nil {
			continue // Skip invalid skill files
		}

		skills = append(skills, frontmatter)
	}

	return skills, nil
}

// Get returns a specific skill by name
func (b *filesystemSkillBackend) Get(ctx context.Context, name string) (Skill, error) {
	skillDir := filepath.Join(b.baseDir, name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	// Check if directory exists
	isDir, err := b.backend.IsDir(ctx, skillDir)
	if err != nil {
		return Skill{}, fmt.Errorf("skill %q not found: %w", name, err)
	}
	if !isDir {
		return Skill{}, fmt.Errorf("skill %q not found", name)
	}

	// Read SKILL.md
	content, err := b.backend.ReadFile(ctx, skillFile)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to read SKILL.md for skill %q: %w", name, err)
	}

	frontmatter, body, err := parseSkillContent(string(content))
	if err != nil {
		return Skill{}, fmt.Errorf("failed to parse SKILL.md for skill %q: %w", name, err)
	}

	// Set default context mode if not specified
	if frontmatter.Context == "" {
		frontmatter.Context = ContextModeInline
	}

	return Skill{
		FrontMatter:   frontmatter,
		Content:       body,
		BaseDirectory: skillDir,
	}, nil
}

// parseSkillFrontMatter parses only the frontmatter from SKILL.md content
func parseSkillFrontMatter(content string) (FrontMatter, error) {
	frontmatter, _, err := parseSkillContent(content)
	return frontmatter, err
}

// parseSkillContent parses frontmatter and body from SKILL.md content
func parseSkillContent(content string) (FrontMatter, string, error) {
	// Check for frontmatter delimiters
	if !strings.HasPrefix(content, "---") {
		return FrontMatter{}, "", fmt.Errorf("invalid SKILL.md: missing frontmatter delimiter")
	}

	// Find the closing delimiter
	lines := strings.Split(content, "\n")
	var end int
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") {
			end = i
			break
		}
	}

	if end == 0 {
		return FrontMatter{}, "", fmt.Errorf("invalid SKILL.md: missing closing frontmatter delimiter")
	}

	// Parse YAML frontmatter
	frontmatterYAML := strings.Join(lines[1:end], "\n")
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
		return FrontMatter{}, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Validate required fields
	if fm.Name == "" {
		return FrontMatter{}, "", fmt.Errorf("invalid SKILL.md: name is required in frontmatter")
	}
	if fm.Description == "" {
		return FrontMatter{}, "", fmt.Errorf("invalid SKILL.md: description is required in frontmatter")
	}

	// Get body content (everything after frontmatter)
	body := strings.TrimSpace(strings.Join(lines[end+1:], "\n"))

	return fm, body, nil
}
