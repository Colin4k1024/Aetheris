# SLA Management

> **Version**: v2.3.0+

Aetheris provides SLA (Service Level Agreement) management for job execution.

## Overview

The SLA system provides:

- **Job Deadline Enforcement**: Ensure jobs complete within time limits
- **Step-Level Tracking**: Monitor individual step performance
- **SLO Monitoring**: Track service level objectives
- **Breach Handling**: Automated actions on SLA violation

## Configuration

### Job Deadline

```yaml
sla:
  deadline:
    enabled: true
    default_ttl: 1h
    enforcement_mode: warn  # none, warn, cancel, failover
```

### Step-Level SLA

```yaml
sla:
  step:
    enabled: true
    timeout: 5m
    enforcement_mode: cancel
```

## Usage

### Job Deadline Manager

```go
import "rag-platform/pkg/sla"

manager := sla.NewJobDeadlineManager()

// Set job deadline
err := manager.SetDeadline(ctx, sla.JobDeadline{
    JobID:       "job-123",
    TenantID:    "tenant-1",
    Deadline:    time.Now().Add(1 * time.Hour),
    Enforcement: sla.EnforcementModeCancel,
})

// Check for breaches
breaches := manager.CheckDeadlines(ctx)
for _, breach := range breaches {
    log.Printf("Job %s breached by %v", breach.JobID, breach.Overdue)
}
```

### Step SLA Tracker

```go
tracker := sla.NewStepSLATracker()

// Set step deadline
tracker.SetStepDeadline(ctx, sla.StepDeadline{
    JobID:    "job-123",
    StepID:   "step-1",
    TenantID: "tenant-1",
    Deadline: time.Now().Add(5 * time.Minute),
})

// Record step timing
tracker.RecordStepStart("job-123", "step-1")
// ... do work ...
tracker.RecordStepEnd("job-123", "step-1", "success")
```

## SLA Reporter

```go
monitor := sla.NewMonitor()
reporter := sla.NewReporter(monitor)

// Generate report
report, err := reporter.GenerateReport(ctx, "tenant-1", 24*time.Hour)

// Output as JSON
json, _ := reporter.FormatJSON(report)

// Output as summary
summary := reporter.FormatSummary(report)
fmt.Print(summary)
```

### Sample Output

```
SLA Report for Tenant: tenant-1
Period: 2026-03-04T10:00:00Z - 2026-03-05T10:00:00Z
Total Jobs: 100
Completed: 95
Failed: 3
Breached: 2

SLO Status:
  ✓ Availability: 95.00% (target: 99.90%)
  ✗ Latency: 92.00% (target: 95.00%)
```

## Enforcement Modes

| Mode | Action |
|------|--------|
| `none` | No action, just log |
| `warn` | Emit warning event |
| `cancel` | Cancel the job/step |
| `failover` | Trigger failover to another region |

## Monitoring

### Metrics

```bash
# Job deadline breaches
aetheris_sla_job_breach_total{tenant="tenant-1"} 5

# Step timeout
aetheris_sla_step_timeout_total{tenant="tenant-1"} 12

# SLO compliance rate
aetheris_slo_compliance_rate{tenant="tenant-1",slo="availability"} 99.5
```
