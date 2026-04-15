import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';

// ============================================================================
// CoRag/Aetheris VSCode Extension
// ============================================================================
// Provides:
// - Snippets for Go agent patterns
// - Syntax highlighting for .corag files
// - Commands for scaffolding agents and viewing traces
// - Configuration validation and quick fixes

// --------------------------------------------------------------------------
// Constants
// --------------------------------------------------------------------------

const CORAG_CONFIG_KEY = 'corag';
const AGENT_TYPES = ['react', 'deer', 'manus', 'conversation', 'graph', 'workflow'];
const JOB_STATUS_VALUES = ['pending', 'running', 'completed', 'failed', 'cancelled', 'waiting', 'parked', 'retrying'];
const EVENT_TYPES = [
    'RUN_CREATED', 'RUN_PAUSED', 'RUN_RESUMED', 'STEP_STARTED', 
    'STEP_COMPLETED', 'TOOL_CALL_STARTED', 'TOOL_CALL_ENDED', 
    'RUN_FAILED', 'RUN_SUCCEEDED', 'HUMAN_INJECTED'
];

// --------------------------------------------------------------------------
// Utility Functions
// --------------------------------------------------------------------------

function getCoragConfig(): vscode.WorkspaceConfiguration {
    return vscode.workspace.getConfiguration(CORAG_CONFIG_KEY);
}

function showInfoMessage(message: string): void {
    vscode.window.showInformationMessage(message);
}

function showErrorMessage(message: string): void {
    vscode.window.showErrorMessage(message);
}

function getWorkspaceRoot(): string | undefined {
    if (vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders.length > 0) {
        return vscode.workspace.workspaceFolders[0].uri.fsPath;
    }
    return undefined;
}

// --------------------------------------------------------------------------
// Command: New Agent
// --------------------------------------------------------------------------

async function newAgentCommand(): Promise<void> {
    const workspaceRoot = getWorkspaceRoot();
    if (!workspaceRoot) {
        showErrorMessage('No workspace folder open');
        return;
    }

    // Get agent name
    const agentName = await vscode.window.showInputBox({
        prompt: 'Enter agent name (e.g., my-agent)',
        validateInput: (value) => {
            if (!value || value.trim() === '') {
                return 'Agent name is required';
            }
            if (!/^[a-z][a-z0-9_-]*$/.test(value)) {
                return 'Agent name must start with lowercase letter and contain only letters, numbers, hyphens, and underscores';
            }
            return null;
        }
    });

    if (!agentName) { return; }

    // Get agent type
    const agentType = await vscode.window.showQuickPick(AGENT_TYPES.map(t => ({
        label: t,
        description: getAgentTypeDescription(t)
    })), {
        prompt: 'Select agent type',
        placeHolder: 'Select agent type'
    });

    if (!agentType) { return; }

    // Create agent directory
    const agentsDir = path.join(workspaceRoot, 'internal', 'agent', 'examples', agentName);
    const mainFile = path.join(agentsDir, 'main.go');
    const testFile = path.join(agentsDir, 'main_test.go');

    // Check if directory already exists
    if (fs.existsSync(agentsDir)) {
        showErrorMessage(`Directory ${agentsDir} already exists`);
        return;
    }

    fs.mkdirSync(agentsDir, { recursive: true });

    // Generate agent content
    const mainContent = generateAgentMainContent(agentName, agentType.label);
    const testContent = generateAgentTestContent(agentName, agentType.label);

    fs.writeFileSync(mainFile, mainContent);
    fs.writeFileSync(testFile, testContent);

    // Open the file
    const doc = await vscode.window.showTextDocument(vscode.Uri.file(mainFile));
    showInfoMessage(`Created new CoRag agent: ${agentName}`);

    // Format the document
    await vscode.commands.executeCommand('editor.action.formatDocument');
}

function getAgentTypeDescription(type: string): string {
    switch (type) {
        case 'react': return 'ReAct (Reasoning + Acting) agent';
        case 'deer': return 'DEER-Go enhanced reasoning agent';
        case 'manus': return 'Manus autonomous execution agent';
        case 'conversation': return 'Simple conversation chain';
        case 'graph': return 'Directed conversation graph';
        case 'workflow': return 'Linear workflow';
        default: return '';
    }
}

function generateAgentMainContent(name: string, type: string): string {
    return `// Copyright 2026 CoRag Contributors
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
	"fmt"
	"os"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/eino"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino-ext/components/model/openai"
)

// ${name}AgentConfig holds configuration for the ${name} agent
type ${toPascalCase(name)}AgentConfig struct {
	Name        string
	Description string
	MaxSteps    int
	Tools       []string
}

// New${toPascalCase(name)}Agent creates a new ${name} agent
func New${toPascalCase(name)}Agent(ctx context.Context) (*adk.Runner, error) {
	// Create ChatModel
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  os.Getenv("LLM_MODEL"),
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	// Build agent config
	agentCfg := &eino.AgentBuildConfig{
		Name:        "${name}",
		Description: "${type} agent",
		Type:        "${type}",
		MaxSteps:    10,
		Streaming:   true,
	}

	// Create agent via AgentFactory
	factory := eino.NewAgentFactory(nil, nil)
	runner, err := factory.CreateAgent(ctx, *agentCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return runner, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create agent
	runner, err := New${toPascalCase(name)}Agent(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\\n", err)
		os.Exit(1)
	}

	// Run agent
	iter := runner.Query(ctx, "Hello, how can you help me?")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\\n", event.Err)
			continue
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg := event.Output.MessageOutput.Message
			if msg != nil && msg.Content != "" {
				fmt.Printf("Response: %s\\n", msg.Content)
			}
		}
	}
}

func toPascalCase(s string) string {
	// Simple PascalCase conversion
	result := ""
	for i, c := range s {
		if c == '_' || c == '-' {
			continue
		}
		if i == 0 || s[i-1] == '_' || s[i-1] == '-' {
			result += string(s[i]-('a'-'A'))
		} else {
			result += string(c)
		}
	}
	return result
}
`;
}

function toPascalCase(name: string): string {
    return name.split(/[_-]/).map(s => 
        s.charAt(0).toUpperCase() + s.slice(1).toLowerCase()
    ).join('');
}

function generateAgentTestContent(name: string, type: string): string {
    return `// Copyright 2026 CoRag Contributors
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
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/eino"
)

func Test${toPascalCase(name)}Agent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create agent config
	cfg := eino.AgentBuildConfig{
		Name:        "${name}",
		Description: "${type} agent",
		Type:        "${type}",
		MaxSteps:    5,
		Streaming:   false,
	}

	// Verify config is valid
	if cfg.Name == "" {
		t.Error("Agent name should not be empty")
	}
	if cfg.MaxSteps <= 0 {
		t.Error("MaxSteps should be positive")
	}
}

func Test${toPascalCase(name)}AgentConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     eino.AgentBuildConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: eino.AgentBuildConfig{
				Name:        "${name}",
				Description: "test agent",
				Type:        "${type}",
				MaxSteps:    10,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			cfg: eino.AgentBuildConfig{
				Description: "test agent",
				Type:        "${type}",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.Name == "" && !tt.wantErr {
				t.Error("Expected error for empty name")
			}
		})
	}
}
`;
}

// --------------------------------------------------------------------------
// Command: View Traces
// --------------------------------------------------------------------------

async function viewTracesCommand(): Promise<void> {
    const config = getCoragConfig();
    const tracePath = config.get<string>('tracePath', './traces');
    
    const workspaceRoot = getWorkspaceRoot();
    if (!workspaceRoot) {
        showErrorMessage('No workspace folder open');
        return;
    }

    const fullTracePath = path.isAbsolute(tracePath) 
        ? tracePath 
        : path.join(workspaceRoot, tracePath);

    // Check if trace directory exists
    if (!fs.existsSync(fullTracePath)) {
        // Create traces directory
        fs.mkdirSync(fullTracePath, { recursive: true });
        showInfoMessage(`Created trace directory: ${fullTracePath}`);
    }

    // Read existing trace files
    let traceFiles: string[] = [];
    try {
        traceFiles = fs.readdirSync(fullTracePath)
            .filter(f => f.endsWith('.json') || f.endsWith('.trace'))
            .map(f => path.join(fullTracePath, f))
            .sort((a, b) => fs.statSync(b).mtime.getTime() - fs.statSync(a).mtime.getTime());
    } catch (e) {
        showErrorMessage(`Failed to read trace directory: ${e}`);
        return;
    }

    if (traceFiles.length === 0) {
        // Create a sample trace file for demonstration
        const sampleTracePath = path.join(fullTracePath, 'sample.trace.json');
        fs.writeFileSync(sampleTracePath, generateSampleTrace());
        traceFiles = [sampleTracePath];
        showInfoMessage('Created sample trace file');
    }

    // Show trace picker
    const selectedTrace = await vscode.window.showQuickPick(
        traceFiles.map(f => ({
            label: path.basename(f),
            description: formatFileDate(fs.statSync(f).mtime),
            path: f
        })),
        { 
            prompt: 'Select a trace file to view',
            placeHolder: 'Choose trace file'
        }
    );

    if (!selectedTrace) { return; }

    // Open trace file
    const doc = await vscode.window.showTextDocument(vscode.Uri.file(selectedTrace.path));
    
    // Switch to JSON language mode if needed
    if (doc.languageId !== 'json') {
        await vscode.languages.setTextDocumentLanguage(doc, 'json');
    }

    // Register trace formatting provider
    vscode.commands.executeCommand('editor.action.formatDocument');
}

function formatFileDate(date: Date): string {
    return date.toLocaleString();
}

function generateSampleTrace(): string {
    const sampleEvents = [
        {
            id: "evt-001",
            run_id: "run-001",
            type: "RUN_CREATED",
            seq: 1,
            actor: "system",
            payload: { workflow_id: "example-workflow" },
            occurred_at: new Date().toISOString()
        },
        {
            id: "evt-002",
            run_id: "run-001",
            step_id: "step-001",
            type: "STEP_STARTED",
            seq: 2,
            actor: "agent",
            payload: { node_name: "analyze_request" },
            occurred_at: new Date().toISOString()
        },
        {
            id: "evt-003",
            run_id: "run-001",
            step_id: "step-001",
            type: "TOOL_CALL_STARTED",
            seq: 3,
            tool_call_id: "tc-001",
            actor: "agent",
            payload: { tool_name: "web_search", input: { query: "example" } },
            occurred_at: new Date().toISOString()
        },
        {
            id: "evt-004",
            run_id: "run-001",
            step_id: "step-001",
            type: "TOOL_CALL_ENDED",
            seq: 4,
            tool_call_id: "tc-001",
            actor: "agent",
            payload: { tool_name: "web_search", output: { results: [] } },
            occurred_at: new Date().toISOString()
        },
        {
            id: "evt-005",
            run_id: "run-001",
            step_id: "step-001",
            type: "STEP_COMPLETED",
            seq: 5,
            actor: "agent",
            payload: { node_name: "analyze_request", status: "success" },
            occurred_at: new Date().toISOString()
        },
        {
            id: "evt-006",
            run_id: "run-001",
            type: "RUN_SUCCEEDED",
            seq: 6,
            actor: "system",
            payload: { output: { answer: "Analysis complete" } },
            occurred_at: new Date().toISOString()
        }
    ];

    return JSON.stringify({
        version: "1.0",
        run: {
            id: "run-001",
            workflow_id: "example-workflow",
            status: "SUCCEEDED",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
        },
        events: sampleEvents
    }, null, 2);
}

// --------------------------------------------------------------------------
// Command: Scaffold Workflow
// --------------------------------------------------------------------------

async function scaffoldWorkflowCommand(): Promise<void> {
    const workspaceRoot = getWorkspaceRoot();
    if (!workspaceRoot) {
        showErrorMessage('No workspace folder open');
        return;
    }

    // Get workflow name
    const workflowName = await vscode.window.showInputBox({
        prompt: 'Enter workflow name (e.g., my-workflow)',
        validateInput: (value) => {
            if (!value || value.trim() === '') {
                return 'Workflow name is required';
            }
            return null;
        }
    });

    if (!workflowName) { return; }

    // Create workflow file
    const workflowsDir = path.join(workspaceRoot, 'workflows');
    const workflowFile = path.join(workflowsDir, `${workflowName}.corag`);
    const workflowYaml = path.join(workflowsDir, `${workflowName}.yaml`);

    fs.mkdirSync(workflowsDir, { recursive: true });

    // Generate .corag workflow content
    const coragContent = generateWorkflowCoragContent(workflowName);
    const yamlContent = generateWorkflowYamlContent(workflowName);

    fs.writeFileSync(workflowFile, coragContent);
    fs.writeFileSync(workflowYaml, yamlContent);

    // Open the corag file
    const doc = await vscode.window.showTextDocument(vscode.Uri.file(workflowFile));
    showInfoMessage(`Created workflow: ${workflowName}`);

    await vscode.commands.executeCommand('editor.action.formatDocument');
}

function generateWorkflowCoragContent(name: string): string {
    return `# CoRag Workflow: ${name}
# This file defines a ${name} workflow for CoRag/Aetheris runtime

workflow ${name} {
  version = "1.0"
  description = "${name} workflow"

  # Define nodes in the workflow
  nodes {
    # Start node
    start {
      type = "trigger"
      input = { message = "${name} workflow started" }
    }

    # LLM processing node
    process {
      type = "llm"
      prompt = "Process the following input and provide a structured response"
      model = "\${LLM_MODEL}"
      max_tokens = 1000
    }

    # Tool call node
    execute {
      type = "tool"
      tool_name = "example_tool"
      timeout = 30
    }

    # Human approval node (optional)
    approval {
      type = "wait"
      wait_kind = "signal"
      correlation_key = "${name}-approval"
      timeout = "1h"
    }

    # End node
    end {
      type = "output"
      output_template = "Workflow ${name} completed successfully"
    }
  }

  # Define edges (connections between nodes)
  edges = [
    { from = "start", to = "process" },
    { from = "process", to = "execute" },
    { from = "execute", to = "approval", condition = "requires_approval" },
    { from = "execute", to = "end", condition = "auto_approve" },
    { from = "approval", to = "end" }
  ]
}
`;
}

function generateWorkflowYamlContent(name: string): string {
    return `# CoRag Agents Configuration
# Workflow: ${name}

agents:
  ${name}:
    type: "workflow"
    description: "${name} workflow agent"
    llm: "default"
    max_iterations: 10
    tools:
      - "web_search"
      - "calculator"

llm:
  provider: "openai"
  model: "gpt-4o-mini"
  api_key: "\${OPENAI_API_KEY}"

tools:
  enabled:
    - "web_search"
    - "calculator"
`;
}

// --------------------------------------------------------------------------
// Command: Validate Config
// --------------------------------------------------------------------------

async function validateConfigCommand(): Promise<void> {
    const workspaceRoot = getWorkspaceRoot();
    if (!workspaceRoot) {
        showErrorMessage('No workspace folder open');
        return;
    }

    const config = getCoragConfig();
    const configPath = config.get<string>('configPath', './configs/agents.yaml');
    const fullConfigPath = path.isAbsolute(configPath) 
        ? configPath 
        : path.join(workspaceRoot, configPath);

    if (!fs.existsSync(fullConfigPath)) {
        showErrorMessage(`Config file not found: ${fullConfigPath}`);
        return;
    }

    try {
        const content = fs.readFileSync(fullConfigPath, 'utf-8');
        const issues = validateAgentsYaml(content);

        if (issues.length === 0) {
            showInfoMessage('Configuration is valid ✓');
            return;
        }

        // Show issues
        const selectedIssue = await vscode.window.showQuickPick(
            issues.map((issue, idx) => ({
                label: `Line ${issue.line}: ${issue.type}`,
                description: issue.message,
                detail: issue.suggestion
            })),
            { 
                prompt: 'Configuration issues found',
                placeHolder: 'Select an issue to view details'
            }
        );

        if (selectedIssue) {
            // Open config file and highlight the issue line
            const doc = await vscode.window.showTextDocument(vscode.Uri.file(fullConfigPath));
            const lineNum = parseInt(selectedIssue.label.split(' ')[1]) - 1;
            const range = new vscode.Range(lineNum, 0, lineNum, 100);
            await doc.revealRange(range, vscode.TextEditorRevealType.InCenter);
        }
    } catch (e) {
        showErrorMessage(`Failed to validate config: ${e}`);
    }
}

interface ConfigIssue {
    line: number;
    type: string;
    message: string;
    suggestion?: string;
}

function validateAgentsYaml(content: string): ConfigIssue[] {
    const issues: ConfigIssue[] = [];
    const lines = content.split('\n');
    const validAgentTypes = ['react', 'deer', 'manus', 'conversation', 'graph', 'workflow'];

    let inAgentsSection = false;
    let currentAgent: string | null = null;
    let inToolsSection = false;

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        const lineNum = i + 1;

        // Track section
        if (line.match(/^agents:\s*$/)) {
            inAgentsSection = true;
            inToolsSection = false;
            continue;
        }
        if (line.match(/^tools:\s*$/)) {
            inAgentsSection = false;
            inToolsSection = true;
            continue;
        }
        if (line.match(/^llm:\s*$/)) {
            inAgentsSection = false;
            inToolsSection = false;
            continue;
        }

        // Check for invalid agent types
        const typeMatch = line.match(/^\s+type:\s*["']?(\w+)["']?\s*$/);
        if (typeMatch && inAgentsSection) {
            const agentType = typeMatch[1];
            if (!validAgentTypes.includes(agentType)) {
                issues.push({
                    line: lineNum,
                    type: 'warning',
                    message: `Unknown agent type: ${agentType}`,
                    suggestion: `Valid types are: ${validAgentTypes.join(', ')}`
                });
            }
        }

        // Check for missing agent description
        if (inAgentsSection && line.match(/^\s{2,}[a-z][a-z0-9_-]+:\s*$/)) {
            currentAgent = line.trim().replace(':', '');
        }

        // Check for empty tools array
        const emptyToolsMatch = line.match(/^\s+tools:\s*\[\s*\]\s*$/);
        if (emptyToolsMatch) {
            issues.push({
                line: lineNum,
                type: 'info',
                message: `Agent '${currentAgent}' has empty tools array`,
                suggestion: 'This means all available tools will be used. Add specific tool names if you want to restrict tools.'
            });
        }

        // Check for hardcoded API keys
        if (line.includes('api_key:') && line.includes('sk-')) {
            issues.push({
                line: lineNum,
                type: 'error',
                message: 'Hardcoded API key detected',
                suggestion: 'Use environment variables: ${OPENAI_API_KEY} instead'
            });
        }

        // Check for invalid indentation
        if (line.length > 0 && line.match(/^[^\s]/)) {
            issues.push({
                line: lineNum,
                type: 'warning',
                message: 'Invalid indentation in YAML',
                suggestion: 'Use spaces for indentation (2 spaces recommended)'
            });
        }
    }

    return issues;
}

// --------------------------------------------------------------------------
// Diagnostics Provider for CoRag Files
// --------------------------------------------------------------------------

class CoRagDiagnosticProvider implements vscode.Disposable {
    private diagnosticCollection: vscode.DiagnosticCollection;
    private debounceTimer: NodeJS.Timeout | null = null;

    constructor() {
        this.diagnosticCollection = vscode.languages.createDiagnosticCollection('corag');
    }

    dispose(): void {
        this.diagnosticCollection.clear();
        this.diagnosticCollection.dispose();
        if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
        }
    }

    analyzeDocument(document: vscode.TextDocument): void {
        if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
        }

        this.debounceTimer = setTimeout(() => {
            this.doAnalyzeDocument(document);
        }, 500);
    }

    private doAnalyzeDocument(document: vscode.TextDocument): void {
        const uri = document.uri;
        const diagnostics: vscode.Diagnostic[] = [];

        if (document.languageId === 'yaml') {
            this.analyzeYamlDocument(document, diagnostics);
        } else if (document.languageId === 'go') {
            this.analyzeGoDocument(document, diagnostics);
        } else if (document.languageId === 'corag') {
            this.analyzeCoragDocument(document, diagnostics);
        }

        this.diagnosticCollection.set(uri, diagnostics);
    }

    private analyzeYamlDocument(document: vscode.TextDocument, diagnostics: vscode.Diagnostic[]): void {
        const content = document.getText();
        const lines = content.split('\n');

        // Check for missing required fields
        if (!content.includes('type:')) {
            // This is informational, not an error
        }

        // Check for hardcoded secrets
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            if (line.match(/api_key:\s*["']sk-[a-zA-Z0-9]/)) {
                const range = new vscode.Range(i, 0, i, line.length);
                diagnostics.push({
                    range,
                    severity: vscode.DiagnosticSeverity.Error,
                    message: 'Hardcoded API key detected. Use environment variable instead.',
                    source: 'CoRag'
                });
            }
        }
    }

    private analyzeGoDocument(document: vscode.TextDocument, diagnostics: vscode.Diagnostic[]): void {
        const content = document.getText();
        const lines = content.split('\n');

        // Check for deprecated Agent usage
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            
            // Check for deprecated agent.New() usage
            if (line.match(/agent\.New\s*\(/)) {
                const range = new vscode.Range(i, 0, i, line.length);
                diagnostics.push({
                    range,
                    severity: vscode.DiagnosticSeverity.Warning,
                    message: 'agent.New() is deprecated. Use eino.AgentFactory instead.',
                    source: 'CoRag',
                    relatedInformation: [
                        new vscode.DiagnosticRelatedInformation(
                            new vscode.Location(document.uri, range),
                            'See AGENTS.md for migration guide'
                        )
                    ]
                });
            }

            // Check for context.Background() in main
            if (line.match(/context\.Background\(\)/) && !line.trim().startsWith('//')) {
                const range = new vscode.Range(i, 0, i, line.length);
                diagnostics.push({
                    range,
                    severity: vscode.DiagnosticSeverity.Hint,
                    message: 'Consider using context.WithTimeout or context.WithCancel instead of context.Background()',
                    source: 'CoRag'
                });
            }
        }
    }

    private analyzeCoragDocument(document: vscode.TextDocument, diagnostics: vscode.Diagnostic[]): void {
        const content = document.getText();
        const lines = content.split('\n');

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];

            // Check for missing closing braces in workflow definition
            // Check for undefined node references in edges
            const edgeMatch = line.match(/^\s*\{?\s*from\s*=\s*["'](\w+)["']/);
            if (edgeMatch) {
                const nodeName = edgeMatch[1];
                // Check if this node is defined somewhere in the document
                const nodeRegex = new RegExp(`^\\s*${nodeName}\\s*\\{`);
                const nodeExists = lines.some(l => l.match(nodeRegex));
                if (!nodeExists) {
                    const range = new vscode.Range(i, 0, i, line.length);
                    diagnostics.push({
                        range,
                        severity: vscode.DiagnosticSeverity.Error,
                        message: `Referenced node '${nodeName}' is not defined`,
                        source: 'CoRag'
                    });
                }
            }

            // Check for invalid event types in trace references
            const eventTypeMatch = line.match(/event_type:\s*["']?(\w+)["']?/);
            if (eventTypeMatch) {
                const eventType = eventTypeMatch[1];
                if (!EVENT_TYPES.includes(eventType)) {
                    const range = new vscode.Range(i, 0, i, line.length);
                    diagnostics.push({
                        range,
                        severity: vscode.DiagnosticSeverity.Warning,
                        message: `Unknown event type: ${eventType}`,
                        source: 'CoRag',
                        relatedInformation: [
                            new vscode.DiagnosticRelatedInformation(
                                new vscode.Location(document.uri, range),
                                `Valid types: ${EVENT_TYPES.join(', ')}`
                            )
                        ]
                    });
                }
            }
        }
    }
}

// --------------------------------------------------------------------------
// Status Bar Item
// --------------------------------------------------------------------------

class CoRagStatusBarItem {
    private item: vscode.StatusBarItem;
    private isActive: boolean = false;

    constructor() {
        this.item = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Left,
            100
        );
        this.item.text = '$(circuit-board) CoRag';
        this.item.tooltip = 'CoRag/Aetheris Agent Runtime';
        this.item.command = 'corag.viewTraces';
        this.update();
    }

    show(): void {
        this.isActive = true;
        this.update();
        this.item.show();
    }

    hide(): void {
        this.isActive = false;
        this.item.hide();
    }

    private update(): void {
        this.item.text = this.isActive 
            ? '$(circuit-board) CoRag: Active'
            : '$(circuit-board) CoRag';
    }

    dispose(): void {
        this.item.dispose();
    }
}

// --------------------------------------------------------------------------
// Extension Activation
// --------------------------------------------------------------------------

let diagnosticProvider: CoRagDiagnosticProvider | null = null;
let statusBarItem: CoRagStatusBarItem | null = null;

export function activate(context: vscode.ExtensionContext): void {
    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('corag.newAgent', newAgentCommand),
        vscode.commands.registerCommand('corag.viewTraces', viewTracesCommand),
        vscode.commands.registerCommand('corag.scaffoldWorkflow', scaffoldWorkflowCommand),
        vscode.commands.registerCommand('corag.validateConfig', validateConfigCommand)
    );

    // Initialize diagnostic provider
    diagnosticProvider = new CoRagDiagnosticProvider();
    context.subscriptions.push(diagnosticProvider);

    // Watch for document changes
    context.subscriptions.push(
        vscode.workspace.onDidChangeTextDocument((event) => {
            if (diagnosticProvider) {
                diagnosticProvider.analyzeDocument(event.document);
            }
        })
    );

    // Watch for file opens
    context.subscriptions.push(
        vscode.workspace.onDidOpenTextDocument((document) => {
            if (diagnosticProvider) {
                diagnosticProvider.analyzeDocument(document);
            }
        })
    );

    // Initialize status bar
    statusBarItem = new CoRagStatusBarItem();
    context.subscriptions.push(statusBarItem);

    // Show status bar when a Go or YAML file is opened
    context.subscriptions.push(
        vscode.window.onDidChangeActiveTextEditor((editor) => {
            if (editor) {
                const langId = editor.document.languageId;
                if (langId === 'go' || langId === 'yaml' || langId === 'corag') {
                    statusBarItem?.show();
                }
            }
        })
    );

    // Show welcome message on first activation
    const welcomeKey = 'corag.welcomeShown';
    const welcomeShown = context.globalState.get<boolean>(welcomeKey, false);
    if (!welcomeShown) {
        context.globalState.update(welcomeKey, true);
        showInfoMessage('CoRag/Aetheris extension activated. Use Ctrl+Shift+A to create a new agent.');
    }
}

export function deactivate(): void {
    if (diagnosticProvider) {
        diagnosticProvider.dispose();
    }
    if (statusBarItem) {
        statusBarItem.dispose();
    }
}
