# CoRag Playground

An online playground for trying out CoRag/Aetheris agents without local installation.

## Features

- **Agent Configuration Editor**: Write and edit agent configurations in YAML or JSON format
- **Real-time Execution Trace**: Watch agent execution step-by-step with thought/action/observation events
- **Pre-built Examples**: Start quickly with ReAct, RAG Assistant, or Minimal configurations
- **API Key Management**: Securely input LLM API keys (stored locally in browser only)
- **Docker Support**: Easy deployment with Docker and docker-compose

## Quick Start

### Using Docker (Recommended)

```bash
cd tools/playground
docker-compose up
```

Then open http://localhost:8081 in your browser.

### Local Development

```bash
cd tools/playground
go mod tidy
go run main.go
```

Open http://localhost:8081 in your browser.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/api/run` | Start an agent run |
| GET | `/api/trace/:run_id` | Get execution trace |
| GET | `/api/runs` | List all runs |
| GET | `/api/run/:run_id` | Get run status |

### POST /api/run

Request body:
```json
{
  "agent_config": "# YAML or JSON configuration",
  "query": "What is CoRag?",
  "api_key": "sk-...",  // optional
  "config_type": "yaml"  // or "json"
}
```

Response:
```json
{
  "run_id": "run_1234567890",
  "status": "running",
  "created_at": "2026-04-14T10:15:00Z"
}
```

### GET /api/trace/:run_id

Response:
```json
{
  "run_id": "run_1234567890",
  "status": "completed",
  "events": [
    {
      "timestamp": "2026-04-14T10:15:00Z",
      "type": "thought",
      "content": "Analyzing the query..."
    }
  ],
  "result": "Final agent response..."
}
```

## Configuration Format

### YAML Format

```yaml
agents:
  react:
    type: "react"
    description: "ReAct Agent"
    llm: "default"
    max_iterations: 10
    tools:
      - "web_search"
      - "calculator"
    system_prompt: |
      You are a helpful AI assistant.

llm:
  provider: "openai"
  model: "gpt-4o-mini"
  api_key: "${OPENAI_API_KEY}"

tools:
  enabled:
    - "web_search"
    - "calculator"
```

### JSON Format

```json
{
  "agents": {
    "react": {
      "type": "react",
      "description": "ReAct Agent",
      "llm": "default",
      "max_iterations": 10,
      "tools": ["web_search", "calculator"],
      "system_prompt": "You are a helpful AI assistant."
    }
  },
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "api_key": "${OPENAI_API_KEY}"
  },
  "tools": {
    "enabled": ["web_search", "calculator"]
  }
}
```

## Security Notes

- **API Key Storage**: Your API key is only stored in your browser's localStorage and is never sent to external servers (when using the standalone playground)
- **Production Deployment**: In production, use environment variables for API keys and enable authentication
- **CORS**: The playground allows all origins by default; restrict this in production

## Agent Types

- **react**: ReAct (Reasoning + Acting) agent with tool use
- **deer**: DEER-Go enhanced reasoning agent
- **manus**: Autonomous execution agent
- **chain**: Simple chain-based conversation
- **graph**: Directed graph conversation
- **workflow**: Linear workflow execution

## License

Same as CoRag/Aetheris project.
