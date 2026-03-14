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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRequest(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo:      Implementation{Name: "test", Version: "1.0"},
	}
	req, err := newRequest(1, MethodInitialize, params)
	require.NoError(t, err)
	require.Equal(t, "2.0", req.JSONRPC)
	require.Equal(t, int64(1), req.ID)
	require.Equal(t, MethodInitialize, req.Method)

	var decoded InitializeParams
	err = json.Unmarshal(req.Params, &decoded)
	require.NoError(t, err)
	require.Equal(t, ProtocolVersion, decoded.ProtocolVersion)
	require.Equal(t, "test", decoded.ClientInfo.Name)
}

func TestNewRequest_NilParams(t *testing.T) {
	req, err := newRequest(42, MethodPing, nil)
	require.NoError(t, err)
	require.Nil(t, req.Params)
	require.Equal(t, int64(42), req.ID)
}

func TestIDGenerator(t *testing.T) {
	var g IDGenerator
	id1 := g.Next()
	id2 := g.Next()
	id3 := g.Next()
	require.Equal(t, int64(1), id1)
	require.Equal(t, int64(2), id2)
	require.Equal(t, int64(3), id3)
}

func TestJSONRPCError_Error(t *testing.T) {
	e := &JSONRPCError{Code: ErrCodeMethodNotFound, Message: "method not found"}
	require.Contains(t, e.Error(), "method not found")
	require.Contains(t, e.Error(), "-32601")
}

func TestProtocolTypes_Marshal(t *testing.T) {
	// ToolsListResult
	result := ToolsListResult{
		Tools: []MCPToolDef{
			{Name: "read_file", Description: "Read a file", InputSchema: map[string]any{"type": "object"}},
		},
		NextCursor: "",
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)
	var decoded ToolsListResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Len(t, decoded.Tools, 1)
	require.Equal(t, "read_file", decoded.Tools[0].Name)

	// ToolsCallResult
	callResult := ToolsCallResult{
		Content: []ContentBlock{{Type: "text", Text: "hello world"}},
		IsError: false,
	}
	data, err = json.Marshal(callResult)
	require.NoError(t, err)
	var decodedCall ToolsCallResult
	err = json.Unmarshal(data, &decodedCall)
	require.NoError(t, err)
	require.Len(t, decodedCall.Content, 1)
	require.Equal(t, "hello world", decodedCall.Content[0].Text)
}

func TestContentToOutput_SingleText(t *testing.T) {
	result := &ToolsCallResult{
		Content: []ContentBlock{{Type: "text", Text: "output data"}},
	}
	out := contentToOutput(result)
	require.Equal(t, "output data", out)
}

func TestContentToOutput_MultipleBlocks(t *testing.T) {
	result := &ToolsCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: "line1"},
			{Type: "image", MimeType: "image/png", Data: "base64data"},
		},
	}
	out := contentToOutput(result)
	blocks, ok := out.([]map[string]any)
	require.True(t, ok)
	require.Len(t, blocks, 2)
	require.Equal(t, "text", blocks[0]["type"])
	require.Equal(t, "line1", blocks[0]["text"])
	require.Equal(t, "image", blocks[1]["type"])
	require.Equal(t, "image/png", blocks[1]["mimeType"])
}

func TestContentToOutput_Nil(t *testing.T) {
	require.Nil(t, contentToOutput(nil))
	require.Nil(t, contentToOutput(&ToolsCallResult{}))
}
