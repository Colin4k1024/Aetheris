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
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
	"github.com/google/go-github/v45/github"
)

// GitHubConfig GitHub 工具配置
type GitHubConfig struct {
	// Token GitHub Personal Access Token
	Token string
	// BaseURL GitHub API 基础 URL，默认为 https://api.github.com
	BaseURL string
	// Timeout 请求超时时间
	Timeout time.Duration
}

// GitHubTool GitHub API MCP 工具
// 提供对 GitHub 仓库、Issues、Pull Requests 的访问能力
type GitHubTool struct {
	client *github.Client
	config *GitHubConfig
}

// NewGitHubTool 创建 GitHub 工具实例
func NewGitHubTool(config *GitHubConfig) *GitHubTool {
	if config == nil {
		config = &GitHubConfig{}
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.github.com"
	}

	var tc *http.Client
	if config.Token != "" {
		tc = &http.Client{
			Transport: &bearerTransport{token: config.Token},
		}
	}

	client := github.NewClient(tc)
	if config.BaseURL != "https://api.github.com" {
		client.BaseURL, _ = client.BaseURL.Parse(config.BaseURL)
	}

	return &GitHubTool{
		client: client,
		config: config,
	}
}

// bearerTransport is an http.RoundTripper that adds a Bearer token to requests.
type bearerTransport struct {
	token     string
	transport http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	transport := t.transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(r)
}

// Name 返回工具名称
func (t *GitHubTool) Name() string {
	return "mcp-github"
}

// Description 返回工具描述
func (t *GitHubTool) Description() string {
	return "GitHub API tool for accessing repositories, issues, and pull requests"
}

// Schema 返回参数 Schema
func (t *GitHubTool) Schema() tool.Schema {
	return tool.Schema{
		Type: "object",
		Properties: map[string]tool.SchemaProperty{
			"action": {
				Type:        "string",
				Description: "Action to perform: search_repos, get_issue, create_issue, list_pulls, get_file, create_pr",
			},
			"query": {
				Type:        "string",
				Description: "Search query for search_repos action",
			},
			"owner": {
				Type:        "string",
				Description: "Repository owner (for issue/PR operations)",
			},
			"repo": {
				Type:        "string",
				Description: "Repository name",
			},
			"issue_number": {
				Type:        "integer",
				Description: "Issue or PR number",
			},
			"title": {
				Type:        "string",
				Description: "Issue or PR title (for create operations)",
			},
			"body": {
				Type:        "string",
				Description: "Issue or PR body content",
			},
			"state": {
				Type:        "string",
				Description: "Issue or PR state: open, closed, all",
			},
			"path": {
				Type:        "string",
				Description: "File path (for get_file action)",
			},
			"ref": {
				Type:        "string",
				Description: "Git reference (branch, tag, commit)",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of results to return",
			},
		},
		Required: []string{"action"},
	}
}

// Execute 执行 GitHub 操作
func (t *GitHubTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
	action, _ := input["action"].(string)
	if action == "" {
		return tool.ToolResult{Err: "action is required"}, nil
	}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	var result string
	var err error

	switch action {
	case "search_repos":
		result, err = t.searchRepos(ctx, input)
	case "get_issue":
		result, err = t.getIssue(ctx, input)
	case "create_issue":
		result, err = t.createIssue(ctx, input)
	case "list_pulls":
		result, err = t.listPulls(ctx, input)
	case "get_file":
		result, err = t.getFile(ctx, input)
	case "create_pr":
		result, err = t.createPR(ctx, input)
	default:
		return tool.ToolResult{Err: fmt.Sprintf("unknown action: %s", action)}, nil
	}

	if err != nil {
		return tool.ToolResult{Err: err.Error()}, nil
	}

	return tool.ToolResult{Content: result}, nil
}

// searchRepos 搜索仓库
func (t *GitHubTool) searchRepos(ctx context.Context, input map[string]any) (string, error) {
	query, _ := input["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required for search_repos")
	}

	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: limit},
	}

	repos, _, err := t.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	result := make([]map[string]any, 0, len(repos.Repositories))
	for _, repo := range repos.Repositories {
		result = append(result, map[string]any{
			"name":        repo.GetName(),
			"full_name":   repo.GetFullName(),
			"description": repo.GetDescription(),
			"url":         repo.GetHTMLURL(),
			"stars":       repo.GetStargazersCount(),
			"language":    repo.GetLanguage(),
			"updated_at":  repo.GetUpdatedAt().Format(time.RFC3339),
		})
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// getIssue 获取 Issue
func (t *GitHubTool) getIssue(ctx context.Context, input map[string]any) (string, error) {
	owner, _ := input["owner"].(string)
	repo, _ := input["repo"].(string)
	issueNumber := int64(0)
	if n, ok := input["issue_number"].(float64); ok {
		issueNumber = int64(n)
	}

	if owner == "" || repo == "" || issueNumber == 0 {
		return "", fmt.Errorf("owner, repo, and issue_number are required")
	}

	issue, _, err := t.client.Issues.Get(ctx, owner, repo, int(issueNumber))
	if err != nil {
		return "", fmt.Errorf("get issue failed: %w", err)
	}

	result := map[string]any{
		"number":   issue.GetNumber(),
		"title":    issue.GetTitle(),
		"body":     issue.GetBody(),
		"state":    issue.GetState(),
		"url":      issue.GetHTMLURL(),
		"user":     issue.GetUser().GetLogin(),
		"labels":   issue.Labels,
		"created":  issue.CreatedAt.Format(time.RFC3339),
		"updated":  issue.UpdatedAt.Format(time.RFC3339),
		"comments": issue.GetComments(),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// createIssue 创建 Issue
func (t *GitHubTool) createIssue(ctx context.Context, input map[string]any) (string, error) {
	owner, _ := input["owner"].(string)
	repo, _ := input["repo"].(string)
	title, _ := input["title"].(string)
	body, _ := input["body"].(string)

	if owner == "" || repo == "" || title == "" {
		return "", fmt.Errorf("owner, repo, and title are required")
	}

	issue, _, err := t.client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
		Title: &title,
		Body:  &body,
	})
	if err != nil {
		return "", fmt.Errorf("create issue failed: %w", err)
	}

	result := map[string]any{
		"number": issue.GetNumber(),
		"title":  issue.GetTitle(),
		"url":    issue.GetHTMLURL(),
		"state":  issue.GetState(),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// listPulls 列出 Pull Requests
func (t *GitHubTool) listPulls(ctx context.Context, input map[string]any) (string, error) {
	owner, _ := input["owner"].(string)
	repo, _ := input["repo"].(string)
	state := "open"
	if s, ok := input["state"].(string); ok {
		state = s
	}

	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	if owner == "" || repo == "" {
		return "", fmt.Errorf("owner and repo are required")
	}

	opts := &github.PullRequestListOptions{
		State:       state,
		ListOptions: github.ListOptions{PerPage: limit},
	}

	prs, _, err := t.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return "", fmt.Errorf("list pulls failed: %w", err)
	}

	result := make([]map[string]any, 0, len(prs))
	for _, pr := range prs {
		result = append(result, map[string]any{
			"number":    pr.GetNumber(),
			"title":     pr.GetTitle(),
			"state":     pr.GetState(),
			"url":       pr.GetHTMLURL(),
			"user":      pr.GetUser().GetLogin(),
			"head":      pr.GetHead().GetRef(),
			"base":      pr.GetBase().GetRef(),
			"created":   pr.CreatedAt.Format(time.RFC3339),
			"updated":   pr.UpdatedAt.Format(time.RFC3339),
			"mergeable": pr.GetMergeable(),
		})
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// getFile 获取文件内容
func (t *GitHubTool) getFile(ctx context.Context, input map[string]any) (string, error) {
	owner, _ := input["owner"].(string)
	repo, _ := input["repo"].(string)
	path, _ := input["path"].(string)
	ref := "main"
	if r, ok := input["ref"].(string); ok {
		ref = r
	}

	if owner == "" || repo == "" || path == "" {
		return "", fmt.Errorf("owner, repo, and path are required")
	}

	fileContent, _, _, err := t.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", fmt.Errorf("get file failed: %w", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("decode file content failed: %w", err)
	}

	result := map[string]any{
		"path":    path,
		"content": content,
		"ref":     ref,
		"sha":     fileContent.GetSHA(),
		"size":    fileContent.GetSize(),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// createPR 创建 Pull Request
func (t *GitHubTool) createPR(ctx context.Context, input map[string]any) (string, error) {
	owner, _ := input["owner"].(string)
	repo, _ := input["repo"].(string)
	title, _ := input["title"].(string)
	body, _ := input["body"].(string)
	head := ""
	base := "main"

	if b, ok := input["base"].(string); ok {
		base = b
	}
	if h, ok := input["head"].(string); ok {
		head = h
	}

	if owner == "" || repo == "" || title == "" || head == "" {
		return "", fmt.Errorf("owner, repo, title, and head are required")
	}

	newPR := &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &head,
		Base:  &base,
	}

	pr, _, err := t.client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return "", fmt.Errorf("create pr failed: %w", err)
	}

	result := map[string]any{
		"number": pr.GetNumber(),
		"title":  pr.GetTitle(),
		"url":    pr.GetHTMLURL(),
		"state":  pr.GetState(),
		"head":   pr.GetHead().GetRef(),
		"base":   pr.GetBase().GetRef(),
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// EnvToken 从环境变量获取 GitHub Token
// Token 格式验证：GitHub PAT 通常为 40 字符的十六进制字符串（ghp_ 前缀或无前缀）
func EnvToken() string {
	token := os.Getenv("GITHUB_TOKEN")
	// 支持多种环境变量名
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	token = strings.TrimSpace(token)

	// Token 格式验证：检查基本格式合理性
	// GitHub Classic PAT 无前缀时为 40 字符十六进制，ghp_ 前缀时为 51 字符
	// gho_ (OAuth), ghs_, ghr_ 等也是有效前缀
	if token != "" {
		stripped := strings.TrimPrefix(token, "ghp_")
		stripped = strings.TrimPrefix(stripped, "gho_")
		stripped = strings.TrimPrefix(stripped, "ghs_")
		stripped = strings.TrimPrefix(stripped, "ghr_")
		stripped = strings.TrimPrefix(stripped, "github_pat_")

		// 验证是否为合理的十六进制字符串（长度应为 36-40）
		if len(stripped) < 36 || len(stripped) > 40 {
			// 长度不合理，可能不是有效的 GitHub Token
			return ""
		}
		for _, c := range stripped {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return ""
			}
		}
	}

	return token
}
