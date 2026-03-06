## Aetheris - Agent Hosting Runtime

[Website](https://github.com/Colin4k1024/Aetheris) |
[Documentation](https://github.com/Colin4k1024/Aetheris/tree/main/docs) |
[Examples](https://github.com/Colin4k1024/Aetheris/tree/main/examples)

> Production-grade runtime for AI agents - "Temporal for Agents"

Aetheris provides a durable, replayable, and observable environment where AI agents can plan, execute, pause, resume, and recover long-running tasks.

### Features

- **Event-sourced execution** - Full audit trail of all agent actions
- **Crash recovery** - Agents resume from checkpoint after failures
- **Human-in-the-loop** - Pause and wait for human approval
- **Framework adapters** - Native support for LangGraph, AutoGen, CrewAI

### Quick Start

```bash
# Install CLI
go install github.com/Colin4k1024/Aetheris/cmd/cli@latest

# Or quick install
curl -sSL https://raw.githubusercontent.com/Colin4k1024/Aetheris/main/scripts/install.sh | bash

# Run locally
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris
make run
```

### Go Version

Go 1.25+
