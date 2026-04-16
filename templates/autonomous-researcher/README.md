# Autonomous Researcher Template

A powerful autonomous research agent that deeply explores topics, gathers information from multiple sources, verifies findings, and produces comprehensive reports.

## Features

- **Autonomous Exploration**: Agent decides research direction and depth
- **Multi-Source Research**: Combines web search, knowledge base, and documents
- **Deep Dives**: Multiple parallel research streams on different aspects
- **Verification**: Fact-checks and validates findings
- **Multiple Deliverables**: Report, summary, and presentation formats
- **Confidence Scoring**: Rates reliability of findings

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTONOMOUS RESEARCHER                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐                                               │
│  │ Analyze Topic│ ◄── LLM plans research approach               │
│  └──────┬───────┘                                               │
│         │                                                         │
│         ├─────────────────┬─────────────────┐                    │
│         ▼                 ▼                 ▼                    │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐           │
│  │ Search Web │   │ Search KB  │   │Load Docs   │           │
│  └─────┬──────┘   └─────┬──────┘   └─────┬──────┘           │
│        │                 │                 │                     │
│        └─────────────────┼─────────────────┘                    │
│                          ▼                                      │
│         ┌──────────────────────────────────────┐              │
│         │         DEEP DIVE RESEARCH           │              │
│         │  ┌─────────┐ ┌─────────┐ ┌─────────┐ │              │
│         │  │  Area 1 │ │  Area 2 │ │  Area 3 │ │              │
│         │  └────┬────┘ └────┬────┘ └────┬────┘ │              │
│         └───────┼────────────┼────────────┼──────┘              │
│                 └────────────┼────────────┘                     │
│                                ▼                                 │
│         ┌──────────────────────────────────────┐              │
│         │         VERIFICATION                 │              │
│         │  • Fact-check against sources        │              │
│         │  • Identify unsupported claims       │              │
│         │  • Rate confidence (0-1)            │              │
│         └──────────────────┬───────────────────┘              │
│                            ▼                                    │
│         ┌──────────────────────────────────────┐              │
│         │         ANALYSIS & SYNTHESIS        │              │
│         │  • Identify patterns                 │              │
│         │  • Draw conclusions                  │              │
│         │  • Note expert consensus            │              │
│         └──────────────────┬───────────────────┘              │
│                            ▼                                    │
│         ┌──────────────────────────────────────┐              │
│         │           REPORT DRAFTING            │              │
│         │  • Executive summary                 │              │
│         │  • Main report                       │              │
│         │  • Review and refine                 │              │
│         └──────────────────┬───────────────────┘              │
│                            ▼                                    │
│         ┌──────────────────────────────────────┐              │
│         │         DELIVERABLES                │              │
│         │  ┌──────────┐ ┌─────────┐           │              │
│         │  │  Report  │ │ Summary │           │              │
│         │  └──────────┘ └─────────┘           │              │
│         │  ┌──────────────┐                   │              │
│         │  │ Presentation │                   │              │
│         │  └──────────────┘                   │              │
│         └──────────────────────────────────────┘              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Files

- `agents.yaml` - Agent configuration for researcher, writer, and verifier
- `workflow.go` - Workflow definition using TaskGraph API

## Quick Start

### 1. Setup

```bash
# Copy the template
cp -r templates/autonomous-researcher/ ./my-researcher/

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
# Run the research workflow
go run templates/autonomous-researcher/workflow.go

# Or via API
curl -X POST http://localhost:8080/research \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "Impact of AI on software development",
    "depth": "deep",
    "audience": "technical executives",
    "deliverables": ["report", "summary", "presentation"]
  }'
```

## Workflow Phases

### 1. Topic Analysis

The researcher:
- Breaks down the topic into key questions
- Identifies required knowledge areas
- Determines information sources needed
- Creates a research plan

### 2. Multi-Source Research

Parallel searches across:
- **Web**: Latest information, news, articles
- **Knowledge Base**: Internal documents and data
- **Documents**: Loaded files and reports

### 3. Deep Dives

Multiple specialized research streams:
- Each focuses on a specific knowledge area
- Finds facts, statistics, expert opinions
- Cross-references with other sources

### 4. Verification

The verification agent:
- Fact-checks against multiple sources
- Identifies unsupported claims
- Assesses source reliability
- Rates confidence levels (0-1)

### 5. Analysis & Synthesis

Analysis of findings:
- Identifies common themes
- Notes trends and patterns
- Draws logical conclusions
- Identifies cause-effect relationships

### 6. Report Generation

Creates deliverables:
- **Executive Summary**: 1-2 paragraph overview
- **Main Report**: Comprehensive structured document
- **Presentation Outline**: Slide-by-slide format

## Example Research Topics

### Technology Assessment

```
Topic: "Impact of AI on software development"
Depth: deep
Audience: technical executives

Output:
- Comprehensive report on AI tooling in SDLC
- Analysis of productivity gains
- Risk assessment and recommendations
- Presentation for board meeting
```

### Market Research

```
Topic: "Electric vehicle market trends 2024-2028"
Depth: medium
Audience: investors

Output:
- Market size and growth projections
- Competitive landscape analysis
- Investment recommendations
- Risk factors assessment
```

### Scientific Literature Review

```
Topic: "Latest advances in mRNA therapeutics"
Depth: deep
Audience: medical researchers

Output:
- Comprehensive literature review
- Key papers and findings
- Research gaps and opportunities
- Comparison with traditional approaches
```

## Customization

### Custom Research Depth

```go
Depth: "deep"     // shallow, medium, deep
// shallow: Quick overview (5-10 sources)
// medium: Standard research (20-30 sources)
// deep: Comprehensive (50+ sources)
```

### Custom Focus Areas

```json
{
  "topic": "AI in healthcare",
  "focus_areas": [
    "diagnostics",
    "drug discovery",
    "patient monitoring"
  ]
}
```

### Custom Source Types

```json
{
  "topic": "Competitive analysis",
  "source_types": [
    "annual_reports",
    "news_articles",
    "industry_databases"
  ]
}
```

## Output Structure

### ResearchOutput

```json
{
  "title": "Research Report Title",
  "executive_summary": "2-paragraph summary...",
  "key_findings": [
    "Finding 1 with citation",
    "Finding 2 with citation"
  ],
  "report": "# Full Report...",
  "sources": [
    {"title": "...", "url": "...", "type": "web", "relevance": "high"}
  ],
  "confidence": 0.85,
  "gaps": ["Areas needing more research"],
  "next_steps": ["Recommended follow-up topics"]
}
```

## Testing

```bash
# Run the workflow demo
go run templates/autonomous-researcher/workflow.go

# Test specific research
go test -run TestResearchWorkflow -v ./...
```

## Performance Considerations

### Time

- **Shallow research**: 2-5 minutes
- **Medium research**: 10-20 minutes
- **Deep research**: 30-60 minutes

### Cost

- Web searches: ~$0.01-0.05 per search
- LLM calls: Depends on depth and sources
- Deep research: ~$2-10 per report

### Rate Limits

- Manage API quotas for long research sessions
- Consider caching frequent topics
- Use parallel searches strategically

## Production Checklist

- [ ] Set up search API with appropriate rate limits
- [ ] Configure document loading for internal sources
- [ ] Set up knowledge base indexing
- [ ] Configure monitoring for research quality
- [ ] Set up caching for repeated topics
- [ ] Configure analytics for research patterns
- [ ] Set up notification for completed research
- [ ] Configure storage for research outputs

## Related Templates

- **rag-assistant**: For targeted question answering
- **multi-agent-debate**: For exploring contrasting perspectives

## License

Apache 2.0 - See LICENSE in the Aetheris repository
