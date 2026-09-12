# Observability Guide

This guide covers monitoring setup for the Jupyter Notebook Validator Operator,
including OpenShift Console dashboards, Grafana integration, and community
contributions.

## Prerequisites

### OpenShift User-Workload Monitoring

The operator's Prometheus metrics require **user-workload monitoring** to be
enabled on the cluster. This is disabled by default on OpenShift.

**Enable user-workload monitoring:**

1. Edit (or create) the `cluster-monitoring-config` ConfigMap in the
   `openshift-monitoring` namespace:

   ```bash
   oc -n openshift-monitoring edit configmap cluster-monitoring-config
   ```

2. Add or ensure the following key exists:

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: cluster-monitoring-config
     namespace: openshift-monitoring
   data:
     config.yaml: |
       enableUserWorkload: true
   ```

3. Wait for the `prometheus-user-workload` pods to start in the
   `openshift-user-workload-monitoring` namespace:

   ```bash
   oc -n openshift-user-workload-monitoring get pods -w
   ```

### ServiceMonitor

The operator deploys a `ServiceMonitor` resource that tells Prometheus where to
scrape metrics. Verify it exists:

```bash
oc get servicemonitor -n jupyter-notebook-validator-operator-system
```

If using the Helm chart, ensure `prometheus.enabled: true` in your values.

---

## OpenShift Console Dashboards (ADR-021)

The operator ships **5 built-in OpenShift Console dashboards** as ConfigMap
resources. They appear automatically in **Observe > Dashboards** once applied.

### Installation

```bash
# Apply all monitoring resources (dashboards + ServiceMonitor)
kubectl apply -k config/monitoring/

# Or apply dashboards only
kubectl apply -k config/monitoring/openshift-console/
```

### Dashboard Inventory

| Dashboard | ConfigMap Name | Description |
|-----------|---------------|-------------|
| Operator Health | `jupyter-notebook-validator-operator-health` | Reconciliation duration, success rate, error rate, active pods |
| Notebook Performance | `jupyter-notebook-validator-notebook-performance` | Validation duration by namespace, cell execution times, trends |
| Model Validation | `jupyter-notebook-validator-model-validation` | Model health checks, prediction validation, platform detection |
| Resource Utilization | `jupyter-notebook-validator-resource-utilization` | Pod CPU/memory, active pods by phase, queue depth |
| Git Operations | `jupyter-notebook-validator-git-operations` | Clone duration by auth type, success rate, repository patterns |

### Namespace

All console dashboards are deployed to `openshift-config-managed` with the
label `console.openshift.io/dashboard: "true"`. The OpenShift Console
automatically discovers and displays them.

### Verification

After applying, confirm the ConfigMaps exist:

```bash
kubectl get configmap -n openshift-config-managed -l console.openshift.io/dashboard=true
```

Then open the OpenShift Console and navigate to **Observe > Dashboards**. The
dashboards appear under names prefixed with "Jupyter Notebook Validator".

---

## Grafana Integration (Optional)

For advanced visualization or non-OpenShift clusters, a Grafana dashboard JSON
is provided at `config/monitoring/grafana/`.

### With Grafana Operator

If the Grafana Operator is installed, apply the kustomization overlay:

```bash
kubectl apply -k config/monitoring/grafana/
```

This creates a ConfigMap with the `grafana_dashboard: "true"` label for
automatic discovery.

### Manual Import

Import `config/monitoring/grafana/jupyter-notebook-validator-dashboard.json`
directly into any Grafana instance via **Dashboards > Import**.

---

## Community Dashboards (ADR-022)

Community-contributed dashboards live in `config/monitoring/community/`.

### Installation

```bash
kubectl apply -k config/monitoring/community/
```

### Contributing

See [docs/dashboards/CONTRIBUTING.md](dashboards/CONTRIBUTING.md) for the full
contribution guide.

---

## Metrics Reference

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `notebookvalidationjob_reconciliation_duration_seconds` | Histogram | `namespace` | Reconciliation loop duration |
| `notebookvalidationjob_reconciliation_errors_total` | Counter | `error_type` | Reconciliation error count |
| `notebookvalidationjob_validations_total` | Counter | `status`, `namespace` | Total validation jobs |
| `notebookvalidationjob_validation_duration_seconds` | Histogram | `namespace` | Notebook validation duration |
| `notebookvalidationjob_git_clone_duration_seconds` | Histogram | `auth_type` | Git clone duration |
| `notebookvalidationjob_active_pods` | Gauge | `phase` | Currently active validation pods |
| `notebookvalidationjob_pod_creations_total` | Counter | `result` | Pod creation attempts |

---

## Troubleshooting

### Dashboards not appearing in Console

1. Verify ConfigMaps are in `openshift-config-managed`:
   ```bash
   kubectl get cm -n openshift-config-managed | grep jupyter
   ```
2. Confirm the `console.openshift.io/dashboard: "true"` label is present.
3. Refresh the OpenShift Console browser tab.

### No metrics data in panels

1. Verify user-workload monitoring is enabled (see Prerequisites above).
2. Check that the ServiceMonitor exists and Prometheus is scraping:
   ```bash
   kubectl get servicemonitor -n jupyter-notebook-validator-operator-system
   ```
3. Confirm the operator is running and has processed at least one
   NotebookValidationJob (metrics are only populated after first use).

### Grafana Operator not discovering dashboards

Ensure the Grafana Operator's `labelSelector` matches `grafana_dashboard: "true"`.
Check the GrafanaDashboard CR or ConfigMap discovery configuration.
