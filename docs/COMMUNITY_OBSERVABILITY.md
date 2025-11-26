# Community Observability Contributions

## 🎯 Overview

We welcome community contributions for observability dashboards, alerts, and monitoring integrations! This guide helps you contribute dashboards for different platforms and use cases.

---

## 🚀 We Need Your Help!

The Jupyter Notebook Validator Operator provides **5 built-in dashboards** for core monitoring, but we need the community to help us cover specialized use cases!

### Why Contribute?

**Impact:**
- Help thousands of users monitor their notebook validation workflows
- Share your expertise with the community
- Solve real-world monitoring challenges

**Recognition:**
- **Badge**: "Dashboard Contributor" on your GitHub profile
- **Newsletter Feature**: Highlighted in our monthly newsletter
- **Speaking Opportunity**: Present your dashboard at our community call
- **Swag**: Contributor t-shirt and stickers

**Support:**
- **Mentorship**: Core team provides guidance and code reviews
- **Office Hours**: Monthly office hours for questions
- **Community**: Active Slack channel for support

---

## 📊 Dashboard Contribution Areas

### **Built-In Dashboards** (Maintained by Core Team)

| Dashboard | Status | Platform | Description | Location |
|-----------|--------|----------|-------------|----------|
| **Operator Health Overview** | ✅ Complete | OpenShift Console | Core operator metrics | `config/monitoring/openshift-console/operator-health-dashboard.yaml` |
| **Notebook Performance** | ✅ Complete | OpenShift Console | Validation performance metrics | `config/monitoring/openshift-console/notebook-performance-dashboard.yaml` |
| **Model Validation** | ✅ Complete | OpenShift Console | Model validation metrics (ADR-020) | `config/monitoring/openshift-console/model-validation-dashboard.yaml` |
| **Resource Utilization** | ✅ Complete | OpenShift Console | Pod and resource metrics | `config/monitoring/openshift-console/resource-utilization-dashboard.yaml` |
| **Git Operations** | ✅ Complete | OpenShift Console | Git clone performance (ADR-009) | `config/monitoring/openshift-console/git-operations-dashboard.yaml` |
| **Grafana Dashboard** | ✅ Complete | Grafana | Comprehensive operator dashboard | `config/monitoring/grafana/jupyter-notebook-validator-dashboard.json` |

---

### **Community Dashboards** (Help Wanted! 🙋)

#### **1. Multi-Cluster Dashboard** - 🔴 NEEDS CONTRIBUTOR

**Platform:** Red Hat Advanced Cluster Management (RHACM) / Grafana  
**Use Case:** Organizations running operators across multiple OpenShift clusters  

**Metrics:**
- Validation jobs across multiple clusters
- Cross-cluster success rates
- Cluster-specific error patterns
- Resource usage by cluster
- Cluster health correlation

**Target Audience:** Platform teams managing multi-cluster deployments  
**Estimated Effort:** 4-6 hours  
**Volunteer:** 🙋 **[Claim this dashboard!](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?labels=dashboard-proposal&template=dashboard-proposal.md&title=Multi-Cluster+Dashboard)**

---

#### **2. Cost Optimization Dashboard** - 🔴 NEEDS CONTRIBUTOR

**Platform:** OpenShift Console / Grafana  
**Use Case:** Teams focused on resource efficiency and cost reduction  

**Metrics:**
- Pod resource requests vs. actual usage
- Validation cost per notebook (CPU-hours, memory-hours)
- Idle pod time analysis
- Resource efficiency score
- Cost trends over time

**Target Audience:** FinOps teams, platform engineers  
**Estimated Effort:** 3-5 hours  
**Volunteer:** 🙋 **[Claim this dashboard!](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?labels=dashboard-proposal&template=dashboard-proposal.md&title=Cost+Optimization+Dashboard)**

---

#### **3. Security & Compliance Dashboard** - 🔴 NEEDS CONTRIBUTOR

**Platform:** OpenShift Console / Grafana  
**Use Case:** Organizations with strict audit and compliance requirements  

**Metrics:**
- Credential usage patterns (auth type distribution)
- Secret rotation status
- RBAC violations
- Audit log summary
- Security event timeline

**Target Audience:** Security teams, compliance officers  
**Estimated Effort:** 4-6 hours  
**Volunteer:** 🙋 **[Claim this dashboard!](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?labels=dashboard-proposal&template=dashboard-proposal.md&title=Security+Compliance+Dashboard)**

---

#### **4. Developer Experience Dashboard** - 🔴 NEEDS CONTRIBUTOR

**Platform:** OpenShift Console / Grafana  
**Use Case:** Teams optimizing for developer productivity  

**Metrics:**
- Average validation time by user/team
- Most common errors by user
- Notebook complexity trends
- User success rate
- Time-to-first-success for new users

**Target Audience:** Developer experience teams, team leads  
**Estimated Effort:** 3-4 hours  
**Volunteer:** 🙋 **[Claim this dashboard!](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?labels=dashboard-proposal&template=dashboard-proposal.md&title=Developer+Experience+Dashboard)**

---

#### **5. Advanced Model Validation Dashboard** - 🔴 NEEDS CONTRIBUTOR

**Platform:** Grafana (requires advanced visualizations)  
**Use Case:** ML teams with complex model validation workflows  

**Metrics:**
- Model health checks by platform (KServe, OpenShift AI, vLLM, etc.)
- Prediction validation results with tolerance analysis
- Platform detection success rate
- Model validation duration by platform
- Top failing models with error analysis

**Target Audience:** ML engineers, MLOps teams  
**Estimated Effort:** 5-7 hours  
**Volunteer:** 🙋 **[Claim this dashboard!](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/new?labels=dashboard-proposal&template=dashboard-proposal.md&title=Advanced+Model+Validation+Dashboard)**

---

## 📋 How to Contribute a Dashboard

### **Step 1: Choose Your Platform** (15 minutes)

**Option A: OpenShift Console Dashboard (Recommended)**
- ✅ Native to OpenShift
- ✅ No additional installation required
- ✅ Uses ConfigMap format
- ✅ Integrated into Console's Observe section

**Option B: Grafana Dashboard**
- ✅ More visualization options
- ✅ Advanced query capabilities
- ⚠️ Requires Grafana Operator
- ⚠️ Additional installation step

---

### **Step 2: Create Dashboard Definition** (60-90 minutes)

#### **For OpenShift Console:**

```yaml
# config/monitoring/community/console-dashboard-<name>.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jupyter-notebook-validator-<name>
  namespace: openshift-config-managed
  labels:
    console.openshift.io/dashboard: "true"
    console.openshift.io/odc-dashboard: "true"
data:
  <name>.json: |
    {
      "dashboard": {
        "title": "Jupyter Notebook Validator - <Your Title>",
        "tags": ["jupyter", "mlops", "operator"],
        "timezone": "browser",
        "panels": [
          {
            "title": "Panel Title",
            "type": "graph",
            "targets": [
              {
                "expr": "your_promql_query_here",
                "legendFormat": "{{label}}"
              }
            ],
            "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
          }
        ]
      }
    }
```

#### **For Grafana:**

```yaml
# config/monitoring/community/grafana-dashboard-<name>.yaml
apiVersion: grafana.integreatly.org/v1beta1
kind: GrafanaDashboard
metadata:
  name: jupyter-notebook-validator-<name>
  labels:
    app: jupyter-notebook-validator-operator
spec:
  instanceSelector:
    matchLabels:
      dashboards: "grafana"
  json: |
    {
      "dashboard": {
        "title": "Jupyter Notebook Validator - <Your Title>",
        "panels": [...]
      }
    }
```

---

### **Step 3: Add Example Queries** (30 minutes)

Document the PromQL queries used in your dashboard:

```markdown
## Example Queries

### Validation Success Rate
\`\`\`promql
sum(rate(notebookvalidationjob_validations_total{status="success"}[5m])) / 
sum(rate(notebookvalidationjob_validations_total[5m])) * 100
\`\`\`

### Active Pods by Phase
\`\`\`promql
sum(notebookvalidationjob_active_pods) by (phase)
\`\`\`

### Reconciliation Duration (p95)
\`\`\`promql
histogram_quantile(0.95, 
  rate(notebookvalidationjob_reconciliation_duration_seconds_bucket[5m])
)
\`\`\`
```

---

### **Step 4: Create Documentation** (30 minutes)

Create `docs/dashboards/<name>.md`:

```markdown
# <Dashboard Name>

## Overview
Brief description of what this dashboard shows and who should use it.

## Use Cases
- Use case 1
- Use case 2

## Panels

### Panel 1: <Name>
- **Metric:** `metric_name`
- **Purpose:** What this panel shows
- **Interpretation:** How to read the data

## Installation

### OpenShift Console
\`\`\`bash
oc apply -f config/monitoring/community/console-dashboard-<name>.yaml
\`\`\`

### Grafana
\`\`\`bash
oc apply -f config/monitoring/community/grafana-dashboard-<name>.yaml
\`\`\`

## Screenshots
![Dashboard Screenshot](../images/dashboard-<name>.png)
```

---

### **Step 5: Add Tests** (Optional, 30 minutes)

Create test queries to verify metrics are working:

```bash
# test/dashboards/<name>_test.sh
#!/bin/bash

# Test that metrics are available
echo "Testing metrics availability..."
curl -s http://localhost:8080/metrics | grep "notebookvalidationjob_"

# Test PromQL queries
echo "Testing PromQL queries..."
oc exec -n openshift-monitoring prometheus-k8s-0 -- \
  promtool query instant http://localhost:9090 \
  'sum(rate(notebookvalidationjob_validations_total[5m]))'
```

---

### **Step 6: Submit PR** (15 minutes)

1. Fork the repository
2. Create a branch: `git checkout -b dashboard/<name>`
3. Add your files:
   - `config/monitoring/community/console-dashboard-<name>.yaml` or `grafana-dashboard-<name>.yaml`
   - `docs/dashboards/<name>.md`
   - `test/dashboards/<name>_test.sh` (optional)
   - Screenshots in `docs/images/dashboard-<name>.png`
4. Update `docs/COMMUNITY_OBSERVABILITY.md` to mark your dashboard as "In Progress"
5. Submit PR with title: `[Dashboard] Add <Name> Dashboard`

---

## 🎨 Dashboard Design Guidelines

### **Visual Hierarchy**
1. **Top Row**: Key metrics (success rate, error rate, active jobs)
2. **Middle Rows**: Detailed graphs (duration, trends, distributions)
3. **Bottom Rows**: Drill-down panels (errors, logs, traces)

### **Color Coding**
- **Green**: Success, healthy, normal
- **Yellow**: Warning, degraded, attention needed
- **Red**: Error, critical, action required
- **Blue**: Informational, neutral

### **Panel Types**
- **Stat**: Single number (success rate, active pods)
- **Graph**: Time series (duration, trends)
- **Table**: Detailed data (errors, logs)
- **Heatmap**: Distribution (latency, size)

---

## 📊 Available Metrics

### **Core Metrics** (Available Now - Phase 7 Complete ✅)
```promql
# Reconciliation Metrics
notebookvalidationjob_reconciliation_duration_seconds{namespace, result}
notebookvalidationjob_reconciliation_errors_total{namespace, error_type}

# Validation Metrics
notebookvalidationjob_validations_total{namespace, status}
notebookvalidationjob_active_pods{namespace, phase}

# Git Operations Metrics
notebookvalidationjob_git_clone_duration_seconds{namespace, auth_type}
notebookvalidationjob_pod_creations_total{namespace, result}

# Model Validation Metrics (✅ NEW in Phase 7)
notebookvalidationjob_model_validation_duration_seconds{namespace, platform, result}
notebookvalidationjob_model_health_checks_total{namespace, platform, status}
notebookvalidationjob_prediction_validations_total{namespace, platform, result}
notebookvalidationjob_platform_detection_duration_seconds{namespace, platform, detected}
```

See `internal/controller/metrics.go` for complete metric definitions.

---

## 💬 Community Support

- **Slack**: [#jupyter-notebook-validator](https://kubernetes.slack.com/archives/jupyter-notebook-validator)
- **Office Hours**: First Tuesday of each month, 10 AM ET
- **GitHub Discussions**: [Observability Category](https://github.com/tosin2013/jupyter-notebook-validator-operator/discussions/categories/observability)
- **Email**: jupyter-validator-maintainers@example.com

---

## 📅 Roadmap

### Release 1.0 (Current)
- ✅ Operator Health Dashboard
- ✅ Notebook Validation Performance Dashboard
- ✅ Resource Utilization Dashboard
- ✅ Git Operations Dashboard

### Release 1.1 (Planned - Q1 2026)
- 🔴 Multi-Cluster Dashboard (community)
- 🔴 Cost Optimization Dashboard (community)
- 🔴 Security & Compliance Dashboard (community)
- 🔴 Developer Experience Dashboard (community)
- 🔴 Advanced Model Validation Dashboard (community)

### Release 1.2+ (Community-Driven)
- 🎯 Your dashboard here!

---

**🎉 We can't wait to see what you build!**

