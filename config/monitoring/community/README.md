# Community Observability Contributions

This directory holds community-contributed dashboards and monitoring integrations for the Jupyter Notebook Validator Operator.

## Contributing

See [`docs/dashboards/CONTRIBUTING.md`](../../../docs/dashboards/CONTRIBUTING.md) for the full contribution guide.

## Dashboard Categories

| Category | Status | Description |
|----------|--------|-------------|
| Model-Aware Validation | 🔴 Needs Contributor | Model health and prediction metrics |
| Multi-Cluster | 🔴 Needs Contributor | Cross-cluster validation job monitoring |
| Cost Optimization | 🔴 Needs Contributor | Resource usage and cost per notebook |
| Security & Compliance | 🔴 Needs Contributor | RBAC audit and credential tracking |
| Developer Experience | 🔴 Needs Contributor | Per-user success rates and error patterns |

## How To Add Your Dashboard

1. Fork this repository
2. Add your dashboard ConfigMap (OpenShift Console format) or Grafana JSON to this directory
3. Add it to `kustomization.yaml` resources list
4. Add documentation in `docs/dashboards/<your-dashboard>.md`
5. Open a PR with the `dashboard-contribution` label
