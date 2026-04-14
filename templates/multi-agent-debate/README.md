# Multi-Agent Debate Template

A structured multi-agent debate system where different agents argue different perspectives, with a moderator synthesizing and rendering judgment.

## Features

- **Multiple Perspectives**: Separate agents for advocate, opponent, and moderator
- **Structured Rounds**: Opening, rebuttal, cross-examination, and closing phases
- **Evidence-Based**: Integrates research and retrieval for factual arguments
- **Moderation**: Neutral synthesis and scoring
- **Verdict**: Clear judgment with reasoning

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      DEBATE WORKFLOW                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────┐                                                │
│  │  Setup  │ ◄── Define motion, scope, rules               │
│  └────┬────┘                                                │
│       │                                                       │
│       ├────────────────┬────────────────┐                    │
│       ▼                 ▼                ▼                    │
│  ┌─────────┐     ┌─────────┐     ┌─────────────┐          │
│  │Research │     │Research │     │   Research  │          │
│  │   For   │     │ Against │     │    Facts    │          │
│  └────┬────┘     └────┬────┘     └──────┬──────┘          │
│       │                 │                 │                   │
│       └────────────────┬┴────────────────┘                   │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │              OPENING ARGUMENTS                │           │
│  │  ┌─────────────┐      ┌─────────────┐      │           │
│  │  │    FOR      │      │   AGAINST   │      │           │
│  │  └─────────────┘      └─────────────┘      │           │
│  └──────────────────────────────────────────────┘           │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │               REBUTTALS                      │           │
│  │  ┌─────────────┐      ┌─────────────┐      │           │
│  │  │    FOR      │      │   AGAINST   │      │           │
│  │  └─────────────┘      └─────────────┘      │           │
│  └──────────────────────────────────────────────┘           │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │           CROSS-EXAMINATION                  │           │
│  │  Q: FOR → A: AGAINST    Q: AGAINST → A: FOR │           │
│  └──────────────────────────────────────────────┘           │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │         MODERATOR SYNTHESIS                 │           │
│  │  - Points of agreement                      │           │
│  │  - Key disagreements                        │           │
│  │  - Scoring (1-10)                          │           │
│  └──────────────────────────────────────────────┘           │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │            CLOSING STATEMENTS               │           │
│  │  ┌─────────────┐      ┌─────────────┐      │           │
│  │  │    FOR      │      │   AGAINST   │      │           │
│  │  └─────────────┘      └─────────────┘      │           │
│  └──────────────────────────────────────────────┘           │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │              FINAL JUDGMENT                 │           │
│  │  - Verdict (Advocate/Opponent/Tie)        │           │
│  │  - Reasoning                                │           │
│  │  - Highlights                               │           │
│  └──────────────────────────────────────────────┘           │
│                        │                                      │
│                        ▼                                      │
│  ┌──────────────────────────────────────────────┐           │
│  │              FINAL REPORT                    │           │
│  │  - Executive summary                        │           │
│  │  - Full analysis                             │           │
│  └──────────────────────────────────────────────┘           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Files

- `agents.yaml` - Agent configuration for advocate, opponent, moderator, and researcher
- `workflow.go` - Workflow definition using TaskGraph API

## Quick Start

### 1. Setup

```bash
# Copy the template
cp -r templates/multi-agent-debate/ ./my-debate/

# Copy agents.yaml
cp agents.yaml /path/to/configs/agents.yaml
```

### 2. Configure

```bash
# Set environment variables
export OPENAI_API_KEY=your-api-key
```

### 3. Run

```bash
# Run the debate workflow
go run templates/multi-agent-debate/workflow.go

# Or via API
curl -X POST http://localhost:8080/debate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "Should AI be regulated?",
    "rounds": 2
  }'
```

## Workflow Phases

### 1. Setup

The moderator:
- Defines key terms and scope
- Identifies stakeholders
- Sets debate rules
- Creates a balanced motion statement

### 2. Research (Parallel)

Three parallel research tasks:
- **Research For**: Evidence supporting the motion
- **Research Against**: Evidence opposing the motion
- **Research Facts**: Neutral facts and statistics

### 3. Opening Arguments

Both agents present their initial positions:
- Clear thesis statement
- 3-5 supporting arguments
- Evidence citations
- Anticipation of counterarguments

### 4. Rebuttals

Direct responses to the opponent's arguments:
- Address key objections
- Refute weak arguments
- Reinforce strong points
- New evidence if needed

### 5. Cross-Examination

Direct questioning between agents:
- Each agent asks one critical question
- Direct, honest responses required
- Challenges opposing position

### 6. Moderator Synthesis

Neutral analysis by the moderator:
- Points of agreement
- Key disagreements
- Scoring (1-10 for logic, evidence, reasoning)
- Fair summary

### 7. Closing Statements

Final compelling arguments:
- 2-3 paragraphs each
- Reinforce strongest points
- Acknowledge opponent's valid points

### 8. Final Judgment

Moderator delivers verdict:
- Winner declared
- Reasoning explained
- Highlights for each side

## Example Debates

### Policy Debate

```
Topic: "Should remote work become the default for tech companies?"

Advocate: "Remote work increases productivity, reduces costs, and improves work-life balance..."
Opponent: "Remote work harms collaboration, mentoring, and company culture..."
[Full debate...]
Moderator: "Both sides present valid points. Slight edge to opponent on collaboration concerns."
Verdict: "Opponent wins narrowly"
```

### Technology Assessment

```
Topic: "Should AI coding assistants be allowed in coding interviews?"

Advocate: "They reflect real-world tool usage and assess problem-solving..."
Opponent: "They bypass the actual skill being tested..."
[Full debate...]
Verdict: "Advocate wins - tools evolve and adaptation is necessary"
```

### Ethical Dilemma

```
Topic: "Should autonomous vehicles be programmed to sacrifice passengers for pedestrians?"

[This generates particularly nuanced multi-round debates]
Verdict: "Tie - more technical and philosophical discussion needed"
```

## Customization

### Custom Debate Topics

```go
{
    ID:   "setup",
    Type: planner.NodeLLM,
    Config: map[string]any{
        "prompt": `Setup debate for topic: {{input.topic}}
            Define motion, scope, and rules...`,
    },
}
```

### Additional Agents

```go
// Add a third party agent
{
    ID:   "third_party",
    Type: planner.NodeLLM,
    Config: map[string]any{
        "prompt": "You represent a neutral third perspective...",
    },
}
```

### Scoring Criteria

```go
"prompt": `Score the arguments:
    - Logic (0-10): Soundness of reasoning
    - Evidence (0-10): Quality of citations
    - Persuasion (0-10): How compelling
    - Civility (0-10): Professional tone`
```

## Testing

```bash
# Run the workflow demo
go run templates/multi-agent-debate/workflow.go

# Test specific debate
go test -run TestDebateWorkflow -v ./...
```

## Production Considerations

- **Rate Limits**: Multiple LLM calls per debate - manage API quotas
- **Cost**: Complex debates with many rounds can be expensive
- **Latency**: Full debates may take several minutes
- **Moderation Quality**: Ensure moderator remains neutral

## Related Templates

- **autonomous-researcher**: For deep research that may inform debates
- **rag-assistant**: For evidence retrieval during research phase

## License

Apache 2.0 - See LICENSE in the Aetheris repository
