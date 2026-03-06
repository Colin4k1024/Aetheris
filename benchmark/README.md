# Aetheris Benchmark Suite

## Prerequisites

- k6 or vegeta installed
- Aetheris running (API + Worker + PostgreSQL)

## Scenarios

### 1. Simple Job Throughput

Single LLM call per job.

```bash
k6 run --vus=10 --duration=30s simple_job.lua
```

Expected: 10-20 jobs/minute per worker

### 2. Complex Job Latency (TBD)

Multi-tool calls + reasoning.

```bash
# TODO: Implement complex_job.lua
# k6 run --vus=5 --duration=60s complex_job.lua
```

Expected: 2-5 jobs/minute per worker

### 3. Long-Running Job (HITL) (TBD)

Simulate park/resume cycles.

```bash
# TODO: Implement hitl_job.lua
# k6 run --vus=2 --duration=120s hitl_job.lua
```

Expected: 1-2 concurrent per worker

### 4. Multi-Worker Scaling (TBD)

Test horizontal scaling with 1/2/4/8 workers.

```bash
# TODO: Implement run_scaling_test.sh
# ./run_scaling_test.sh
```

## Results

> Note: Results directory and analysis documentation pending implementation of benchmark scenarios.

```bash
# mkdir -p benchmark/results
# Results will be saved to benchmark/results/ directory
```
