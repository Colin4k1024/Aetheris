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

// Package main demonstrates how to connect to Hermes MCP Server via stdio transport
// and call its tools from an Aetheris Eino Workflow.
//
// This example shows the complete flow:
//   - Connect to hermes mcp serve subprocess via stdio
//   - Initialize the MCP connection
//   - List available tools
//   - Call a tool (conversations_list)
//   - Close the connection
//
// Run:
//
//	go run ./examples/workflow/hermes_mcp_example.go
//
// Prerequisites:
//   - hermes command must be available in PATH
//   - hermes mcp serve must be executable without errors
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/tools/mcp"
)

func main() {
	ctx := context.Background()

	// ========================================================================
	// Step 1: Create stdio transport connecting to "hermes mcp serve"
	// ========================================================================
	transport, err := mcp.NewStdioTransport(ctx, mcp.StdioConfig{
		Command: "hermes",
		Args:    []string{"mcp", "serve"},
		// Optional: set working directory or environment variables
		// Dir: "/path/to/hermes",
		// Env: []string{"HERMES_HOME=/Users/jiafan/.hermes"},
	})
	if err != nil {
		log.Fatalf("Failed to create stdio transport: %v", err)
	}
	defer func() {
		if closeErr := transport.Close(); closeErr != nil {
			log.Printf("Warning: error closing transport: %v", closeErr)
		}
	}()

	// ========================================================================
	// Step 2: Create MCP client and initialize connection
	// ========================================================================
	client := mcp.NewClient(mcp.ClientConfig{
		Name:        "hermes",
		Transport:   transport,
		InitTimeout: 30 * time.Second,
		CallTimeout: 60 * time.Second,
	})

	if err := client.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize MCP client: %v", err)
	}

	// Log server info
	serverInfo := client.ServerInfo()
	if serverInfo != nil {
		fmt.Printf("Connected to Hermes MCP Server: %s v%s\n",
			serverInfo.ServerInfo.Name, serverInfo.ServerInfo.Version)
		fmt.Printf("Protocol version: %s\n", serverInfo.ProtocolVersion)
	}

	// ========================================================================
	// Step 3: List available tools
	// ========================================================================
	tools := client.Tools()
	fmt.Printf("\nAvailable tools (%d):\n", len(tools))
	for _, tool := range tools {
		desc := tool.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Printf("  - %s: %s\n", tool.Name, desc)
	}

	// ========================================================================
	// Step 4: Call conversations_list tool
	// ========================================================================
	fmt.Println("\n--- Calling conversations_list ---")

	result, err := client.CallTool(ctx, "conversations_list", map[string]any{
		"limit": 10,
	})
	if err != nil {
		log.Fatalf("Failed to call conversations_list: %v", err)
	}

	// Parse and display result
	if len(result.Content) > 0 && result.Content[0].Type == "text" {
		var data struct {
			Count         int `json:"count"`
			Conversations []struct {
				SessionKey  string `json:"session_key"`
				Platform    string `json:"platform"`
				DisplayName string `json:"display_name"`
				UpdatedAt   string `json:"updated_at"`
			} `json:"conversations"`
		}
		if unmarshalErr := json.Unmarshal([]byte(result.Content[0].Text), &data); unmarshalErr != nil {
			fmt.Printf("Raw result:\n%s\n", result.Content[0].Text)
		} else {
			fmt.Printf("Found %d conversations:\n", data.Count)
			for _, conv := range data.Conversations {
				fmt.Printf("  [%s] %s (updated: %s)\n",
					conv.Platform, conv.DisplayName, conv.UpdatedAt)
			}
		}
	}

	// ========================================================================
	// Step 5: Call channels_list tool
	// ========================================================================
	fmt.Println("\n--- Calling channels_list ---")

	channelsResult, err := client.CallTool(ctx, "channels_list", map[string]any{
		//"platform": "telegram", // optional filter
	})
	if err != nil {
		log.Printf("Warning: channels_list failed: %v", err)
	} else if len(channelsResult.Content) > 0 && channelsResult.Content[0].Type == "text" {
		fmt.Printf("Channels result:\n%s\n", channelsResult.Content[0].Text)
	}

	// ========================================================================
	// Step 6: Close connection
	// ========================================================================
	if err := client.Close(); err != nil {
		log.Printf("Warning: error closing client: %v", err)
	}

	fmt.Println("\nHermes MCP example completed successfully.")
}

// Helper function to extract text from ContentBlock slice
func extractText(blocks []mcp.ContentBlock) string {
	var sb strings.Builder
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}
