# Dashboard Contribution Guide

**ADR-022: Community Observability Contributions**

Thank you for contributing a dashboard to the Jupyter Notebook Validator Operator!

## Quick Start

1. **Open a proposal issue** using the `dashboard-proposal` label.
2. Implement your dashboard (see formats below).
3. Add documentation (see template below).
4. Submit a pull request.

---

## Dashboard Formats

### OpenShift Console Dashboard (preferred for OCP users)

Create a `ConfigMap` in `config/monitoring/community/` with the following labels:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jupyter-notebook-validator-<your-dashboard-name>
  namespace: openshift-config-managed
  labels:
    console.openshift.io/dashboard: "true"
    console.openshift.io/odc-dashboard: "true"
    app.kubernetes.io/name: jupyter-notebook-validator-operator
    app.kubernetes.io/component: observability
    app.kubernetes.io/part-of: community-dashboards
data:
  <your-dashboard>.json: |
    {
      "dashboard": {
        "title": "Jupyter Notebook Validator - <Your Title>",
        "tags": ["jupyter", "mlops", "operator", "<your-tag>"],
        "panels": []
      }
    }
```

### Grafana Dashboard

Provide a Grafana-compatible JSON file in `config/monitoring/community/` and a `ConfigMap` wrapper. Label with `grafana_dashboard: "true"` for automatic discovery by Grafana Operator.

---

## Required Metrics

All dashboards must use metrics already exposed by the operator. Available metrics include:

| Metric | Description |
|--------|-------------|
| `notebookvalidationjob_reconciliation_duration_seconds` | Reconciliation duration histogram |
| `notebookvalidationjob_validation_total` | Total validation jobs (labels: `status`) |
| `notebookvalidationjob_validation_duration_seconds` | Notebook validation duration histogram |
| `notebookvalidationjob_git_clone_duration_seconds` | Git clone duration histogram |
| `notebookvalidationjob_active_pods` | Currently active validation pods |
| `notebookvalidationjob_pod_creation_total` | Pod creation counter |

If your dashboard requires new metrics, open a separate issue with the `metrics-proposal` label.

---

## Documentation Template

Create `docs/dashboards/<your-dashboard-name>.md`:

```markdown
# Dashboard: <Your Title>

**Category:** <one of: Model-Aware / Multi-Cluster / Cost / Security / Developer Experience / Other>
**Format:** OpenShift Console / Grafana
**Contributor:** @<github-handle>
**Requires OCP:** 4.x+

## Purpose

<1-2 sentences describing who should use this dashboard and why.>

## Prerequisites

- OpenShift user-workload monitoring enabled (`enableUserWorkload: true` in cluster-monitoring-config)
- Operator deployed with Prometheus ServiceMonitor enabled

## Installation

```bash
kubectl apply -k config/monitoring/community/
```

## Panels

| Panel | Query | Description |
|-------|-------|-------------|
| ... | ... | ... |

## Screenshots

_(optional but encouraged)_
```

---

## Review Criteria

Core team reviewers check for:

- [ ] Uses only existing operator metrics (or new metrics are proposed separately)
- [ ] ConfigMap is in the correct namespace (`openshift-config-managed` for console dashboards)
- [ ] Required labels are present
- [ ] Documentation file exists in `docs/dashboards/`
- [ ] DCO sign-off (`git commit -s`)

---

## Contribution Categories

| Category | Label | Description |
|----------|-------|-------------|
| Model-Aware Validation | `dashboard/model-aware` | KServe, OpenShift AI metrics |
| Multi-Cluster | `dashboard/multi-cluster` | Cross-cluster Thanos queries |
| Cost Optimization | `dashboard/cost` | Resource efficiency metrics |
| Security & Compliance | `dashboard/security` | RBAC and audit metrics |
| Developer Experience | `dashboard/dx` | Per-user and per-namespace metrics |
| Platform Integration | `dashboard/platform` | Datadog, New Relic, Splunk exports |

---

## Maintenance

Dashboard contributors are expected to respond to issues within 30 days. If a dashboard becomes unmaintained, the core team will open an adoption issue (`good first issue` label) so the community can adopt it.
