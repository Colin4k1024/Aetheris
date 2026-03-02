# Awesome Lists Submission Guide

## awesome-go

**Repository**: https://github.com/avelino/awesome-go

**How to submit**:
1. Fork awesome-go
2. Add Aetheris to the appropriate category (Agent Systems)
3. Create PR

**Suggested location**: Add under "Agent systems" or "Machine Learning" section

**PR Title**: Add Aetheris - Agent Hosting Runtime

**PR Description**:
```markdown
**Aetheris** is an execution runtime for intelligent agents - "Temporal for Agents".

- Provides durable, replayable, and observable agent execution
- Event-sourced execution with full audit trail
- Human-in-the-loop approval workflows
- Native adapters for LangGraph, AutoGen, and CrewAI

Website: https://github.com/Colin4k1024/Aetheris
```

---

## awesome-ai-agents

**Repository**: https://github.com/e2b-dev/awesome-ai-agents

**How to submit**:
1. Fork the repo
2. Add to the "Infrastructure / Agent Hosting" section
3. Create PR

**Suggested location**: Infrastructure > Agent Hosting

**PR Title**: Add Aetheris - Production-grade Agent Runtime

**PR Description**:
```markdown
[Aetheris](https://github.com/Colin4k1024/Aetheris) - Agent hosting runtime with event-sourced execution, crash recovery, and human-in-the-loop support. Supports LangGraph, AutoGen, and CrewAI adapters.
```

---

## Other Lists to Consider

1. **awesome-selfhosted** - https://github.com/awesome-selfhosted/awesome-selfhosted
2. **awesome-go** - https://github.com/avelino/awesome-go
3. **awesome-ai-agents** - https://github.com/e2b-dev/awesome-ai-agents
4. **awesome-huggingface** - (if applicable)

---

## Manual Submission Commands

```bash
# Clone awesome-go
git clone https://github.com/avelino/awesome-go.git
cd awesome-go

# Create branch
git checkout -b add-aetheris

# Edit README.md - add to appropriate section
# Then commit and push
git add .
git commit -m "Add Aetheris - Agent Hosting Runtime"
git push origin add-aetheris

# Open PR
gh pr create --title "Add Aetheris" --body-file ../aetheris-pr-description.md
```
