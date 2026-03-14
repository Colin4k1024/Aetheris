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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// StdioTransport communicates with an MCP server via a subprocess's stdin/stdout.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser // captured for diagnostics

	scanner *bufio.Scanner

	mu     sync.Mutex
	closed bool
}

// StdioConfig configures how to launch an MCP server subprocess.
type StdioConfig struct {
	Command string
	Args    []string
	Env     []string // KEY=VALUE pairs appended to the current env
	Dir     string   // working directory; empty means current
}

// NewStdioTransport spawns the MCP server process and returns a transport.
func NewStdioTransport(ctx context.Context, cfg StdioConfig) (*StdioTransport, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("mcp stdio: command is required")
	}
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	if cfg.Dir != "" {
		cmd.Dir = cfg.Dir
	}
	if len(cfg.Env) > 0 {
		cmd.Env = append(cmd.Environ(), cfg.Env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp stdio: start process %q: %w", cfg.Command, err)
	}

	scanner := bufio.NewScanner(stdout)
	// MCP messages can be large; allow up to 10MB per line.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	return &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: scanner,
	}, nil
}

// Send writes a JSON-RPC request to the server's stdin.
func (t *StdioTransport) Send(msg *JSONRPCRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("mcp stdio: transport closed")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("mcp stdio: marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("mcp stdio: write: %w", err)
	}
	return nil
}

// SendNotification writes a JSON-RPC notification (no ID) to the server's stdin.
func (t *StdioTransport) SendNotification(msg *JSONRPCNotification) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("mcp stdio: transport closed")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("mcp stdio: marshal notification: %w", err)
	}
	data = append(data, '\n')
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("mcp stdio: write notification: %w", err)
	}
	return nil
}

// Receive reads and parses the next JSON-RPC response from the server's stdout.
func (t *StdioTransport) Receive() (*JSONRPCResponse, error) {
	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("mcp stdio: read: %w", err)
		}
		return nil, io.EOF
	}
	line := t.scanner.Bytes()
	var resp JSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("mcp stdio: unmarshal response: %w", err)
	}
	return &resp, nil
}

// Close terminates the subprocess and releases resources.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	_ = t.stdin.Close()
	// Wait for the process to exit; ignore error since we are shutting down.
	_ = t.cmd.Wait()
	return nil
}
