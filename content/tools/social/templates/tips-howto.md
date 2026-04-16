# Tips & How-To Tweet Templates

Use for educational content, quick tips, and practical guidance.

## Quick Tip Format (Single Tweet)

```
💡 Quick tip:

[Tip statement - actionable advice]

[Code/example if applicable]

#Tip #Aetheris
```

### Examples

---

**Single Tip Tweet:**
> 💡 Quick tip:
> 
> When using Aetheris checkpoints, name them descriptively.
> 
> ❌ `job.Checkpoint(ctx, "c1")`
> ✅ `job.Checkpoint(ctx, "before_payment_call")`
> 
> Your future self will thank you when debugging.

---

**Single Tip Tweet with Code:**
> 💡 How to pause an Aetheris job for human approval:
> 
> ```go
> if needsHumanReview(ctx, result) {
>     job.Pause(ctx, "awaiting_approval")
>     notifyApprover(ctx, result)
>     // Resume when approved
> }
> ```

---

## Thread: How-To Format

```
Tweet 1: Hook — "How to [do something] in X steps"
Tweet 2: Prerequisites or context
Tweet 3-5: Step-by-step instructions
Tweet 6: Common pitfalls to avoid
Tweet 7: CTA — Questions or share your result
```

### Example How-To Thread

---

**Tweet 1:**
> 🧵 How to add crash recovery to your Go agent in 5 minutes
> 
> (Even if it's not built with a "durable" framework)
> 
> Let's go 👇

---

**Tweet 2:**
> **Prerequisites:**
> 
> - Go 1.21+
> - An existing Go agent (or we'll use a simple example)
> - `go get github.com/aetheris/runtime-go`
> 
> That's it.

---

**Tweet 3:**
> **Step 1: Wrap your agent**
> 
> ```go
> import "github.com/aetheris/runtime-go"
> 
> // Wrap your existing agent
> durableAgent := aetheris.Wrap(myAgent)
> ```

---

**Tweet 4:**
> **Step 2: Submit jobs**
> 
> ```go
> job, err := runtime.Submit(ctx, durableAgent, input)
> if err != nil {
>     log.Fatal(err)
> }
> ```

---

**Tweet 5:**
> **Step 3: Handle failures**
> 
> ```go
> // Aetheris handles crashes automatically
> // But you can also manually recover:
> recovered, err := runtime.Resume(ctx, jobID)
> ```

---

**Tweet 6:**
> **Common mistakes:**
> 
> ❌ Don't checkpoint too frequently (overhead)
> ❌ Don't checkpoint sensitive data (logs everything)
> ✅ Name your checkpoints descriptively
> ✅ Use `Pause()` for human-in-the-loop

---

**Tweet 7:**
> **Try it:**
> 
> Full example repo: [link]
> 
> Questions? Drop them below 👇
> 
> #GoLang #AI #Tutorial

---

## Tip Series Formats

### "3 Things" Format
> 3 things you might not know about Aetheris:
> 
> 1️⃣ [Feature/fact]
> 2️⃣ [Feature/fact]
> 3️⃣ [Feature/fact]
> 
> [CTA or question]

### "Do/Don't" Format
> The right way to handle agent errors in Aetheris:
> 
> ✅ DO: [Best practice]
> ❌ DON'T: [Anti-pattern]
> 
> [Explanation]

### "Comparison" Format
> "Should I use checkpoints or event sourcing?"
> 
> Short answer: Use both.
> 
> Checkpoints = Fast recovery points
> Events = Full audit trail
> 
> Aetheris uses both automatically.

### "Code Trick" Format
> Aetheris pro tip:
> 
> You can name your jobs for easier debugging:
> 
> ```go
> job, _ := runtime.Submit(ctx, agent, input,
>     runtime.WithName("payment-processing-order-1234"))
> ```
> 
> `aetheris jobs list` will show the names!

---

## Fill-in-the-Blank Versions

### Quick Tip (Single Tweet)
```
💡 Quick tip:

[Actionable tip]

```[optional code]
[short example]
```

#Tip #Aetheris #[relevant stack]
```

### How-To Thread
```
🧵 How to [achieve outcome]:

[Intro context - who is this for?]

**What you need:**
- [Prereq 1]
- [Prereq 2]

**Step 1:**
[Instruction + code if applicable]

**Step 2:**
[Instruction + code if applicable]

**Common pitfall:**
[What to avoid]

**Try it:**
[Link to docs/repo]

Questions? 👇
```

### "3 Things" Thread
```
3 things about [topic]:

1️⃣ [Thing 1]
2️⃣ [Thing 2]  
3️⃣ [Thing 3]

Which one surprised you? 👇
```

---

## Best Practices

1. **Be specific** — "Use descriptive checkpoint names" > "Name things well"
2. **Show code** — Code snippets make tips actionable
3. **Lead with benefit** — What's the payoff?
4. **Keep it scannable** — Use emojis, numbers, bullets
5. **Invite questions** — End with "Questions?" or "Share yours"
6. **Threading tips** — If tip is complex, make it a 3-4 tweet thread

## Content Calendar

| Day | Type | Example |
|-----|------|---------|
| Monday | Quick tip | Single actionable tip |
| Wednesday | How-to | Step-by-step tutorial thread |
| Friday | "3 things" | Educational series |
