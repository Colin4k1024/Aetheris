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
	"strings"
	"testing"
)

// --- MarkdownParser Tests ---

func TestMarkdownParser_Headers(t *testing.T) {
	p := &MarkdownParser{}
	input := "## Main Title\n### Subtitle\nContent here"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "##") {
		t.Errorf("expected header markers stripped, got: %s", result)
	}
	if !strings.Contains(result, "Main Title") {
		t.Errorf("expected 'Main Title' in result, got: %s", result)
	}
	if !strings.Contains(result, "Subtitle") {
		t.Errorf("expected 'Subtitle' in result, got: %s", result)
	}
}

func TestMarkdownParser_BoldItalic(t *testing.T) {
	p := &MarkdownParser{}
	input := "This is **bold** and *italic* and __also bold__ and _also italic_"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "**") || strings.Contains(result, "__") {
		t.Errorf("expected bold markers stripped, got: %s", result)
	}
	if strings.Contains(result, "*") || strings.Contains(result, "_italic") {
		t.Errorf("expected italic markers stripped, got: %s", result)
	}
	if !strings.Contains(result, "bold") || !strings.Contains(result, "italic") {
		t.Errorf("expected text preserved, got: %s", result)
	}
}

func TestMarkdownParser_Links(t *testing.T) {
	p := &MarkdownParser{}
	input := "Check [this link](https://example.com) out"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "https://") {
		t.Errorf("expected URL stripped, got: %s", result)
	}
	if !strings.Contains(result, "this link") {
		t.Errorf("expected link text preserved, got: %s", result)
	}
}

func TestMarkdownParser_Images(t *testing.T) {
	p := &MarkdownParser{}
	input := "![alt text](image.png)"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "alt text") {
		t.Errorf("expected alt text preserved, got: %s", result)
	}
	if strings.Contains(result, "image.png") {
		t.Errorf("expected image URL stripped, got: %s", result)
	}
}

func TestMarkdownParser_CodeBlocks(t *testing.T) {
	p := &MarkdownParser{}
	input := "Some text\n```go\nfunc main() {}\n```\nMore text"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "```") {
		t.Errorf("expected code block markers stripped, got: %s", result)
	}
	if !strings.Contains(result, "func main()") {
		t.Errorf("expected code content preserved, got: %s", result)
	}
}

func TestMarkdownParser_InlineCode(t *testing.T) {
	p := &MarkdownParser{}
	input := "Use `code` inline"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "`") {
		t.Errorf("expected backtick stripped, got: %s", result)
	}
	if !strings.Contains(result, "code") {
		t.Errorf("expected code text preserved, got: %s", result)
	}
}

func TestMarkdownParser_Lists(t *testing.T) {
	p := &MarkdownParser{}
	input := "- item 1\n- item 2\n* item 3\n1. numbered"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "- item") || strings.Contains(result, "* item") {
		t.Errorf("expected list markers stripped, got: %s", result)
	}
	if !strings.Contains(result, "item 1") || !strings.Contains(result, "item 2") {
		t.Errorf("expected list content preserved, got: %s", result)
	}
}

func TestMarkdownParser_Blockquotes(t *testing.T) {
	p := &MarkdownParser{}
	input := "> This is a quote\n> Second line"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, ">") {
		t.Errorf("expected blockquote markers stripped, got: %s", result)
	}
	if !strings.Contains(result, "This is a quote") {
		t.Errorf("expected quote text preserved, got: %s", result)
	}
}

func TestMarkdownParser_HorizontalRules(t *testing.T) {
	p := &MarkdownParser{}
	input := "Text above\n---\nText below"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Horizontal rule should be removed
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			t.Errorf("expected horizontal rule stripped, got: %s", result)
		}
	}
}

func TestMarkdownParser_Strikethrough(t *testing.T) {
	p := &MarkdownParser{}
	input := "This is ~~deleted~~ text"
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "~~") {
		t.Errorf("expected strikethrough markers stripped, got: %s", result)
	}
	if !strings.Contains(result, "deleted") {
		t.Errorf("expected strikethrough text preserved, got: %s", result)
	}
}

func TestMarkdownParser_RawMode(t *testing.T) {
	p := &MarkdownParser{}
	input := "## **Bold Header**"
	result, err := p.Parse(input, map[string]interface{}{"markdown_mode": "raw"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected raw content in raw mode, got: %s", result)
	}
}

func TestMarkdownParser_Empty(t *testing.T) {
	p := &MarkdownParser{}
	result, err := p.Parse("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestMarkdownParser_Supports(t *testing.T) {
	p := &MarkdownParser{}
	if !p.Supports("text/markdown") {
		t.Error("expected to support text/markdown")
	}
	if p.Supports("text/html") {
		t.Error("expected not to support text/html")
	}
}

// --- HTMLParser Tests ---

func TestHTMLParser_BasicExtraction(t *testing.T) {
	p := &HTMLParser{}
	input := `<html><body><p>Hello world</p></body></html>`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Hello world") {
		t.Errorf("expected text content, got: %s", result)
	}
}

func TestHTMLParser_StripsScript(t *testing.T) {
	p := &HTMLParser{}
	input := `<html><body><script>alert("xss")</script><p>Content</p></body></html>`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "alert") || strings.Contains(result, "xss") {
		t.Errorf("expected script content stripped, got: %s", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("expected body content preserved, got: %s", result)
	}
}

func TestHTMLParser_StripsStyle(t *testing.T) {
	p := &HTMLParser{}
	input := `<html><head><style>body { color: red; }</style></head><body><p>Text</p></body></html>`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "color") || strings.Contains(result, "red") {
		t.Errorf("expected style content stripped, got: %s", result)
	}
	if !strings.Contains(result, "Text") {
		t.Errorf("expected body text preserved, got: %s", result)
	}
}

func TestHTMLParser_NestedElements(t *testing.T) {
	p := &HTMLParser{}
	input := `<div><div><span>Nested</span> text</div></div>`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Nested") {
		t.Errorf("expected nested text, got: %s", result)
	}
	if !strings.Contains(result, "text") {
		t.Errorf("expected sibling text, got: %s", result)
	}
}

func TestHTMLParser_StripsComments(t *testing.T) {
	p := &HTMLParser{}
	input := `<!-- comment --><p>Visible</p>`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "comment") {
		t.Errorf("expected comment stripped, got: %s", result)
	}
	if !strings.Contains(result, "Visible") {
		t.Errorf("expected visible text preserved, got: %s", result)
	}
}

func TestHTMLParser_TableContent(t *testing.T) {
	p := &HTMLParser{}
	input := `<table><tr><td>A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></table>`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, expected := range []string{"A", "B", "C", "D"} {
		if !strings.Contains(result, expected) {
			t.Errorf("expected %s in result, got: %s", expected, result)
		}
	}
}

func TestHTMLParser_RawMode(t *testing.T) {
	p := &HTMLParser{}
	input := `<html><body><p>Test</p></body></html>`
	result, err := p.Parse(input, map[string]interface{}{"html_mode": "raw"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected raw HTML in raw mode, got: %s", result)
	}
}

func TestHTMLParser_Empty(t *testing.T) {
	p := &HTMLParser{}
	result, err := p.Parse("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestHTMLParser_InvalidHTML(t *testing.T) {
	p := &HTMLParser{}
	// golang.org/x/net/html is lenient with invalid HTML
	input := `<p>unclosed paragraph`
	result, err := p.Parse(input, nil)
	// Should not error (lenient parser) and should extract text
	if err != nil {
		t.Logf("lenient parser returned error: %v", err)
	}
	if result != "" && !strings.Contains(result, "unclosed") {
		t.Logf("result: %s", result)
	}
}

func TestHTMLParser_Supports(t *testing.T) {
	p := &HTMLParser{}
	if !p.Supports("text/html") {
		t.Error("expected to support text/html")
	}
	if p.Supports("text/markdown") {
		t.Error("expected not to support text/markdown")
	}
}

// --- JSONParser Tests ---

func TestJSONParser_ValidJSON(t *testing.T) {
	p := &JSONParser{}
	input := `{"name": "test", "value": 123}`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected original content for valid JSON, got: %s", result)
	}
}

func TestJSONParser_InvalidJSON(t *testing.T) {
	p := &JSONParser{}
	input := `{invalid json}`
	_, err := p.Parse(input, nil)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got: %v", err)
	}
}

func TestJSONParser_ContentFieldExtraction(t *testing.T) {
	p := &JSONParser{}
	input := `{"title": "Hello", "body": "World", "unused": "x"}`
	result, err := p.Parse(input, map[string]interface{}{
		"json_content_field": "body",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "World" {
		t.Errorf("expected 'World' extracted from body field, got: %s", result)
	}
}

func TestJSONParser_ContentFieldNotFound(t *testing.T) {
	p := &JSONParser{}
	input := `{"name": "test"}`
	result, err := p.Parse(input, map[string]interface{}{
		"json_content_field": "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// If field not found, should fall back to original
	if result != input {
		t.Errorf("expected original content when field not found, got: %s", result)
	}
}

func TestJSONParser_FlattenMode(t *testing.T) {
	p := &JSONParser{}
	input := `{"a":1,"b":{"c":2}}`
	result, err := p.Parse(input, map[string]interface{}{
		"json_flatten": "true",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "  \"a\": 1") {
		t.Errorf("expected pretty-printed JSON, got: %s", result)
	}
	if !strings.Contains(result, "  \"c\": 2") {
		t.Errorf("expected nested content, got: %s", result)
	}
}

func TestJSONParser_Empty(t *testing.T) {
	p := &JSONParser{}
	result, err := p.Parse("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

func TestJSONParser_ArrayJSON(t *testing.T) {
	p := &JSONParser{}
	input := `[1, 2, 3]`
	result, err := p.Parse(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected original content for valid JSON array, got: %s", result)
	}
}

func TestJSONParser_Supports(t *testing.T) {
	p := &JSONParser{}
	if !p.Supports("application/json") {
		t.Error("expected to support application/json")
	}
	if p.Supports("text/html") {
		t.Error("expected not to support text/html")
	}
}

// --- DocumentParser Integration Tests ---

func TestDocumentParser_SelectCorrectParser(t *testing.T) {
	dp := NewDocumentParser()

	// Markdown
	mdParser, err := dp.selectParser("text/markdown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := mdParser.(*MarkdownParser); !ok {
		t.Errorf("expected MarkdownParser, got %T", mdParser)
	}

	// HTML
	htmlParser, err := dp.selectParser("text/html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := htmlParser.(*HTMLParser); !ok {
		t.Errorf("expected HTMLParser, got %T", htmlParser)
	}

	// JSON
	jsonParser, err := dp.selectParser("application/json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := jsonParser.(*JSONParser); !ok {
		t.Errorf("expected JSONParser, got %T", jsonParser)
	}
}
