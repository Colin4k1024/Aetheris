# Sample Discord Post: LangGraph Channel

**Channel:** #langgraph or #rag-and-retrieval

**Post Type:** Discussion

**Message:**

Hey! Contributor on **CoRag/Aetheris** here. We build durable RAG infrastructure for enterprise (billions of queries, compliance requirements, etc.).

I've been geeking out on LangGraph's checkpointing approach. Question for anyone who's tried both:

**How does LangGraph handle recovery from mid-execution failures?**

In our system, if a workflow crashes at step 47 of 50, we replay from the last checkpoint. But I'm curious:
- Does LangGraph guarantee exactly-once semantics?
- Can you replay a failed subgraph without re-executing parent nodes?
- How do you handle compensating actions (e.g., rolling back a partially completed multi-step process)?

Also, we'd love to share what we've learned about:
- Hybrid retrieval (vector + BM25 + knowledge graph)  
- RAG evaluation without human labels
- Production observability for RAG at scale

Would anyone be interested in a joint community discussion about "RAG in production"? Happy to help organize.

---

**Engagement Tips:**
- Wait for responses before diving into feature promotion
- Genuinely answer follow-up questions
- Offer value before asking questions back

**Shorter version for quick questions:**

> Quick Q: How does LangGraph recover from failures mid-workflow? We're building similar durable execution in CoRag and curious about your checkpointing model vs our event-sourcing approach.
