# Eino Stateful Agent Example

This example demonstrates a stateful agent pattern with conversation history and checkpoint mechanism.

## Overview

This example shows:
- Session state management (conversation history)
- Context preservation across multiple turns
- Checkpoint mechanism implementation
- Integration with CoRag persistence

## Running

```bash
go run ./examples/eino_stateful/main.go
```

## Key Concepts

### Session State

```go
type SessionState struct {
    Messages  []Message       // Conversation history
    Variables map[string]any  // State variables
    CheckpointID string       // Checkpoint ID for persistence
}
```

### Checkpoint Mechanism

The stateful agent demonstrates:
1. Saving conversation state after each interaction
2. Restoring state from checkpoints
3. Maintaining context across multiple turns

```go
// Save checkpoint
state.CheckpointID = saveCheckpoint(state)

// Restore from checkpoint
state := restoreCheckpoint(checkpointID)
```

## Use Cases

- Multi-turn conversations with memory
- Long-running agents with state persistence
- Resume after crash scenarios
- Human-in-the-loop with state preservation

## Related Documentation

- [Eino Examples Adapter](../docs/adapters/eino-examples.md)
- [Checkpoint Documentation](../docs/guides/getting-started-agents.md)
