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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	mcpgithub "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("⚠️  GITHUB_TOKEN not set, using mock mode")
		token = "mock"
	}

	ctx := context.Background()

	// Create GitHub MCP tool
	githubTool := mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
		Token: token,
	})

	// Print tool schema
	schema := githubTool.Schema()
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Printf("Tool: %s\n", githubTool.Name())
	fmt.Printf("Description: %s\n", githubTool.Description())
	fmt.Printf("Schema:\n%s\n\n", schemaJSON)

	// Execute search (will fail gracefully if no real token)
	result, err := githubTool.Execute(ctx, map[string]any{
		"action": "search_repos",
		"query":  "aetheris agent runtime golang",
		"limit":  3,
	})
	if err != nil {
		log.Printf("Search failed (expected without real token): %v", err)
	} else {
		fmt.Printf("Search result:\n%s\n", result)
	}
}
