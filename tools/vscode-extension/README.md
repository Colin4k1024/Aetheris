# CoRag/Aetheris VSCode Extension

A VSCode extension for developing CoRag/Aetheris intelligent agent runtime applications.

## Features

### 1. Code Snippets

#### Go Snippets
| Prefix | Description |
|--------|-------------|
| `corag-import` | Import CoRag and Eino packages |
| `corag-agent-config` | Define agent configuration struct |
| `corag-new-agent` | Create new CoRag agent with AgentFactory |
| `corag-runner-query` | Run agent with Query and handle events |
| `corag-run-with-session` | Run agent with session for stateful execution |
| `corag-checkpoint` | Create agent with checkpoint support |
| `corag-human-approval` | Define human approval workflow with NodeWait |
| `corag-job-status` | Job status constants and handling |
| `corag-runtime-event` | Create runtime event for tracing |
| `corag-workflow` | Define workflow using eino compose graph |
| `corag-tool-bridge` | Use tool bridge to convert tools to Eino format |
| `corag-trace-event` | Create trace event for debugging |

#### YAML Snippets (agents.yaml)
| Prefix | Description |
|--------|-------------|
| `corag-agents-yaml` | Complete agents.yaml configuration |
| `corag-react-agent` | Add ReAct agent configuration |
| `corag-deer-agent` | Add DEER agent configuration |
| `corag-manus-agent` | Add Manus agent configuration |
| `corag-conversation-chain` | Add conversation chain configuration |
| `corag-workflow-agent` | Add linear workflow configuration |
| `corag-graph-agent` | Add directed graph configuration |
| `corag-llm-config` | Add LLM configuration |
| `corag-tools-config` | Add tools configuration |

### 2. Syntax Highlighting

- `.corag` files: Custom workflow configuration files with syntax highlighting for:
  - Keywords (`workflow`, `node`, `edge`, `trigger`, `llm`, `tool`, `wait`)
  - Comments
  - Strings and numbers
  - Node and edge definitions

### 3. Commands

| Command | Description | Keybinding |
|---------|-------------|------------|
| `CoRag: New Agent` | Scaffold a new agent file | `Ctrl+Shift+A` / `Cmd+Shift+A` |
| `CoRag: View Traces` | Open trace viewer | `Ctrl+Shift+T` / `Cmd+Shift+T` |
| `CoRag: Scaffold Workflow` | Create a new workflow file | - |
| `CoRag: Validate Config` | Validate agents.yaml configuration | - |

### 4. Diagnostics & Quick Fixes

The extension provides real-time validation for:

#### Go Files
- **Deprecated `agent.New()`**: Warns when using the deprecated Agent struct, suggests using `eino.AgentFactory` instead
- **Context usage**: Hints to use `context.WithTimeout` or `context.WithCancel` instead of `context.Background()`

#### YAML Files (agents.yaml)
- **Hardcoded API keys**: Detects and warns about hardcoded API keys (should use `${OPENAI_API_KEY}` pattern)
- **Invalid agent types**: Validates agent type against known types (react, deer, manus, conversation, graph, workflow)
- **Empty tools arrays**: Informational warning about empty tools meaning "use all tools"

#### .corag Files
- **Undefined node references**: Validates that nodes referenced in edges are actually defined
- **Invalid event types**: Validates runtime event types against known types

### 5. Status Bar

Shows a CoRag status indicator in the VSCode status bar when working with Go, YAML, or .corag files.

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris/tools/vscode-extension
```

2. Install dependencies:
```bash
npm install
```

3. Compile TypeScript:
```bash
npm run compile
```

4. Open in VSCode and run:
```bash
code .
```

5. Press `F5` to launch the Extension Development Host

### Package and Install

```bash
npm install -g vsce
vsce package
code --install-extension corag-aetheris-0.1.0.vsix
```

## Configuration

The extension adds the following settings to VSCode:

| Setting | Default | Description |
|---------|---------|-------------|
| `corag.tracePath` | `./traces` | Path to CoRag trace files |
| `corag.configPath` | `./configs/agents.yaml` | Path to CoRag agents configuration |
| `corag.enableValidation` | `true` | Enable real-time configuration validation |
| `corag.telemetryEnabled` | `false` | Enable telemetry for trace collection |

## Usage

### Creating a New Agent

1. Press `Ctrl+Shift+A` (or `Cmd+Shift+A` on macOS)
2. Enter the agent name (e.g., `my-agent`)
3. Select the agent type (react, deer, manus, etc.)
4. The extension creates:
   - `internal/agent/examples/{agent-name}/main.go`
   - `internal/agent/examples/{agent-name}/main_test.go`

### Viewing Traces

1. Press `Ctrl+Shift+T` (or `Cmd+Shift+T` on macOS)
2. Select a trace file from the list
3. The trace opens in a formatted JSON view

### Creating a Workflow

1. Run `CoRag: Scaffold Workflow` command
2. Enter the workflow name
3. Creates:
   - `workflows/{name}.corag` - Custom workflow file
   - `workflows/{name}.yaml` - YAML configuration

### Validating Configuration

Run `CoRag: Validate Config` to check for issues in `agents.yaml`.

## Project Structure

```
tools/vscode-extension/
├── package.json              # Extension manifest
├── README.md                # This file
├── tsconfig.json            # TypeScript configuration
├── snippets/
│   ├── go.json              # Go code snippets
│   └── agents.yaml.json     # YAML configuration snippets
├── syntaxes/
│   └── corag.tmLanguage.json # Syntax highlighting for .corag files
└── src/
    └── extension.ts          # Main extension code
```

## CoRag/Aetheris Concepts

### Agent Types

- **react**: ReAct (Reasoning + Acting) agent - basic think-act-observe loop
- **deer**: DEER-Go enhanced reasoning agent
- **manus**: Manus autonomous execution agent
- **conversation**: Simple conversation chain
- **graph**: Directed conversation graph
- **workflow**: Linear workflow

### Job Status

- `pending`: Job is waiting to be processed
- `running`: Job is currently being executed
- `completed`: Job finished successfully
- `failed`: Job encountered an error
- `cancelled`: Job was cancelled
- `waiting`: Job is waiting for a signal (short wait)
- `parked`: Job is parked (long wait, scheduler skips)
- `retrying`: Job failed and is waiting to retry

### Runtime Events

- `RUN_CREATED`: Run instance created
- `RUN_PAUSED`: Run paused
- `RUN_RESUMED`: Run resumed
- `STEP_STARTED`: Step execution started
- `STEP_COMPLETED`: Step execution completed
- `TOOL_CALL_STARTED`: Tool call started
- `TOOL_CALL_ENDED`: Tool call ended
- `RUN_FAILED`: Run failed
- `RUN_SUCCEEDED`: Run succeeded
- `HUMAN_INJECTED`: Human decision injected

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) in the main repository.

## License

Apache License 2.0 - see LICENSE in the main repository.
