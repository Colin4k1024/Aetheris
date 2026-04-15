# Customer Service Agent Template

A complete, production-ready customer service agent workflow for Aetheris/CoRag.

## Features

- **Intelligent Triage**: Automatically classifies customer requests into categories (technical, billing, refund, general)
- **Specialized Handlers**: Dedicated agents for different request types
- **Human Approval Workflow**: Sensitive operations (refunds > $100, account changes) require human approval
- **Multi-turn Conversations**: Maintains context across customer interactions
- **Escalation Support**: Complex issues can be escalated to human agents

## Architecture

```
┌─────────────┐
│   Input     │
│ User Message│
└─────┬───────┘
      │
      ▼
┌─────────────┐
│   Triage    │ ◄── LLM classifies the request
└─────┬───────┘
      │
      ├──────────────────┬──────────────────┬──────────────────┐
      │                  │                  │                  │
      ▼                  ▼                  ▼                  ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Technical  │  │   Billing   │  │   Refund    │  │   General   │
│  Support    │  │             │  │   Request   │  │   Inquiry   │
└─────┬───────┘  └─────┬───────┘  └─────┬───────┘  └─────┬───────┘
      │                  │                  │                  │
      └──────────────────┴────────┬─────────┴──────────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │    Human Approval       │ ◄── For refunds & escalations
                    │    (if required)        │
                    └───────────┬─────────────┘
                                │
                                ▼
                    ┌─────────────────────────┐
                    │   Format Response       │
                    └───────────┬─────────────┘
                                │
                                ▼
                    ┌─────────────────────────┐
                    │    Send Response         │
                    └─────────────────────────┘
```

## Files

- `agents.yaml` - Agent configuration for customer service and approval agents
- `workflow.go` - Workflow definition using TaskGraph API

## Quick Start

### 1. Setup

```bash
# Copy the template to your project
cp -r templates/customer-service-agent/ ./my-customer-service/

# Copy agents.yaml to your configs
cp agents.yaml /path/to/your/configs/agents.yaml
```

### 2. Configure

Edit `agents.yaml` to customize:

```yaml
# Set your LLM provider
llm:
  provider: "openai"  # or "ollama", "anthropic", etc.
  model: "gpt-4o-mini"
  api_key: "${OPENAI_API_KEY}"

# Configure tools
tools:
  enabled:
    - "web_search"      # For product information lookup
    - "calculator"     # For billing calculations
    - "file_reader"    # For accessing customer records
```

### 3. Run

```bash
# Set environment variables
export OPENAI_API_KEY=your-api-key

# Run the worker
go run ./cmd/worker

# Or use the CLI
go run ./cmd/cli run --workflow customer_service --input '{"user_message": "Where is my order?"}'
```

## Workflow Details

### Triage Agent

The triage agent uses an LLM to classify incoming requests:

- **technical_support**: Product issues, bugs, how-to questions
- **billing**: Payment issues, invoices, pricing
- **refund**: Refund requests (triggers approval flow)
- **general**: Everything else

### Human Approval Flow

For sensitive operations:

1. Agent identifies need for approval (refund > $100, account changes)
2. Workflow pauses at `human_review` node
3. Manager receives notification
4. Manager approves/rejects via dashboard or API
5. Workflow resumes with decision

### Conversation Memory

The agent maintains conversation history for:
- Context across multiple messages
- Customer preference learning
- Issue resolution tracking

## Customization

### Adding New Categories

Modify the triage prompt in `workflow.go`:

```go
Config: map[string]any{
    "prompt": `Classify into: technical_support, billing, refund, general, SHIPPING, RETURNS...`,
}
```

### Custom Handlers

Add new handler nodes:

```go
{
    ID:   "shipping_handler",
    Type: planner.NodeLLM,
    Config: map[string]any{
        "prompt": "You are a shipping specialist...",
    },
},
```

### Tool Integration

Add custom tools in `agents.yaml`:

```yaml
tools:
  enabled:
    - "web_search"
    - "calculator"
    - "file_reader"
    - "http_request"
    - "your_custom_tool"  # Add your tool here
```

## Example Interactions

### Basic Inquiry

```
User: "What's the status of my order #12345?"
Agent: "Let me check that for you... Your order is currently being processed 
        and is expected to ship within 2 business days."
```

### Refund Request (with approval)

```
User: "I'd like to return my recent purchase"
Agent: "I can help you with that. I'm processing a refund of $99.99.
        This request requires manager approval."
        
[Waiting for approval...]

Manager: Approved
Agent: "Great! Your refund has been approved and will be processed 
        within 3-5 business days."
```

### Technical Support

```
User: "I'm getting an error when trying to login"
Agent: "I understand you're having login issues. Let me help troubleshoot:
        
        1. Have you tried resetting your password?
        2. Can you clear your browser cache and try again?
        3. Try using an incognito window
        
        If the issue persists, I can escalate to our technical team."
```

## Testing

```bash
# Run the workflow directly
go run templates/customer-service-agent/workflow.go

# This will output the workflow structure and save customer_service_workflow.json
```

## Production Checklist

- [ ] Configure production LLM with appropriate rate limits
- [ ] Set up monitoring and alerting for approval queues
- [ ] Configure backup/restore for conversation history
- [ ] Set up analytics for triage accuracy
- [ ] Configure notification channels for approvals (email, Slack, etc.)
- [ ] Set up human approval permissions and audit logging

## License

Apache 2.0 - See LICENSE in the Aetheris repository
