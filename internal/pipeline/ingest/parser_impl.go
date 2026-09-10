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

package ingest

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// --- MarkdownParser ---

// Parse implements markdown-to-plain-text extraction.
// It strips markdown syntax (headers, emphasis, links, images, code blocks,
// blockquotes, lists, horizontal rules) and normalizes whitespace while
// preserving the readable text content. Raw markdown mode can be requested
// via metadata["markdown_mode"] = "raw".
func (p *MarkdownParser) Parse(content string, metadata map[string]interface{}) (string, error) {
	if content == "" {
		return "", nil
	}

	// Check if raw mode is explicitly requested
	if metadata != nil {
		if mode, ok := metadata["markdown_mode"].(string); ok && mode == "raw" {
			return content, nil
		}
	}

	result := stripMarkdownSyntax(content)
	result = normalizeWhitespace(result)
	return strings.TrimSpace(result), nil
}

// stripMarkdownSyntax removes markdown formatting elements while preserving text content.
func stripMarkdownSyntax(content string) string {
	// Remove code blocks (```...```)
	codeBlockRe := regexp.MustCompile("(?s)```[\\s\\S]*?```")
	content = codeBlockRe.ReplaceAllStringFunc(content, func(match string) string {
		// Extract text inside code block (skip the ``` and optional language specifier)
		lines := strings.Split(match, "\n")
		if len(lines) > 2 {
			return strings.Join(lines[1:len(lines)-1], "\n")
		}
		return ""
	})

	// Remove inline code (`code`)
	inlineCodeRe := regexp.MustCompile("`([^`]+)`")
	content = inlineCodeRe.ReplaceAllString(content, "$1")

	// Remove headers (## Header → Header)
	headerRe := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	content = headerRe.ReplaceAllString(content, "$1")

	// Remove bold/italic markers (**text**, __text__, *text*, _text_)
	content = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(content, "$1")
	// Remove single underscore italic — use word boundary to avoid matching __ (already handled above)
	content = regexp.MustCompile(`\b_([^_]+)_\b`).ReplaceAllString(content, "$1")

	// Remove strikethrough (~~text~~)
	content = regexp.MustCompile(`~~([^~]+)~~`).ReplaceAllString(content, "$1")

	// Remove images (![alt](url) → alt)
	content = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(content, "$1")

	// Remove links ([text](url) → text)
	content = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(content, "$1")

	// Remove reference links ([text][ref])
	content = regexp.MustCompile(`\[([^\]]+)\]\[[^\]]*\]`).ReplaceAllString(content, "$1")

	// Remove blockquotes (> text → text)
	content = regexp.MustCompile(`(?m)^>\s*`).ReplaceAllString(content, "")

	// Remove horizontal rules (---, ***, ___)
	content = regexp.MustCompile(`(?m)^(-{3,}|\*{3,}|_{3,})$`).ReplaceAllString(content, "")

	// Remove list markers (-, *, +, 1.)
	content = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`).ReplaceAllString(content, "")

	// Remove HTML tags that might be in markdown
	content = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, "")

	return content
}

// normalizeWhitespace collapses multiple spaces and blank lines.
func normalizeWhitespace(s string) string {
	// Collapse multiple spaces into one (preserving newlines)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	// Collapse multiple blank lines
	result := strings.Join(lines, "\n")
	result = regexp.MustCompile("\n{3,}").ReplaceAllString(result, "\n\n")
	return result
}

// --- HTMLParser ---

// Parse extracts visible text content from HTML.
// It removes script, style, noscript, and comment elements, then extracts
// the text content of the remaining elements. Metadata can include
// "html_mode" = "raw" to return the original HTML without parsing.
func (p *HTMLParser) Parse(content string, metadata map[string]interface{}) (string, error) {
	if content == "" {
		return "", nil
	}

	// Check if raw mode is explicitly requested
	if metadata != nil {
		if mode, ok := metadata["html_mode"].(string); ok && mode == "raw" {
			return content, nil
		}
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var sb strings.Builder
	extractHTMLText(doc, &sb)

	result := normalizeWhitespace(sb.String())
	return strings.TrimSpace(result), nil
}

// extractHTMLText walks the HTML node tree, skipping script/style/noscript
// elements and collecting text from the remaining nodes.
func extractHTMLText(n *html.Node, sb *strings.Builder) {
	if n == nil {
		return
	}

	// Skip non-visible elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "noscript", "head", "meta", "link", "title":
			return
		}
	}

	// Skip comments
	if n.Type == html.CommentNode {
		return
	}

	// Collect text nodes
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			if sb.Len() > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(text)
		}
	}

	// Recurse into children
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		extractHTMLText(child, sb)
	}

	// Add newline after block elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "div", "p", "br", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6", "hr":
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
		}
	}
}

// --- JSONParser ---

// Parse validates JSON and optionally extracts specific fields to content.
// If metadata["json_content_field"] is set, that field's value becomes the content.
// If metadata["json_flatten"] is "true", the JSON is pretty-printed as content.
// Invalid JSON returns an error.
func (p *JSONParser) Parse(content string, metadata map[string]interface{}) (string, error) {
	if content == "" {
		return "", nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Check for configured content field extraction
	if metadata != nil {
		if field, ok := metadata["json_content_field"].(string); ok && field != "" {
			if obj, ok := data.(map[string]interface{}); ok {
				if val, exists := obj[field]; exists {
					return fmt.Sprintf("%v", val), nil
				}
			}
		}

		// Check for flatten mode
		if mode, ok := metadata["json_flatten"].(string); ok && mode == "true" {
			pretty, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return "", fmt.Errorf("marshal JSON: %w", err)
			}
			return string(pretty), nil
		}
	}

	// Default: return original content (validated)
	return content, nil
}

// Ensure compile-time interface compliance
var _ io.Reader = (*strings.Reader)(nil)
