# ADR 021: OpenShift-Native Dashboard Strategy

## Status
Proposed

## Context

The Jupyter Notebook Validator Operator exposes Prometheus metrics for monitoring operator health, validation performance, and resource utilization. Currently, we have:

1. **Existing Metrics** (ADR-010):
   - Reconciliation duration and errors
   - Validation job success/failure rates
   - Git clone performance
   - Active pod counts
   - Pod creation metrics

2. **Upcoming Metrics** (ADR-020):
   - Model validation duration
   - Model health checks
   - Prediction validation results
   - Platform detection performance

### Current Challenges

1. **Dashboard Fragmentation**: Users must install separate Grafana instances to visualize metrics
2. **Operational Overhead**: Managing Grafana Operator adds complexity
3. **OpenShift Integration**: Not leveraging OpenShift's built-in monitoring capabilities
4. **User Experience**: Inconsistent dashboard experience across deployments

### OpenShift Monitoring Capabilities

OpenShift 4.8+ provides:
- **Built-in Prometheus**: User workload monitoring with Thanos Querier
- **Console Dashboards**: Native dashboard support via ConfigMaps
- **No Additional Installation**: Dashboards work out-of-the-box
- **Consistent UX**: Integrated into OpenShift Console's Observe section

## Decision

We will adopt **OpenShift-native dashboards using ConfigMaps** as our primary dashboard strategy, with Grafana as an optional alternative for advanced use cases.

### Dashboard Strategy

**Tier 1: OpenShift Console Dashboards (Primary)**
- Use `ConfigMap` resources with `console.openshift.io/dashboard: "true"` label
- Deploy to `openshift-config-managed` namespace
- Leverage built-in Thanos Querier for metrics
- No additional operator installation required

**Tier 2: Grafana Dashboards (Optional)**
- Provide Grafana dashboard JSON for users who prefer Grafana
- Support Grafana Operator integration
- Document installation and configuration

### Core Dashboard Set

We will provide **5 built-in OpenShift Console dashboards**:

1. **Operator Health Overview**
   - Reconciliation duration (p50, p95, p99)
   - Validation success rate
   - Active validation pods
   - Error rate by type
   - Git clone performance

2. **Notebook Validation Performance**
   - Validation duration by namespace
   - Cell execution time distribution
   - Notebook size distribution
   - Success/failure trends
   - Top 10 slowest notebooks

3. **Model-Aware Validation** (Phase 4.3)
   - Model health check status by platform
   - Prediction validation results
   - Platform detection success rate
   - Model validation duration by platform
   - Top failing models

4. **Resource Utilization**
   - Pod CPU/memory usage
   - Active pods by phase
   - Pod creation/deletion rate
   - Queue depth
   - Workload distribution by namespace

5. **Git Operations & Credentials**
   - Git clone duration by auth type
   - Git operation success rate
   - Credential resolution time
   - Repository access patterns
   - Top repositories by validation count

### ConfigMap Structure

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jupyter-notebook-validator-<dashboard-name>
  namespace: openshift-config-managed
  labels:
    console.openshift.io/dashboard: "true"
    console.openshift.io/odc-dashboard: "true"
data:
  <dashboard-name>.json: |
    {
      "dashboard": {
        "title": "Jupyter Notebook Validator - <Title>",
        "tags": ["jupyter", "mlops", "operator"],
        "panels": [...]
      }
    }
```

## Consequences

### Positive

1. **Zero Installation Overhead**: Works out-of-the-box on OpenShift
2. **Consistent UX**: Integrated into OpenShift Console
3. **Lower Maintenance**: No separate Grafana operator to manage
4. **Better Integration**: Leverages OpenShift RBAC and authentication
5. **Faster Adoption**: Users can view dashboards immediately after operator installation

### Negative

1. **OpenShift-Specific**: Dashboards only work on OpenShift (not vanilla Kubernetes)
2. **Limited Customization**: Less flexible than Grafana for advanced visualizations
3. **Feature Parity**: Some Grafana features not available in Console dashboards

### Mitigation

1. **Provide Grafana Alternative**: Include Grafana dashboard JSON for non-OpenShift users
2. **Document Limitations**: Clearly document what's supported in each tier
3. **Community Contributions**: Enable community to create advanced Grafana dashboards

## Alternatives Considered

### Alternative 1: Grafana-Only Strategy
**Rejected**: Requires additional operator installation, increases operational overhead, not OpenShift-native.

### Alternative 2: No Dashboards (Metrics Only)
**Rejected**: Poor user experience, users must create their own dashboards, inconsistent adoption.

### Alternative 3: Embedded Grafana in Operator
**Rejected**: Significantly increases operator complexity, resource overhead, security concerns.

## Implementation Plan

### Phase 1: Core Dashboards (Week 1-2)
- [ ] Create ConfigMap templates for 5 core dashboards
- [ ] Add Kustomize overlays for dashboard deployment
- [ ] Test dashboards on OpenShift 4.18+ cluster
- [ ] Document dashboard installation

### Phase 2: Grafana Alternative (Week 3)
- [ ] Export Grafana-compatible JSON for each dashboard
- [ ] Create Grafana Operator integration guide
- [ ] Test Grafana dashboards with Grafana Operator

### Phase 3: Documentation (Week 4)
- [ ] Create dashboard user guide
- [ ] Add screenshots to documentation
- [ ] Document PromQL queries used
- [ ] Create troubleshooting guide

### Phase 4: Model Validation Dashboards (Phase 4.3)
- [ ] Add model-aware validation dashboard
- [ ] Update metrics for model validation
- [ ] Test with KServe and OpenShift AI

## Related ADRs

- **ADR-010**: Observability and Monitoring Strategy (defines metrics)
- **ADR-020**: Model-Aware Validation Strategy (defines model metrics)
- **ADR-022**: Community Observability Contributions (defines contribution process)

## References

- [OpenShift Monitoring Overview](https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html/monitoring/monitoring-overview)
- [OpenShift Console Dashboards](https://docs.openshift.com/container-platform/4.18/web_console/creating-quick-start-tutorials.html)
- [Prometheus Operator](https://prometheus-operator.dev/)
- [Grafana Operator](https://github.com/grafana-operator/grafana-operator)

## Revision History

| Date       | Author | Description |
|------------|--------|-------------|
| 2025-11-08 | Team   | Initial OpenShift-native dashboard strategy |

