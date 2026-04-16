# RAG Assistant Template

A complete Retrieval-Augmented Generation (RAG) workflow for Aetheris/CoRag that combines vector search with LLM generation for accurate, cited answers.

## Features

- **Intelligent Query Processing**: Rewrites queries for optimal retrieval
- **Multi-source Retrieval**: Combines vector database search with web search
- **Re-ranking**: Filters and prioritizes retrieved documents
- **Context Synthesis**: Integrates information from multiple sources
- **Citation Generation**: Provides source citations in answers
- **Quality Assurance**: Validates generated answers before delivery

## Architecture

```
┌─────────────┐
│ User Query  │
└──────┬──────┘
       │
       ▼
┌──────────────────┐
│  Query Rewrite  │ ◄── LLM analyzes and optimizes the query
└────────┬────────┘
         │
         ├─────────────────────────────┐
         │                             │
         ▼                             ▼
┌──────────────────┐        ┌──────────────────┐
│ Vector Retrieval │        │   Web Search    │ ◄── Parallel retrieval
│   (Knowledge)    │        │   (Internet)    │
└────────┬────────┘        └────────┬────────┘
         │                             │
         └──────────┬──────────────────┘
                    │
                    ▼
         ┌──────────────────┐
         │   Re-rank &     │ ◄── LLM scores and filters
         │   Filter        │
         └────────┬────────┘
                  │
                  ▼
         ┌──────────────────┐
         │   Synthesize     │ ◄── Combine information
         └────────┬────────┘
                  │
                  ▼
         ┌──────────────────┐
         │ Generate Answer │ ◄── Create cited response
         └────────┬────────┘
                  │
                  ▼
         ┌──────────────────┐
         │  Quality Check  │ ◄── Validate accuracy
         └────────┬────────┘
                  │
                  ▼
         ┌──────────────────┐
         │    Finalize     │
         └──────────────────┘
```

## Files

- `agents.yaml` - Agent configuration for RAG and research assistants
- `workflow.go` - Workflow definition using TaskGraph API

## Quick Start

### 1. Setup

```bash
# Copy the template
cp -r templates/rag-assistant/ ./my-rag-assistant/

# Copy agents.yaml
cp agents.yaml /path/to/configs/agents.yaml
```

### 2. Configure

```bash
# Set environment variables
export OPENAI_API_KEY=your-api-key
export SEARCH_API_KEY=your-search-api-key  # Optional
```

### 3. Run

```bash
# Index your documents first
./scripts/index_documents.sh --path ./documents

# Start the worker
go run ./cmd/worker

# Query the RAG system
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "What is the return policy?"}'
```

## Workflow Details

### Query Rewrite

The query rewrite step:
- Identifies key concepts and entities
- Expands abbreviations and acronyms
- Clarifies ambiguous terms
- Generates alternative phrasings

### Multi-source Retrieval

Retrieves from multiple sources in parallel:
- **Vector Database**: Primary source for indexed documents
- **Web Search**: Additional context and updated information

### Re-ranking

The re-ranking step:
- Scores each document for relevance
- Removes duplicates
- Filters low-quality content
- Selects top 5 most relevant documents

### Quality Assurance

The quality check verifies:
- Factual accuracy against sources
- Completeness of answer
- Proper citation formatting
- Clarity and readability

## Customization

### Adding Custom Retrievers

```go
{
    ID:   "custom_retriever",
    Type: planner.NodeTool,
    Config: map[string]any{
        "tool_name": "your_custom_retriever",
        "top_k":     10,
    },
},
```

### Custom Preprocessing

Modify the query rewrite prompt:

```go
"prompt": `Your custom preprocessing instructions...
    Return JSON with your specified format.`,
```

### Post-processing

Add custom formatting:

```go
{
    ID:   "format_output",
    Type: planner.NodeLLM,
    Config: map[string]any{
        "prompt": `Format output as: markdown, JSON, HTML, etc.`,
    },
},
```

## Example Queries

### Basic Question

```
User: "What is the shipping policy?"
RAG: "Based on our documentation, standard shipping takes 3-5 business days.
     Expedited shipping (1-2 days) is available for an additional fee.
     [Source: Shipping Policy v2.3, Section 3.1]"
```

### Complex Research Query

```
User: "Compare our competitors' pricing for enterprise plans"
RAG: "Based on our research and market analysis:

     | Competitor | Enterprise Plan | Includes |
     |------------|-----------------|----------|
     | Competitor A | $99/mo | 10 users, 100GB |
     | Competitor B | $149/mo | 25 users, 500GB |

     Our plan at $199/mo includes unlimited users and 1TB storage.
     [Sources: Competitor Analysis Q1 2024, Pricing Report]"
```

### Multi-concept Query

```
User: "How do I set up SSO with Okta?"
RAG: "To configure SSO with Okta:

     1. In Okta Admin, go to Applications > Create App Integration
     2. Select SAML 2.0 as the sign-in method
     3. Enter your Aetheris SSO URL: https://app.aetheris.io/saml/callback
     4. Download the metadata XML
     5. Upload to Aetheris: Settings > Security > SSO

     Prerequisites:
     - Admin access to Okta
     - Enterprise plan subscription

     [Source: SSO Configuration Guide v1.2]"
```

## Testing

```bash
# Run the workflow demo
go run templates/rag-assistant/workflow.go

# Run RAG-specific tests
go test ./internal/pipeline/query/... -v

# Test with sample documents
./scripts/test_rag.sh --docs ./test_documents
```

## Performance Tuning

### Retrieval Parameters

```yaml
tools:
  retriever:
    top_k: 10           # Number of documents to retrieve
    similarity_threshold: 0.7  # Minimum similarity score
    max_context_length: 4000  # Context window limit
```

### Generation Parameters

```yaml
llm:
  model: "gpt-4o"       # Use more capable model for complex queries
  temperature: 0.3      # Lower for factual, higher for creative
  max_tokens: 2000      # Response length limit
```

## Production Checklist

- [ ] Set up vector database (Pinecone, Weaviate, Qdrant, etc.)
- [ ] Configure document preprocessing pipeline
- [ ] Set up incremental indexing for updates
- [ ] Configure monitoring for retrieval quality
- [ ] Set up feedback loop for continuous improvement
- [ ] Configure rate limiting and caching
- [ ] Set up analytics for query patterns

## Related Templates

- **autonomous-researcher**: Extends RAG with multi-document research capabilities
- **multi-agent-debate**: Uses multiple RAG agents for contrasting views

## License

Apache 2.0 - See LICENSE in the Aetheris repository
