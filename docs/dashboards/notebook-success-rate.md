# Dashboard: Notebook Validation Success Rate

**Category:** Developer Experience
**Format:** OpenShift Console
**Contributor:** @tosin2013
**Requires OCP:** 4.20+

## Purpose

Provides a focused view of notebook validation success and failure rates across
namespaces. Useful for platform teams monitoring MLOps pipeline health and for
notebook authors tracking their validation outcomes over time.

## Prerequisites

- OpenShift user-workload monitoring enabled (`enableUserWorkload: true` in `cluster-monitoring-config`)
- Operator deployed with Prometheus ServiceMonitor enabled

## Installation

```bash
kubectl apply -k config/monitoring/community/
```

## Panels

| Panel | Query | Description |
|-------|-------|-------------|
| Overall Success Rate | `sum(rate(...{status="succeeded"}[5m])) / sum(rate(...[5m])) * 100` | Percentage of validations succeeding in a 5-minute window |
| Validations by Status | `sum(rate(...[5m])) by (status)` | Time series of validation rates broken down by status |
| Success Rate by Namespace | `...{status="succeeded"} by (namespace) / ... by (namespace) * 100` | Per-namespace success percentage |
| Validation Duration (p50, p95) | `histogram_quantile(0.50\|0.95, ...)` | Validation execution time percentiles |
| Total Validations (24h) | `sum(increase(...[24h]))` | Count of all validations in the last 24 hours |
| Failed Validations (24h) | `sum(increase(...{status="failed"}[24h]))` | Count of failed validations with color thresholds |

## Metrics Used

All metrics are exposed by the operator out of the box:

- `notebookvalidationjob_validations_total` (counter, labels: `status`, `namespace`)
- `notebookvalidationjob_validation_duration_seconds` (histogram)
