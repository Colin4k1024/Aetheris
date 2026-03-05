# Multi-Region Deployment

> **Version**: v2.3.0+

This document describes Aetheris multi-region deployment capabilities.

## Overview

Aetheris supports multi-region deployment for high availability and disaster recovery. The system provides:

- **GeoDNS Routing**: Latency-based region selection
- **Region Failover**: Automatic failover on region failure
- **Cross-Region Replication**: Event replication between regions

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   us-east-1     │     │   us-west-1    │
│  ┌───────────┐  │     │  ┌───────────┐  │
│  │   API     │  │     │  │   API     │  │
│  └───────────┘  │     │  └───────────┘  │
│  ┌───────────┐  │     │  ┌───────────┐  │
│  │  Worker   │  │     │  │  Worker   │  │
│  └───────────┘  │     │  └───────────┘  │
│  ┌───────────┐  │     │  ┌───────────┐  │
│  │  Postgres │  │◄───►│  │  Postgres │  │
│  └───────────┘  │Replication│ └───────────┘  │
└─────────────────┘     └─────────────────┘
         │
         ▼
    ┌──────────┐
    │  Redis   │
    │ (GeoDNS) │
    └──────────┘
```

## Configuration

### Region Configuration

```yaml
region:
  current_region: us-east-1
  enable_cross_region_replication: true
  replication_mode: async
  regions:
    - id: us-east-1
      name: US East
      endpoint: https://api-us-east-1.aetheris.dev
      is_primary: true
      continent: na
    - id: us-west-1
      name: US West
      endpoint: https://api-us-west-1.aetheris.dev
      continent: na
    - id: eu-west-1
      name: EU West
      endpoint: https://api-eu-west-1.aetheris.dev
      continent: eu
```

### GeoDNS Configuration

```yaml
geodns:
  enabled: true
  update_interval: 30s
  fallback_timeout: 5s
```

## Features

### Latency-Based Routing

The `GeoDNSResolver` selects the best region based on measured latency:

```go
resolver := region.NewGeoDNSResolver(regions)
bestRegion := resolver.Resolve(ctx)
```

### Region Failover

The `RegionFailoverManager` handles automatic failover:

```go
failoverMgr := region.NewRegionFailoverManager(regions)
fallback := failoverMgr.GetFallbackRegion(ctx, failedRegion)
```

### Cross-Region Replication

The `Replicator` copies events between regions:

```go
replicator, err := replication.NewReplicator(cfg, localStore)
replicator.AddPeer(Peer{RegionID: "us-west-1", Endpoint: "..."})
replicator.Start(ctx)
```

#### Replication Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `sync` | Wait for all regions | Critical data |
| `async` | Fire and forget | Best performance |

#### Conflict Resolution

| Strategy | Description |
|----------|-------------|
| `local_wins` | Keep local version |
| `remote_wins` | Keep remote version |
| `merged` | Merge changes |

## Monitoring

### Metrics

```bash
# Region latency
aetheris_region_latency{region="us-east-1"} 45.2

# Replication lag
aetheris_replication_lag{region="us-west-1"} 120ms

# Failover events
aetheris_failover_total 3
```

### Health Checks

```bash
# Check region health
curl http://localhost:8080/api/health/region
```

## Best Practices

1. **Minimum 2 regions** for HA
2. **Async replication** for better performance
3. **Same continent** preferred for lower latency
4. **Regular failover drills** to test recovery
