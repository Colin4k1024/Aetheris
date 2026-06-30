# Aetheris Crash Recovery Demo — VHS Tape
# Usage:
#   1. Start Aetheris stack: make docker-run (or docker compose -f deployments/compose/docker-compose.yml up -d)
#   2. Install VHS: https://github.com/charmbracelet/vhs
#   3. Run: vhs docs/assets/crash-recovery.tape
#   4. Output: docs/assets/crash-recovery.gif

Output docs/assets/crash-recovery.gif

Set FontSize 18
Set Width 1200
Set Height 600
Set Padding 20
Set Theme "Dracula"

# Title
Type "# Aetheris: Agent Crash Recovery Demo" Enter
Sleep 1s
Type "# Worker crashes. Job resumes. No data lost." Enter
Sleep 2s

# Step 1: Submit a job
Type "echo '▸ Step 1: Submit a long-running job to Aetheris...'" Enter
Sleep 500ms
Type "curl -s -X POST http://localhost:8080/api/agents/demo/message \\" Enter
Sleep 200ms
Type "  -H 'Content-Type: application/json' \\" Enter
Sleep 200ms
Type "  -d '{\"message\": \"Process 500 records\"}' | jq .job_id" Enter
Sleep 2s

Type "echo '  ✓ Job submitted: job_abc123'" Enter
Sleep 1s

# Step 2: Worker is processing
Type "echo ''" Enter
Type "echo '▸ Step 2: Worker is processing steps...'" Enter
Sleep 500ms
Type "echo '  [worker-1] step 1/10 — parsing input          ✓'" Enter
Sleep 400ms
Type "echo '  [worker-1] step 2/10 — validating schema       ✓'" Enter
Sleep 400ms
Type "echo '  [worker-1] step 3/10 — enriching data          ✓'" Enter
Sleep 400ms
Type "echo '  [worker-1] step 4/10 — calling LLM             ✓'" Enter
Sleep 400ms
Type "echo '  [worker-1] step 5/10 — processing batch        ✓'" Enter
Sleep 400ms
Type "echo '  [worker-1] step 6/10 — writing results         ✓'" Enter
Sleep 1s

# Step 3: Crash!
Type "echo ''" Enter
Type "echo '▸ Step 3: 💥 Worker crashes! (kill -9)'" Enter
Sleep 500ms
Type "echo '  [worker-1] step 7/10 — calling external API...'" Enter
Sleep 800ms
Type "echo '  ✗ Connection lost. Worker terminated.'" Enter
Sleep 1s
Type "echo '  ⚠ Without Aetheris: all progress lost. Start from step 1.'" Enter
Sleep 2s

# Step 4: Restart
Type "echo ''" Enter
Type "echo '▸ Step 4: Restart worker (automatic recovery)'" Enter
Sleep 500ms
Type "echo '  [worker-2] Starting...'" Enter
Sleep 500ms
Type "echo '  [worker-2] Claiming pending jobs...'" Enter
Sleep 500ms
Type "echo '  [worker-2] Found job_abc123 with 6/10 completed steps'" Enter
Sleep 500ms
Type "echo '  [worker-2] ✓ Resuming from step 7 (skipping 1-6)'" Enter
Sleep 1s

# Step 5: Continue from where it left off
Type "echo ''" Enter
Type "echo '▸ Step 5: Job continues from checkpoint'" Enter
Sleep 500ms
Type "echo '  [worker-2] step 7/10 — calling external API     ✓'" Enter
Sleep 400ms
Type "echo '  [worker-2] step 8/10 — aggregating results     ✓'" Enter
Sleep 400ms
Type "echo '  [worker-2] step 9/10 — generating report       ✓'" Enter
Sleep 400ms
Type "echo '  [worker-2] step 10/10 — sending notification   ✓'" Enter
Sleep 1s

# Done
Type "echo ''" Enter
Type "echo '  ✓ Job completed successfully!'" Enter
Type "echo ''" Enter
Type "echo '# With Aetheris: resumed at step 7. Zero data lost.'" Enter
Sleep 2s
