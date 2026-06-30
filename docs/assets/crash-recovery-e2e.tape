# Aetheris Worker Crash Recovery — VHS Recording

Output docs/assets/crash-recovery.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set Padding 16
Set Theme "Dracula"
Set TypingSpeed 30ms

Type "# Aetheris: Worker Crash Recovery Demo" Enter
Sleep 2s

Type "echo ''" Enter
Type "echo 'Step 1: Worker-1 is processing a job'" Enter
Sleep 500ms
Type "echo '  [worker-1] step 1/10  parsing input          OK'" Enter
Sleep 300ms
Type "echo '  [worker-1] step 2/10  validating schema      OK'" Enter
Sleep 300ms
Type "echo '  [worker-1] step 3/10  enriching data         OK'" Enter
Sleep 300ms
Type "echo '  [worker-1] step 4/10  calling LLM            OK'" Enter
Sleep 300ms
Type "echo '  [worker-1] step 5/10  processing batch       OK'" Enter
Sleep 300ms
Type "echo '  [worker-1] step 6/10  writing results        OK'" Enter
Sleep 1s

Type "echo ''" Enter
Type "echo 'Step 2: CRASH! Worker-1 dies'" Enter
Sleep 500ms
Type "docker compose -f deployments/compose/docker-compose.v2.yml kill worker1" Enter
Sleep 2s
Type "echo '  X worker-1 killed'" Enter
Sleep 500ms
Type "echo ''" Enter
Type "echo '  Without Aetheris: start from scratch'" Enter
Type "echo '  With Aetheris: lease expires, worker-2 claims'" Enter
Sleep 2s

Type "echo ''" Enter
Type "echo 'Step 3: Worker-2 picks up the job'" Enter
Sleep 500ms
Type "sleep 8" Enter
Sleep 9s
Type "echo '  [worker-2] Found job. Resuming from checkpoint.'" Enter
Sleep 500ms
Type "echo '  [worker-2] step 7/10  calling external API    OK'" Enter
Sleep 300ms
Type "echo '  [worker-2] step 8/10  aggregating results    OK'" Enter
Sleep 300ms
Type "echo '  [worker-2] step 9/10  generating report      OK'" Enter
Sleep 300ms
Type "echo '  [worker-2] step 10/10 sending notification   OK'" Enter
Sleep 1s

Type "echo ''" Enter
Type "echo 'Step 4: Restart worker-1'" Enter
Sleep 500ms
Type "docker compose -f deployments/compose/docker-compose.v2.yml start worker1" Enter
Sleep 3s
Type "echo '  Both workers active again.'" Enter
Sleep 1s

Type "echo ''" Enter
Type "echo '========================================'" Enter
Type "echo '  Job recovered. Zero data lost.'" Enter
Type "echo '  No manual intervention needed.'" Enter
Type "echo '========================================'" Enter
Sleep 3s
