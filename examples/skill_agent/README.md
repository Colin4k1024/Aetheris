# Skill Agent Example

This example demonstrates how to use eino Skill capability in Aetheris framework.

## Overview

Skill is a folder containing instructions, scripts, and resources that Agents can discover and use on-demand to extend their capabilities. The core of a Skill is the `SKILL.md` file, which contains metadata (at least `name` and `description`) and instructions for the Agent to execute specific tasks.

This example shows:
- Defining Skill directory structure (with SKILL.md)
- Using eino skill backend for skill management
- Progressive loading mechanism (discovery → activation → execution)
- Multiple context modes (inline, fork, isolate)

## Skill Structure

```
skills/
├── pdf_analyzer/
│   ├── SKILL.md          # Required: skill definition
│   ├── scripts/          # Optional: executable scripts
│   │   └── analyze.py
│   ├── references/      # Optional: reference documents
│   └── assets/          # Optional: resource files
└── log_analyzer/
    ├── SKILL.md
    └── scripts/
        └── parse.py
```

## SKILL.md Format

```yaml
---
name: skill_name
description: Skill description
context: fork  # or inline, isolate
agent: agent_name  # Optional
model: model_name  # Optional
---
# Skill Instructions

Your skill instructions and execution steps...
```

## Context Modes

| Mode | Description |
| ---- |-------------|
| **inline** (default) | Skill content returned directly as tool result, processed by current Agent |
| **fork** | Create new Agent with copied conversation history, execute Skill independently |
| **isolate** | Create new Agent with isolated context (only Skill content), execute independently |

## Progressive Loading

1. **Discovery**: When Agent starts, only loads each Skill's name and description - enough to determine when a Skill might be needed

2. **Activation**: When a task matches a Skill's description, Agent loads the complete SKILL.md content into context

3. **Execution**: Agent follows the Skill instructions to execute the task, can also load other files or execute bundled scripts as needed

## Running

```bash
# Run the example
go run ./examples/skill_agent/main.go
```

## Code Structure

```go
// 1. Create filesystem backend
fsBackend, err := eino.NewLocalFileBackend(ctx, &eino.LocalFileBackendConfig{
    BaseDir: skillsDir,
})

// 2. Create skill backend
skillBackend, err := eino.NewSkillBackendFromFilesystem(ctx, fsBackend, skillsDir)

// 3. List all available skills
skills, err := skillBackend.List(ctx)

// 4. Get specific skill
skill, err := skillBackend.Get(ctx, "pdf_analyzer")
```

## Integration with eino Agent

To create a full Skill-enabled Agent:

```go
import (
    "github.com/cloudwego/eino/adk/middlewares/skill"
    "github.com/cloudwego/eino/adk"
)

skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
    Backend:        skillBackend,
    SkillToolName:  ptr.String("skill"),
})

agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "DocumentAssistant",
    Instruction: "You are a helpful assistant.",
    Model:       chatModel,
    Handlers:    []adk.ChatModelAgentMiddleware{skillMiddleware},
})
```

## Related Documentation

- [Eino Skill Documentation](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_skill/)
