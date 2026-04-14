# AI Customer Service Bot Demo

This demo showcases an AI customer service bot built with the Aetheris/CoRag framework, featuring multi-turn conversation, human review/approval mechanism, conversation history persistence, and visual execution tracing.

## Features

- **Multi-turn Conversation**: The bot maintains context across multiple user interactions
- **Human Review/Approval**: Certain operations require human approval before execution
- **Conversation History Persistence**: Conversations are stored in JSON format for persistence
- **Visual Execution Tracing**: Step-by-step execution is logged for debugging and monitoring

## Prerequisites

- Go 1.26+
- OPENAI_API_KEY environment variable (or use Ollama as alternative)

## Quick Start

```bash
# Set your API key
export OPENAI_API_KEY=your_key_here

# Run the demo
go run .

# Or with Ollama (local LLM)
OLLAMA_MODEL=llama3 OLLAMA_BASE_URL=http://localhost:11434 go run .
```

## Project Structure

- `main.go` - Entry point, sets up the bot and starts the conversation loop
- `bot.go` - Bot logic including multi-turn conversation handling
- `store.go` - Conversation history persistence (JSON file storage)
- `human_review.go` - Human approval mechanism for sensitive operations

## Usage

The bot simulates a customer service scenario where:
1. User asks questions or makes requests
2. Bot analyzes the request
3. For sensitive operations (refunds, account changes), human approval is requested
4. Bot responds based on the outcome

## Example Interactions

```
=== AI Customer Service Bot ===
Bot: Hello! I'm your AI customer service assistant. How can I help you today?
You: I want to know my order status for order #12345
Bot: Let me check that for you... Your order #12345 is currently being processed and is expected to ship within 2 business days.

You: I'd like to return my recent purchase
Bot: I can help you with that. I'm requesting approval for a refund of $99.99.
[Waiting for human approval...]

=== Human Approval Required ===
Request ID: approval-xxx
Type: Refund Request
Details: Order #12345, Amount: $99.99
Description: Customer requested return

Approve this request? (yes/no): yes
Human approved!

Bot: Great! Your refund has been approved and will be processed within 3-5 business days. You'll receive a confirmation email shortly.
```

## Configuration

The bot can be configured via environment variables:
- `OPENAI_API_KEY` - OpenAI API key (required for OpenAI models)
- `OLLAMA_MODEL` - Ollama model name (default: llama3)
- `OLLAMA_BASE_URL` - Ollama base URL (default: http://localhost:11434)
- `STORE_FILE` - Path to conversation history file (default: conversations.json)
- `LOG_LEVEL` - Logging level (default: info)
