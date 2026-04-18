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
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/tool"
)

// WebSearchConfig 网页搜索工具配置
type WebSearchConfig struct {
	// APIKey 搜索 API Key
	APIKey string
	// Engine 搜索引擎: google, bing, duckduckgo
	Engine string
	// Timeout 请求超时时间
	Timeout time.Duration
	// UserAgent HTTP User-Agent
	UserAgent string
}

// SearchEngine 接口，支持不同的搜索服务
type SearchEngine interface {
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	GetContent(ctx context.Context, url string) (string, error)
}

// SearchResult 搜索结果
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at,omitempty"`
}

// webSearchTool 网页搜索 MCP 工具
type webSearchTool struct {
	engine SearchEngine
	config *WebSearchConfig
	client *http.Client
}

// NewWebSearchTool 创建网页搜索工具实例
func NewWebSearchTool(config *WebSearchConfig) *webSearchTool {
	if config == nil {
		config = &WebSearchConfig{}
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Engine == "" {
		config.Engine = "duckduckgo" // 默认使用 DuckDuckGo
	}
	if config.UserAgent == "" {
		config.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	}

	tool := &webSearchTool{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// 根据引擎创建搜索实现
	switch config.Engine {
	case "duckduckgo":
		tool.engine = newDuckDuckGoEngine(tool.client, config.UserAgent)
	case "bing":
		tool.engine = newBingEngine(tool.client, config.APIKey)
	case "google":
		tool.engine = newGoogleEngine(tool.client, config.APIKey)
	default:
		tool.engine = newDuckDuckGoEngine(tool.client, config.UserAgent)
	}

	return tool
}

// Name 返回工具名称
func (t *webSearchTool) Name() string {
	return "mcp-web-search"
}

// Description 返回工具描述
func (t *webSearchTool) Description() string {
	return "Web search and content extraction tool"
}

// Schema 返回参数 Schema
func (t *webSearchTool) Schema() tool.Schema {
	return tool.Schema{
		Type: "object",
		Properties: map[string]tool.SchemaProperty{
			"action": {
				Type:        "string",
				Description: "Action to perform: search, get_content",
			},
			"query": {
				Type:        "string",
				Description: "Search query (for search action)",
			},
			"url": {
				Type:        "string",
				Description: "URL to fetch content from (for get_content action)",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of search results (default: 10)",
			},
		},
		Required: []string{"action"},
	}
}

// Execute 执行网页搜索操作
func (t *webSearchTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
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
	case "search":
		result, err = t.search(ctx, input)
	case "get_content":
		result, err = t.getContent(ctx, input)
	default:
		return tool.ToolResult{Err: fmt.Sprintf("unknown action: %s", action)}, nil
	}

	if err != nil {
		return tool.ToolResult{Err: err.Error()}, nil
	}

	return tool.ToolResult{Content: result}, nil
}

// search 执行搜索
func (t *webSearchTool) search(ctx context.Context, input map[string]any) (string, error) {
	query, _ := input["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required for search action")
	}

	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	results, err := t.engine.Search(ctx, query, limit)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	output := map[string]any{
		"query":   query,
		"engine":  t.config.Engine,
		"total":   len(results),
		"results": results,
	}

	data, _ := json.Marshal(output)
	return string(data), nil
}

// getContent 提取页面内容
func (t *webSearchTool) getContent(ctx context.Context, input map[string]any) (string, error) {
	targetURL, _ := input["url"].(string)
	if targetURL == "" {
		return "", fmt.Errorf("url is required for get_content action")
	}

	content, err := t.engine.GetContent(ctx, targetURL)
	if err != nil {
		return "", fmt.Errorf("get content failed: %w", err)
	}

	output := map[string]any{
		"url":     targetURL,
		"content": content,
		"length":  len(content),
	}

	data, _ := json.Marshal(output)
	return string(data), nil
}

// === DuckDuckGo Engine ===

type duckDuckGoEngine struct {
	client    *http.Client
	userAgent string
}

func newDuckDuckGoEngine(client *http.Client, userAgent string) SearchEngine {
	return &duckDuckGoEngine{
		client:    client,
		userAgent: userAgent,
	}
}

func (e *duckDuckGoEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// DuckDuckGo HTML search
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}

	return parseDuckDuckGoResults(string(body), limit)
}

// maxBodySize is the maximum response body size to read (10MB)
const maxBodySize = 10 * 1024 * 1024

// validateURLScheme validates that the URL uses an allowed scheme (http or https)
func validateURLScheme(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("invalid URL scheme %q: only http and https are allowed", scheme)
	}
	return nil
}

func (e *duckDuckGoEngine) GetContent(ctx context.Context, targetURL string) (string, error) {
	// Validate URL scheme to prevent SSRF attacks
	if err := validateURLScheme(targetURL); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", err
	}

	// 简单提取文本内容（实际应用中建议使用 goquery）
	content := extractText(string(body))
	return content, nil
}

func parseDuckDuckGoResults(html string, limit int) ([]SearchResult, error) {
	// 解析 DuckDuckGo HTML 结果
	// 格式: <a class="result__a" href="...">Title</a> ... <a class="result__snippet" href="...">Description</a>

	var results []SearchResult

	// 匹配结果条目
	resultPattern := regexp.MustCompile(`<a class="result__a" href="([^"]+)">([^<]+)</a>`)
	snippetPattern := regexp.MustCompile(`<a class="result__snippet"[^>]*>([^<]+)</a>`)

	resultMatches := resultPattern.FindAllStringSubmatch(html, -1)
	snippetMatches := snippetPattern.FindAllStringSubmatch(html, -1)

	count := 0
	for i, match := range resultMatches {
		if count >= limit {
			break
		}

		result := SearchResult{
			URL:    match[1],
			Title:  strings.TrimSpace(match[2]),
			Source: "duckduckgo",
		}

		if i < len(snippetMatches) {
			result.Description = strings.TrimSpace(snippetMatches[i][1])
			// 清理 HTML 标签
			result.Description = strings.ReplaceAll(result.Description, "<em>", "")
			result.Description = strings.ReplaceAll(result.Description, "</em>", "")
		}

		results = append(results, result)
		count++
	}

	return results, nil
}

// === Bing Engine ===

type bingEngine struct {
	client *http.Client
	apiKey string
}

func newBingEngine(client *http.Client, apiKey string) SearchEngine {
	return &bingEngine{
		client: client,
		apiKey: apiKey,
	}
}

func (e *bingEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing search failed with status: %d", resp.StatusCode)
	}

	var bingResp struct {
		WebPages struct {
			Value []struct {
				Name          string `json:"name"`
				URL           string `json:"url"`
				Snippet       string `json:"snippet"`
				DatePublished string `json:"datePublished"`
			} `json:"value"`
		} `json:"webPages"`
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err := json.Unmarshal(body, &bingResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(bingResp.WebPages.Value))
	for i, item := range bingResp.WebPages.Value {
		if i >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:       item.Name,
			URL:         item.URL,
			Description: item.Snippet,
			Source:      "bing",
			PublishedAt: item.DatePublished,
		})
	}

	return results, nil
}

func (e *bingEngine) GetContent(ctx context.Context, targetURL string) (string, error) {
	// Validate URL scheme to prevent SSRF attacks
	if err := validateURLScheme(targetURL); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", err
	}
	return extractText(string(body)), nil
}

// === Google Engine ===

type googleEngine struct {
	client *http.Client
	apiKey string
}

func newGoogleEngine(client *http.Client, apiKey string) SearchEngine {
	return &googleEngine{
		client: client,
		apiKey: apiKey,
	}
}

func (e *googleEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// 使用 Google Custom Search API
	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&q=%s&num=%d",
		e.apiKey, url.QueryEscape(query), limit,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google search failed with status: %d", resp.StatusCode)
	}

	var googleResp struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(googleResp.Items))
	for _, item := range googleResp.Items {
		results = append(results, SearchResult{
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Snippet,
			Source:      "google",
		})
	}

	return results, nil
}

func (e *googleEngine) GetContent(ctx context.Context, targetURL string) (string, error) {
	// Validate URL scheme to prevent SSRF attacks
	if err := validateURLScheme(targetURL); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", err
	}
	return extractText(string(body)), nil
}

// === Helper Functions ===

// extractText 从 HTML 中提取纯文本
func extractText(html string) string {
	// 移除 script 和 style 标签
	scriptPattern := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptPattern.ReplaceAllString(html, "")

	stylePattern := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = stylePattern.ReplaceAllString(html, "")

	// 替换常见标签为换行
	tagPattern := regexp.MustCompile(`(?is)<br[^>]*>`)
	html = tagPattern.ReplaceAllString(html, "\n")

	pPattern := regexp.MustCompile(`(?is)</p>`)
	html = pPattern.ReplaceAllString(html, "\n\n")

	divPattern := regexp.MustCompile(`(?is)</div>`)
	html = divPattern.ReplaceAllString(html, "\n")

	// 移除所有 HTML 标签
	tagRemovePattern := regexp.MustCompile(`(?is)<[^>]+>`)
	text := tagRemovePattern.ReplaceAllString(html, "")

	// 解码 HTML 实体
	text = htmlDecode(text)

	// 清理空白
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n\n")
}

// htmlDecode 解码 HTML 实体
func htmlDecode(s string) string {
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}

// EnvAPIKey 从环境变量获取 API Key
func EnvAPIKey() string {
	return os.Getenv("SEARCH_API_KEY")
}
