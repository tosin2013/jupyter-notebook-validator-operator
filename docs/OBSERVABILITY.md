# Observability and Monitoring Guide

## Overview

The Jupyter Notebook Validator Operator provides comprehensive observability through Prometheus metrics, Grafana dashboards, and OpenShift Console dashboards. This guide covers installation, configuration, and troubleshooting.

**Based on:**
- ADR-010: Observability and Monitoring Strategy
- ADR-021: OpenShift-Native Dashboard Strategy
- ADR-022: Community Observability Contributions

---

## Quick Start

### Prerequisites

- OpenShift 4.18+ or Kubernetes 1.31+ cluster
- Prometheus Operator installed (for Kubernetes)
- User workload monitoring enabled (for OpenShift)

### Installation

The operator automatically exposes metrics on the `/metrics` endpoint. No additional configuration is required for basic monitoring.

---

## Prometheus Metrics

### ServiceMonitor Configuration

The operator includes a ServiceMonitor resource that automatically configures Prometheus to scrape metrics:

**Location:** `config/prometheus/monitor.yaml`

**Features:**
- 30-second scrape interval
- 10-second scrape timeout
- Automatic namespace, pod, and service labels
- Metric filtering for operator-specific metrics only

### Enabling User Workload Monitoring (OpenShift)

```bash
# Enable user workload monitoring
oc -n openshift-user-workload-monitoring get configmap user-workload-monitoring-config || \
oc -n openshift-user-workload-monitoring create configmap user-workload-monitoring-config \
  --from-literal=config.yaml='enableUserWorkload: true'

# Verify monitoring is enabled
oc -n openshift-user-workload-monitoring get pods
```

### Available Metrics

#### Core Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `notebookvalidationjob_reconciliation_duration_seconds` | Histogram | namespace, result | Reconciliation loop duration |
| `notebookvalidationjob_reconciliation_errors_total` | Counter | namespace, error_type | Reconciliation errors |
| `notebookvalidationjob_validations_total` | Counter | namespace, status | Total validation jobs |
| `notebookvalidationjob_active_pods` | Gauge | namespace, phase | Active validation pods |
| `notebookvalidationjob_git_clone_duration_seconds` | Histogram | namespace, auth_type | Git clone duration |
| `notebookvalidationjob_pod_creations_total` | Counter | namespace, result | Pod creation attempts |

#### Model Validation Metrics (Phase 7 ✅)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `notebookvalidationjob_model_validation_duration_seconds` | Histogram | namespace, platform, result | Model validation duration |
| `notebookvalidationjob_model_health_checks_total` | Counter | namespace, platform, status | Model health checks |
| `notebookvalidationjob_prediction_validations_total` | Counter | namespace, platform, result | Prediction validations |
| `notebookvalidationjob_platform_detection_duration_seconds` | Histogram | namespace, platform, detected | Platform detection duration |

### Querying Metrics

```promql
# Validation success rate
sum(rate(notebookvalidationjob_validations_total{status="succeeded"}[5m])) 
/ 
sum(rate(notebookvalidationjob_validations_total[5m])) * 100

# P95 reconciliation duration
histogram_quantile(0.95, 
  sum(rate(notebookvalidationjob_reconciliation_duration_seconds_bucket[5m])) by (le)
)

# Active pods by phase
sum(notebookvalidationjob_active_pods) by (phase)

# Git clone performance by auth type
histogram_quantile(0.95, 
  sum(rate(notebookvalidationjob_git_clone_duration_seconds_bucket[5m])) by (le, auth_type)
)

# Model validation success rate
sum(rate(notebookvalidationjob_model_validation_duration_seconds_count{result="success"}[5m])) 
/ 
sum(rate(notebookvalidationjob_model_validation_duration_seconds_count[5m])) * 100
```

---

## OpenShift Console Dashboards

### Installation

OpenShift Console dashboards are deployed as ConfigMaps in the `openshift-config-managed` namespace:

```bash
# Deploy all dashboards
oc apply -f config/monitoring/openshift-console/

# Verify dashboards are deployed
oc get configmap -n openshift-config-managed -l console.openshift.io/dashboard=true

# View dashboards in OpenShift Console
# Navigate to: Observe → Dashboards → Select "Jupyter Notebook Validator" dashboards
```

### Available Dashboards

1. **Operator Health Overview** (`operator-health-dashboard.yaml`)
   - Reconciliation duration (p50, p95, p99)
   - Validation success rate
   - Active validation pods
   - Error rate by type
   - Pod creation success rate

2. **Notebook Performance** (`notebook-performance-dashboard.yaml`)
   - Validation duration by namespace
   - Success rate by namespace
   - Validation jobs by status
   - Success vs failure trends

3. **Model Validation** (`model-validation-dashboard.yaml`)
   - Model validation duration by platform
   - Model health checks by platform
   - Prediction validation results
   - Platform detection performance

4. **Resource Utilization** (`resource-utilization-dashboard.yaml`)
   - Active pods by phase
   - Pod creation rate
   - Workload distribution by namespace
   - Resource usage trends

5. **Git Operations** (`git-operations-dashboard.yaml`)
   - Git clone duration by auth type
   - Git clone success rate
   - Anonymous vs authenticated clones
   - Git operation failures

### Accessing Dashboards

1. Log in to OpenShift Console
2. Navigate to **Observe** → **Dashboards**
3. Select **Jupyter Notebook Validator** dashboards from the dropdown
4. Use time range selector to adjust view (Last 1h, 6h, 24h, etc.)

---

## Grafana Dashboards

### Installation

For users who prefer Grafana, we provide a comprehensive Grafana dashboard JSON:

**Location:** `config/monitoring/grafana/jupyter-notebook-validator-dashboard.json`

#### Option 1: Manual Import

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Upload `config/monitoring/grafana/jupyter-notebook-validator-dashboard.json`
4. Select Prometheus data source
5. Click **Import**

#### Option 2: Grafana Operator

```yaml
apiVersion: grafana.integreatly.org/v1beta1
kind: GrafanaDashboard
metadata:
  name: jupyter-notebook-validator-dashboard
  namespace: grafana
spec:
  instanceSelector:
    matchLabels:
      dashboards: "grafana"
  json: |
    # Paste content from jupyter-notebook-validator-dashboard.json
```

---

## Prometheus Alerting Rules

### Installation

Deploy Prometheus alerting rules to receive notifications for critical issues:

```bash
# Deploy alerting rules
oc apply -f config/prometheus/alerting-rules.yaml

# Verify rules are loaded
oc get prometheusrule -n jupyter-notebook-validator-operator-system
```

### Available Alerts

| Alert | Severity | Threshold | Description |
|-------|----------|-----------|-------------|
| `HighValidationFailureRate` | Warning | >20% for 10m | High validation failure rate |
| `CriticalValidationFailureRate` | Critical | >50% for 5m | Critical validation failure rate |
| `HighReconciliationErrorRate` | Warning | >1 error/sec for 10m | High reconciliation error rate |
| `SlowReconciliationPerformance` | Warning | P95 >60s for 15m | Slow reconciliation performance |
| `HighActivePodCount` | Warning | >20 pods for 10m | High number of active pods |
| `HighGitCloneFailureRate` | Warning | >10% for 10m | High Git clone failure rate |
| `HighModelValidationFailureRate` | Warning | >20% for 10m | High model validation failure rate |
| `PlatformDetectionFailures` | Warning | >0.5 failures/sec for 10m | Platform detection failures |
| `HighPodCreationFailureRate` | Warning | >10% for 10m | High pod creation failure rate |

### Configuring Alertmanager

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alertmanager-config
  namespace: openshift-monitoring
data:
  alertmanager.yaml: |
    global:
      resolve_timeout: 5m
    route:
      group_by: ['alertname', 'component']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 12h
      receiver: 'default'
      routes:
        - match:
            component: jupyter-notebook-validator
          receiver: 'jupyter-notebook-validator-team'
    receivers:
      - name: 'default'
        # Default receiver
      - name: 'jupyter-notebook-validator-team'
        slack_configs:
          - api_url: 'YOUR_SLACK_WEBHOOK_URL'
            channel: '#jupyter-notebook-validator-alerts'
            title: '{{ .GroupLabels.alertname }}'
            text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

---

## Troubleshooting

### No Metrics Appearing

**Problem:** Prometheus is not scraping metrics from the operator.

**Solution:**

1. Verify ServiceMonitor is deployed:
   ```bash
   oc get servicemonitor -n jupyter-notebook-validator-operator-system
   ```

2. Check if user workload monitoring is enabled (OpenShift):
   ```bash
   oc -n openshift-user-workload-monitoring get pods
   ```

3. Verify the metrics service is running:
   ```bash
   oc get svc -n jupyter-notebook-validator-operator-system
   oc get endpoints -n jupyter-notebook-validator-operator-system
   ```

4. Check Prometheus targets:
   ```bash
   # OpenShift Console: Observe → Targets
   # Look for "jupyter-notebook-validator-operator" targets
   ```

5. Test metrics endpoint directly:
   ```bash
   oc port-forward -n jupyter-notebook-validator-operator-system \
     svc/controller-manager-metrics-service 8443:8443

   curl -k https://localhost:8443/metrics
   ```

### Dashboards Not Appearing in OpenShift Console

**Problem:** OpenShift Console dashboards are not visible.

**Solution:**

1. Verify ConfigMaps are deployed:
   ```bash
   oc get configmap -n openshift-config-managed \
     -l console.openshift.io/dashboard=true
   ```

2. Check ConfigMap labels:
   ```bash
   oc get configmap jupyter-notebook-validator-operator-health \
     -n openshift-config-managed -o yaml
   ```

3. Verify the `console.openshift.io/dashboard: "true"` label is present.

4. Restart the OpenShift Console pods:
   ```bash
   oc delete pods -n openshift-console -l app=console
   ```

### High Cardinality Metrics

**Problem:** Too many unique label combinations causing Prometheus performance issues.

**Solution:**

1. Review metric cardinality:
   ```promql
   count by (__name__) ({__name__=~"notebookvalidationjob_.*"})
   ```

2. Adjust ServiceMonitor metric relabeling to drop high-cardinality labels:
   ```yaml
   metricRelabelings:
     - sourceLabels: [namespace]
       regex: '(default|kube-.*)'
       action: drop
   ```

### Alerts Not Firing

**Problem:** Prometheus alerts are not triggering.

**Solution:**

1. Verify PrometheusRule is loaded:
   ```bash
   oc get prometheusrule jupyter-notebook-validator-alerts \
     -n jupyter-notebook-validator-operator-system
   ```

2. Check Prometheus rule evaluation:
   ```bash
   # OpenShift Console: Observe → Alerting → Alert Rules
   # Search for "jupyter-notebook-validator"
   ```

3. Test alert query manually:
   ```promql
   (
     sum(rate(notebookvalidationjob_validations_total{status="failed"}[5m]))
     /
     sum(rate(notebookvalidationjob_validations_total[5m]))
   ) > 0.20
   ```

4. Verify Alertmanager configuration:
   ```bash
   oc get secret alertmanager-main -n openshift-monitoring -o yaml
   ```

---

## Best Practices

### Metric Retention

- **Short-term:** 15 days (Prometheus)
- **Long-term:** 6 months (Thanos/Victoria Metrics)
- **Aggregation:** Use recording rules for frequently queried metrics

### Dashboard Organization

- **Operator Health:** Daily monitoring by SRE team
- **Notebook Performance:** Weekly review by development team
- **Model Validation:** Review after each model deployment
- **Resource Utilization:** Monthly capacity planning
- **Git Operations:** Review when credential issues occur

### Alert Tuning

- **Warning alerts:** 10-15 minute evaluation period
- **Critical alerts:** 5 minute evaluation period
- **Avoid alert fatigue:** Tune thresholds based on baseline metrics
- **Runbooks:** Document resolution steps for each alert

---

## Additional Resources

- **ADR-010:** [Observability and Monitoring Strategy](adrs/010-observability-and-monitoring-strategy.md)
- **ADR-021:** [OpenShift-Native Dashboard Strategy](adrs/021-openshift-native-dashboard-strategy.md)
- **ADR-022:** [Community Observability Contributions](adrs/022-community-observability-contributions.md)
- **Community Dashboards:** [COMMUNITY_OBSERVABILITY.md](COMMUNITY_OBSERVABILITY.md)
- **Prometheus Documentation:** https://prometheus.io/docs/
- **OpenShift Monitoring:** https://docs.openshift.com/container-platform/4.18/monitoring/monitoring-overview.html



