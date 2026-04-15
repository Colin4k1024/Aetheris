# Workflow Examples: Comparing Approaches

This document shows equivalent workflow patterns in CoRag and LangGraph to help developers understand both systems.

## Example 1: Basic RAG Pipeline

### CoRag (Go)

```go
func BasicRAG(ctx context.Context, query string) (string, error) {
    engine := eino.NewEngine()
    
    workflow := workflow.NewBuilder("basic-rag").
        AddStep("retrieve", func(ctx context.Context, input string) ([]string, error) {
            docs, err := vectorStore.SimilaritySearch(ctx, input, 5)
            return docs, err
        }).
        AddStep("generate", func(ctx context.Context, docs []string) (string, error) {
            prompt := fmt.Sprintf("Context: %s\n\nQuestion: %s", 
                strings.Join(docs, "\n"), input)
            return llm.Generate(ctx, prompt)
        }).
        Build()
    
    return engine.Execute(ctx, workflow, query)
}
```

### LangGraph (Python)

```python
def basic_rag_graph():
    workflow = StateGraph(RAGState)
    
    workflow.add_node("retrieve", retrieve_node)
    workflow.add_node("generate", generate_node)
    
    workflow.set_entry_point("retrieve")
    workflow.add_edge("retrieve", "generate")
    workflow.set_finish_point("generate")
    
    return workflow.compile()

# Usage
app = basic_rag_graph()
result = app.invoke({"question": "What is X?"})
```

---

## Example 2: RAG with Human-in-the-Loop

### CoRag (Go)

```go
func RAGWithHumanReview(ctx context.Context, query string) (string, error) {
    workflow := workflow.NewBuilder("rag-review").
        AddStep("retrieve", retrieveDocuments).
        AddStep("check", confidenceCheck).
        AddConditionalStep(
            "human_review", 
            func(ctx context.Context, state *State) (*Response, error) {
                // Pause workflow, notify human
                review, err := humanReviewer.Submit(ctx, &ReviewRequest{
                    Query:  query,
                    Docs:   state.Documents,
                    Reason: "Low confidence: " + state.Reason,
                })
                return review.Response, err
            },
            // Only execute if confidence < 0.7
            when = func(state *State) bool {
                return state.Confidence < 0.7
            },
        ).
        AddStep("generate", generateResponse).
        Build()
    
    result, err := engine.Execute(ctx, workflow, query)
    if err != nil {
        if errors.Is(err, eino.ErrPaused) {
            // Workflow paused, waiting for human
            return "", err
        }
    }
    return result, nil
}
```

### LangGraph (Python)

```python
from langgraph.types import interrupt

def retrieve_with_check(state: RAGState):
    docs = vectorstore.similarity_search(state["question"])
    confidence = calculate_confidence(docs)
    
    if confidence < 0.7:
        # This will pause execution and wait for human input
        human_input = interrupt({
            "question": state["question"],
            "docs": docs,
            "reason": f"Confidence {confidence} below threshold"
        })
        return {"docs": human_input["reviewed_docs"]}
    
    return {"docs": docs}

workflow = StateGraph(RAGState)
workflow.add_node("retrieve", retrieve_with_check)
workflow.add_node("generate", generate_node)
workflow.add_edge("retrieve", "generate")
```

---

## Example 3: Multi-Strategy Retrieval

### CoRag (Go)

```go
func HybridRetrieval(ctx context.Context, query string) (*Result, error) {
    engine := eino.NewEngine()
    
    workflow := workflow.NewBuilder("hybrid-retrieval").
        // Run three retrievers in parallel
        AddParallelStep("parallel_retrieve",
            []workflow.Step{
                {Name: "dense", Fn: denseRetrieval},
                {Name: "sparse", Fn: sparseRetrieval},
                {Name: "graph", Fn: knowledgeGraphRetrieval},
            },
            eino.ParallelConfig{MaxConcurrency: 3},
        ).
        // Fuse results using RRF
        AddStep("fuse", func(ctx context.Context, results [][]RetrievedDoc) ([]RetrievedDoc, error) {
            return rrf.Fuse(results, rrf.Config{K: 60})
        }).
        // Rerank and take top 5
        AddStep("rerank", func(ctx context.Context, docs []RetrievedDoc) ([]RetrievedDoc, error) {
            return crossEncoder.Rerank(query, docs, 5)
        }).
        Build()
    
    return engine.Execute(ctx, workflow, query)
}
```

### LangGraph (Python)

```python
from typing import TypedDict, List

class HybridState(TypedDict):
    question: str
    dense_docs: List[Document]
    sparse_docs: List[Document]
    graph_docs: List[Document]
    fused_docs: List[Document]

def parallel_retrieve(state: HybridState):
    # LangGraph doesn't have native parallel, but can use Send
    return [
        Send("dense_retrieve", state),
        Send("sparse_retrieve", state),
        Send("graph_retrieve", state),
    ]

def dense_retrieve(state: HybridState):
    return {"dense_docs": vectorstore.similarity_search(state["question"])}

def sparse_retrieve(state: HybridState):
    return {"sparse_docs": bm25.search(state["question"])}

def fuse_results(state: HybridState):
    all_docs = state["dense_docs"] + state["sparse_docs"] + state["graph_docs"]
    return {"fused_docs": rrf_fusion(all_docs)}
```

---

## Example 4: Multi-Agent Collaboration

### CoRag (Go)

```go
func MultiAgentResearch(ctx context.Context, topic string) (*ResearchReport, error) {
    workflow := workflow.NewBuilder("research-team").
        // Supervisor dispatches to specialized agents
        AddStep("supervisor", supervisor.Dispatch(
            []workflow.Agent{
                {Name: "web-search", Fn: webSearchAgent},
                {Name: "academic", Fn: academicSearchAgent},
                {Name: "internal", Fn: internalKBAgent},
            },
            // Route based on topic classification
            router = classifyTopic,
        )).
        // Synthesize findings
        AddStep("synthesize", func(ctx context.Context, findings []AgentFinding) (*Report, error) {
            return synthesizer.Synthesize(ctx, findings)
        }).
        // Quality review
        AddStep("review", qualityReview).
        AddConditionalStep(
            "revise",
            revisionAgent,
            when = func(s *State) bool { return s.ReviewScore < 0.8 },
        ).
        Build()
    
    return engine.Execute(ctx, workflow, topic)
}
```

### LangGraph (Python)

```python
from langgraph.supervisor import supervisor_chain

research_team = supervisor_chain(
    [web_search_agent, academic_agent, internal_kb_agent],
    supervisor_name="supervisor"
)

def classify_and_route(state: ResearchState) -> str:
    topic = classify(state["topic"])
    if "technical" in topic:
        return "academic_agent"
    elif "internal" in topic:
        return "internal_kb_agent"
    else:
        return "web_search_agent"

workflow = StateGraph(ResearchState)
workflow.add_node("supervisor", research_team)
workflow.add_node("synthesize", synthesize_node)
workflow.add_node("review", review_node)
workflow.add_node("revise", revise_node)

workflow.add_edge("supervisor", "synthesize")
workflow.add_conditional_edges(
    "review",
    lambda state: "revise" if state["score"] < 0.8 else "finish"
)
```

---

## Example 5: Streaming Responses

### CoRag (Go)

```go
func StreamingRAG(ctx context.Context, query string) error {
    workflow := workflow.NewBuilder("streaming-rag").
        AddStep("retrieve", retrieveDocuments).
        AddStep("generate", func(ctx context.Context, docs []string, stream chan string) error {
            prompt := buildPrompt(docs, query)
            return llm.GenerateStream(ctx, prompt, stream)
        }).
        Build()
    
    // stream is a channel that receives tokens as they're generated
    return engine.ExecuteStream(ctx, workflow, query, stream)
}

// Client usage
stream := make(chan string)
go func() {
    engine.ExecuteStream(ctx, query, stream)
}()
for token := range stream {
    fmt.Print(token)
}
```

### LangGraph (Python)

```python
from langgraph.types import Stream

def generate_with_stream(state: RAGState, stream: Stream):
    prompt = build_prompt(state["docs"], state["question"])
    for chunk in llm.stream(prompt):
        stream.write(chunk)

workflow = StateGraph(RAGState)
workflow.add_node("generate", generate_with_stream)

app = workflow.compile()

# Usage
for chunk in app.stream({"question": "..."}, stream=True):
    print(chunk, end="", flush=True)
```

---

## Choosing the Right Pattern

| Pattern | CoRag Strength | LangGraph Strength |
|---------|---------------|-------------------|
| Simple RAG | Less boilerplate | Very clean |
| Complex retrieval | Native RRF fusion | More manual |
| Human-in-loop | Native pause/resume | interrupt() |
| Multi-agent | Supervisor pattern | Built-in supervisor |
| Streaming | Channel-based | Async iterator |
| Durability | Event sourcing | Checkpointing |

---

## Migration Paths

If you're moving from one to the other:

**LangGraph → CoRag:**
- Convert Python nodes to Go functions
- Replace state dicts with Go structs
- Map conditional edges to `AddConditionalStep`

**CoRag → LangGraph:**
- Convert workflow builder to StateGraph
- Replace event sourcing with checkpointing
- Map steps to node functions

Both approaches are valid—the best choice depends on your team's language preferences and specific requirements.

---

*These examples are meant for educational purposes. Production code would include proper error handling, logging, and configuration.*
