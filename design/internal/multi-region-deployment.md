# Multi-region Deployment Design

## Overview

This document describes the design for multi-region deployment in Aetheris, enabling high availability and global distribution of agent workloads.

## Goals

1. **High Availability** - Survive region failures without downtime
2. **Low Latency** - Route requests to the nearest region
3. **Data Locality** - Keep data in compliant regions (GDPR)
4. **Horizontal Scaling** - Scale workloads across regions

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   us-east-1     │     │   us-west-2    │     │   eu-west-1    │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │   API     │  │     │  │   API     │  │     │  │   API     │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │  Worker   │  │     │  │  Worker   │  │     │  │  Worker   │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
│  ┌───────────┐  │◄───►│  ┌───────────┐  │◄───►│  ┌───────────┐  │
│  │ Postgres  │  │     │  │ Postgres  │  │     │  │ Postgres  │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Components

### 1. Region Configuration

```yaml
region:
  current_region: "us-east-1"
  regions:
    - id: "us-east-1"
      name: "US East"
      endpoint: "https://api-us-east-1.aetheris.io"
      priority: 1
      is_primary: true
    - id: "us-west-2"
      name: "US West"
      endpoint: "https://api-us-west-2.aetheris.io"
      priority: 2
    - id: "eu-west-1"
      name: "EU West"
      endpoint: "https://api-eu-west-1.aetheris.io"
      priority: 3
  enable_cross_region_replication: true
  replication_mode: "async"
```

### 2. Region-aware Scheduler

- **Local-first**: Prefer executing jobs in the same region
- **Hash-based routing**: Consistent hashing to deterministic region
- **Failover**: Automatically failover to secondary regions

### 3. Cross-region Replication

#### Replication Modes

| Mode  | Consistency | Latency | Use Case      |
| ----- | ----------- | ------- | ------------- |
| sync  | Strong      | High    | Critical jobs |
| async | Eventual    | Low     | Batch jobs    |

#### Replication Protocol

1. **Write Path**: Primary region writes first, then replicates
2. **Conflict Resolution**: Last-write-wins with vector clocks
3. **Read Path**: Read from local, fallback to remote

### 4. Data Partitioning

- **Job Data**: Sharded by jobID across regions
- **Metadata**: Replicated to all regions
- **Checkpoints**: Local-first with periodic sync

## Job Routing

```
┌──────────────┐
│   Request    │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│  Route by JobID  │
│  (Hash-based)    │
└────────┬─────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐ ┌───────┐
│Local  │ │Remote │
│Region │ │Region │
└───────┘ └───────┘
```

## Failure Scenarios

### 1. Region Failure

- Detect failure via health checks
- Route new requests to healthy regions
- Resume in-progress jobs in secondary regions

### 2. Network Partition

- Enable local execution with eventual consistency
- Queue cross-region operations for replay
- Resolve conflicts on reconnection

### 3. Worker Failure

- Lease fencing prevents duplicate execution
- Jobs automatically picked up by other workers
- No data corruption from concurrent access

## Implementation Status

- [x] Region configuration (`pkg/experimental/region/region.go`)
- [x] Region-aware scheduler (`pkg/experimental/region/scheduler.go`)
- [ ] Cross-region replication protocol
- [ ] Global load balancer
- [ ] Data residency enforcement

## Configuration

### API Service

```yaml
region:
  current_region: ${REGION:-us-east-1}
  regions:
    - id: ${REGION:-us-east-1}
      name: ${REGION_NAME:-US East}
      endpoint: ${API_ENDPOINT}
      priority: 1
      is_primary: true
  enable_cross_region_replication: ${ENABLE_REPLICATION:-false}
  replication_mode: ${REPLICATION_MODE:-async}
```

### Worker Service

```yaml
region:
  current_region: ${REGION:-us-east-1}
  regions:
    - id: ${REGION:-us-east-1}
      name: ${REGION_NAME:-US East}
      endpoint: ${API_ENDPOINT}
      priority: 1
  enable_cross_region_replication: ${ENABLE_REPLICATION:-false}
```

## Monitoring

Key metrics for multi-region:

- `aetheris_region_jobs_total` - Jobs processed per region
- `aetheris_region_latency_seconds` - Cross-region latency
- `aetheris_replication_lag_seconds` - Replication lag
- `aetheris_region_failover_total` - Failover events
