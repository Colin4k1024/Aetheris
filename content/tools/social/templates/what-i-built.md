# "What I Built This Week" Thread Template

Use for community showcase threads or personal project updates.

## Template Structure

```
Tweet 1: "What I built this week" hook
Tweet 2: The problem / Why it matters  
Tweet 3-4: How it works (architecture/approach)
Tweet 5-6: Key code or demo
Tweet 7: Results / What it enables
Tweet 8: CTA / What's next
```

## Example Thread

---

**Tweet 1:**
> 🧵 What I built this week: an AI customer support agent that never loses a ticket.
> 
> Let me show you how @AetherisAI made this possible 👇

---

**Tweet 2:**
> **The Problem:**
> 
> Traditional chatbots lose context when:
> - They crash mid-conversation
> - The user goes offline and returns
> - A human needs to take over
> 
> Every lost session = frustrated customers = lost business

---

**Tweet 3:**
> **My Solution:**
> 
> A durable agent that:
> - Pauses for human approval on refunds > $100
> - Checkpoints every conversation turn
> - Resumes from exactly where it left off
> 
> Zero lost tickets. Zero restart-from-scratch.

---

**Tweet 4:**
> **Architecture:**
> ```
> User → Agent (Aetheris) → LLM
>              ↓
>         Checkpoint Store
>              ↓
> Human Approval (if needed)
>              ↓
>         Resume / Complete
> ```
> 
> The agent runs in Aetheris. I handle the UI.

---

**Tweet 5:**
> **The key code:**
> ```go
> // Agent pauses for approval
> if order.Value > 100 {
>     job.Pause(ctx, "approval_needed")
>     // Send to human queue
>     notifyHuman(ctx, order)
> }
> 
> // When approved, resume exactly here
> job.Resume(ctx, approvalResult)
> ```

---

**Tweet 6:**
> **What Aetheris handles for me:**
> 
> ✅ At-most-once tool calls (no duplicate refunds!)
> ✅ Crash recovery (server restarts don't lose state)
> ✅ Full audit trail (compliance ready)
> 
> I just write the agent logic.

---

**Tweet 7:**
> **Results:**
> 
> - 🚀 Response time: < 2 seconds
> - 💰 Caught $3k in fraudulent requests (approval needed)
> - 📊 Customer satisfaction: +15%
> - 😴 I sleep at night knowing it won't lose tickets

---

**Tweet 8:**
> **What's next:**
> 
> Adding multi-language support and sentiment-aware escalation.
> 
> Want the full repo? Drop 👋 in replies.
> 
> #BuildInPublic #AI #Agents

---

## Fill-in-the-Blank Version

```
🧵 What I built this week: [Project Name]

[Hook - what problem does it solve?]

**The Problem:**
[What's broken / inefficient / missing]

**My Solution:**
[How you approached it with Aetheris]

**Architecture:**
[Simple diagram or description]

**Key Code:**
```[language]
[representative snippet]
```

**What Aetheris handles:**
- ✅ [Feature 1]
- ✅ [Feature 2]  
- ✅ [Feature 3]

**Results:**
- [Metric 1]
- [Metric 2]

**What's next:**
[Future improvements]

Code link / Demo: [URL]
Questions? Ask away 👇
```

## Tips for Community "What I Built" Posts

1. **Lead with value** — What problem does your build solve?
2. **Be specific** — Share real metrics and outcomes
3. **Credit contributors** — Mention anyone who helped
4. **Show code** — Real code builds credibility
5. **Invite questions** — End with an open invitation
6. **Share mistakes** — What didn't work at first adds authenticity

## Alternative Formats

### Short Version (3-4 tweets)
```
Tweet 1: Hook + one-liner description
Tweet 2: Problem → Solution (with Aetheris mention)
Tweet 3: Key result or demo
Tweet 4: CTA (code link, questions)
```

### Long Version (10+ tweets)
Add:
- Step-by-step implementation
- Comparison with alternatives tried
- Detailed metrics
- Screenshots of the working product
- Future roadmap
