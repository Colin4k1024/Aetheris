# Aetheris Workflow Templates

A collection of preset workflow templates for the Aetheris/CoRag agent framework. These templates are ready-to-use and can be customized for your specific use cases.

## Available Templates

### 1. [Customer Service Agent](./customer-service-agent/)

A multi-turn customer service agent with intelligent triage, specialized handlers, and human approval workflows.

**Features:**
- Automatic request classification (technical, billing, refund, general)
- Specialized handling for each request type
- Human approval for sensitive operations
- Escalation support for complex issues
- Conversation memory

**Use Cases:**
- Customer support automation
- Help desk systems
- Order management

---

### 2. [RAG Assistant](./rag-assistant/)

Retrieval-Augmented Generation workflow combining vector search with LLM generation for accurate, cited answers.

**Features:**
- Intelligent query processing and rewrite
- Multi-source retrieval (vector DB + web)
- Re-ranking and filtering
- Context synthesis
- Citation generation
- Quality assurance

**Use Cases:**
- Knowledge base Q&A
- Document question answering
- Technical documentation search
- Research assistance

---

### 3. [Multi-Agent Debate](./multi-agent-debate/)

Structured debate system with multiple agents arguing different perspectives, moderated by a neutral agent.

**Features:**
- Multiple perspectives (advocate, opponent, moderator)
- Structured rounds (opening, rebuttal, cross-examination)
- Evidence-based arguments
- Real-time synthesis and scoring
- Final judgment with reasoning

**Use Cases:**
- Decision making with multiple perspectives
- Policy analysis
- Technology assessment
- Ethical dilemma exploration

---

### 4. [Autonomous Researcher](./autonomous-researcher/)

Deep research agent that autonomously explores topics, verifies findings, and produces comprehensive reports.

**Features:**
- Autonomous topic exploration
- Multi-source information gathering
- Deep dive research on knowledge areas
- Verification and fact-checking
- Multiple deliverable formats (report, summary, presentation)
- Confidence scoring

**Use Cases:**
- Market research
- Literature reviews
- Competitive analysis
- Technical assessments
- Investment research

---

## Quick Start

### Using a Template

1. **Choose a template** that matches your use case
2. **Copy to your project**:
   ```bash
   cp -r templates/customer-service-agent/ ./my-project/
   ```
3. **Configure agents** by editing `agents.yaml`
4. **Run the workflow**:
   ```bash
   go run ./cmd/worker --config configs/agents.yaml
   ```

### Creating from Template

```bash
# Create a new project from template
mkdir my-agent && cd my-agent
cp ../templates/rag-assistant/* .

# Customize for your needs
vim agents.yaml
vim workflow.go
```

## Template Structure

Each template follows a consistent structure:

```
template-name/
├── agents.yaml    # Agent configuration (tools, prompts, settings)
├── workflow.go    # Workflow definition using TaskGraph API
└── README.md      # Documentation and usage guide
```

### agents.yaml

The agent configuration file defines:
- **Agent types**: react, chain, workflow, etc.
- **System prompts**: Agent behavior and guidelines
- **Tools**: Available tools for each agent
- **LLM settings**: Model and provider configuration

### workflow.go

The workflow definition includes:
- **TaskGraph structure**: Nodes and edges defining the flow
- **Node types**: tool, llm, wait, approval, etc.
- **Input/Output types**: Typed data structures
- **Demo code**: Example usage

## Common Patterns

### Adding Custom Tools

```yaml
# In agents.yaml
tools:
  enabled:
    - "web_search"
    - "retriever"
    - "your_custom_tool"  # Add here
```

### Customizing Agent Behavior

```yaml
# In agents.yaml
agents:
  my_agent:
    type: "react"
    max_iterations: 20  # Increase for complex tasks
    system_prompt: |
      Your custom prompt here...
```

### Extending Workflows

```go
// In workflow.go
Nodes: []planner.TaskNode{
    // Add custom nodes
    {
        ID:   "my_node",
        Type: planner.NodeLLM,
        Config: map[string]any{
            "prompt": "Your custom logic...",
        },
    },
}
```

## Configuration

### LLM Configuration

```yaml
llm:
  provider: "openai"  # openai, anthropic, ollama, etc.
  model: "gpt-4o-mini"
  api_key: "${OPENAI_API_KEY}"  # Use environment variable
```

### Tool Configuration

```yaml
tools:
  enabled:
    - "web_search"
    - "retriever"
    - "calculator"

  web_search:
    api_key: "${SEARCH_API_KEY}"
    engine: "duckduckgo"  # or "google", "bing"

  retriever:
    collection: "my_knowledge_base"
    top_k: 10
```

## Workflow Node Types

| Type | Description | Use Case |
|------|-------------|----------|
| `tool` | Calls a tool (retriever, web_search, etc.) | Action execution |
| `llm` | LLM call for reasoning/generation | Analysis, synthesis |
| `wait` | Pauses for external input | Human approval, webhooks |
| `approval` | Specialized wait for approval | Sensitive operations |
| `condition` | Waits for a condition | Event-driven flows |

## Examples

### Basic Query

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "What is the return policy?"}'
```

### Research Request

```bash
curl -X POST http://localhost:8080/research \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "AI trends in healthcare",
    "depth": "medium",
    "deliverables": ["report", "summary"]
  }'
```

### Debate

```bash
curl -X POST http://localhost:8080/debate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "Should AI be regulated?",
    "rounds": 2
  }'
```

## Contributing

To add a new template:

1. Create a new directory under `templates/`
2. Add `agents.yaml`, `workflow.go`, and `README.md`
3. Update this README with your template
4. Submit a pull request

## License

Apache 2.0 - See LICENSE in the Aetheris repository
