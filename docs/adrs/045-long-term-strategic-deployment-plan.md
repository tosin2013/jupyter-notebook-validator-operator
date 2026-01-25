# ADR 045: Long-Term Strategic Deployment Plan for Production Cluster

## Status
Accepted

## Context

The Jupyter Notebook Validator Operator is ready for deployment to a production OpenShift 4.18.21 cluster with Red Hat OpenShift AI (RHODS) 2.22.2. This ADR documents the strategic plan for deploying, integrating, testing, and operationalizing the operator in a phased approach.

### Current Environment Assessment

**Cluster Infrastructure:**
- **Platform**: OpenShift 4.18.21 (Kubernetes 1.31.10)
- **Cluster**: `api.cluster-wsvr7.wsvr7.sandbox930.opentlc.com:6443`
- **User**: `system:admin` (cluster-admin privileges)
- **Nodes**: 
  - 3 control-plane/master nodes
  - 2 standard worker nodes
  - 1 GPU-enabled worker node (NVIDIA GPU, 1 GPU allocatable)

**Installed Components:**
- **RHODS 2.22.2** (Red Hat OpenShift AI Self-Managed)
  - KServe v0.14.0 (Serverless mode, Managed)
  - ModelMesh Serving v0.12.0 (Managed)
  - Knative Serving (Managed)
  - Data Science Pipelines (Kubeflow Pipelines 2.5.0)
  - Workbenches (Kubeflow Notebook Controller 1.10.0)
  - Dashboard (Managed)
- **Operators**:
  - OpenShift Pipelines (Tekton)
  - OpenShift GitOps (ArgoCD)
  - Serverless Operator (Knative)
  - Service Mesh Operator
  - NVIDIA GPU Operator
  - Cert Manager
  - Authorino Operator
  - DevWorkspace Operator
  - Web Terminal

**GPU Resources:**
- 1 NVIDIA GPU on worker-gpu node
- 3.5 CPU cores, 14GB memory allocatable
- GPU scheduling enabled via NVIDIA GPU Operator

**Security Context:**
- SCCs available: `restricted`, `restricted-v2`, `anyuid`, `privileged`, `hostmount-anyuid`
- Default SCC: `restricted-v2` for most workloads
- RBAC enabled with cluster-admin access for deployment

**Model Serving Capabilities:**
- KServe InferenceServices (CRD available)
- ServingRuntimes (CRD available)
- Knative Services for serverless inference
- No active InferenceServices currently deployed

### Strategic Objectives

1. **Seamless Integration**: Deploy operator without disrupting existing RHODS workloads
2. **Model-Aware Validation**: Leverage KServe and ModelMesh for notebook validation against deployed models
3. **GPU Utilization**: Enable GPU-accelerated notebook validation for ML/AI workloads
4. **Production Readiness**: Implement comprehensive monitoring, security, and reliability controls
5. **Team Adoption**: Create documentation and training for data science and platform teams

## Decision

We will implement a **6-stage phased deployment plan** over 12-16 weeks, with each stage building on the previous one. This approach minimizes risk, enables iterative learning, and ensures production readiness.

### Stage 0: Catalog Validation & Fixing (Week 0 - IMMEDIATE) 🔥 IN PROGRESS

**Objectives:**
- Confirm OperatorHub.io catalog installation failure on validation cluster
- Fix volume attachment issues on main cluster (development)
- Update catalog annotations for proper version range support
- Validate fixed catalog on validation cluster

**Workflow**: Development-first on main cluster, validation-second on validation cluster

**Main Cluster** (api.cluster-wsvr7.wsvr7.sandbox930.opentlc.com):
- Primary development and testing environment
- Fix volume attachment issues with PVC/SCC/RBAC
- Test all operator features before catalog publication

**Validation Cluster** (api.cluster-hh8nc.hh8nc.sandbox5424.opentlc.com):
- **Updated**: OpenShift 4.20.5 (was 4.19.19) - Kubernetes 1.33.5
- Validate OperatorHub.io catalog installation
- Confirm operator appears and installs correctly
- Test basic functionality only
- Test Tekton v1 API support (OpenShift Pipelines 1.20+)

**Tasks:**
1. ✅ Test OperatorHub.io install on validation cluster (confirmed failure)
2. ✅ Fix volume attachment issues on main cluster (tested on validation cluster)
3. ✅ Identify bundle versioning issue (community-operators maintainer feedback)
4. ⏳ Fix consecutive upgrade chain (1.0.7 → 1.0.8 → 1.0.9) - **IN PROGRESS**
5. ⏳ Update catalog version annotations (v4.18-v4.20 ranges)
6. ⏳ Validate fixed catalog on validation cluster (OpenShift 4.20.5)

**Success Criteria:**
- Volume attachment works on main cluster
- Operator appears in OperatorHub on validation cluster
- Installation succeeds from catalog
- Basic notebook validation works

**Deliverables:**
- ✅ Catalog issues analysis (CATALOG-ISSUES-ANALYSIS.md)
- ✅ Development workflow documentation (DEVELOPMENT-VALIDATION-WORKFLOW.md)
- ✅ Fixed volume attachment on main cluster (tested on validation cluster)
- ✅ ADR-047: Fix Bundle Versioning for Consecutive Upgrade Chain
- ✅ Bundle Version Fix Plan (BUNDLE-VERSION-FIX-PLAN.md)
- ⏳ Rebuilt bundles with consecutive versions (1.0.7, 1.0.8, 1.0.9) - **IN PROGRESS**
- ⏳ Updated catalog with correct annotations
- ⏳ Validated catalog on validation cluster (OpenShift 4.20.5)

**Status**: IN PROGRESS - Fixing Bundle Versioning (2025-12-03)

### Stage 1: Cluster Environment Assessment & Baseline (Weeks 1-2) ✅ COMPLETE

**Objectives:**
- Document cluster infrastructure and capabilities
- Assess RHODS components and integration points
- Establish security and RBAC baseline

**Deliverables:**
- ✅ Cluster infrastructure documentation
- ✅ RHODS 2.22.2 component inventory
- ✅ GPU resource assessment
- ✅ Pipeline and CI/CD audit
- ✅ Security baseline documentation
- ✅ This ADR (045)

**Status**: COMPLETE

### Stage 2: Operator Deployment & Integration Planning (Weeks 3-4)

**Objectives:**
- Deploy operator to cluster in dedicated namespace
- Configure RBAC and service accounts
- Integrate with existing RHODS infrastructure
- Validate basic functionality with Tier 1 notebooks

**Tasks:**
1. Create operator namespace (`jupyter-notebook-validator-operator-system`)
2. Deploy CRDs and operator controller
3. Configure RBAC roles and bindings
4. Set up service accounts with appropriate SCCs
5. Deploy sample NotebookValidationJobs (Tier 1: simple Python)
6. Validate pod creation and execution
7. Test Git integration (HTTPS and SSH)
8. Configure credential management (Secrets)

**Success Criteria:**
- Operator running and reconciling CRs
- Tier 1 notebooks execute successfully
- Git clone working with credentials
- Pod logs accessible and sanitized
- No disruption to existing RHODS workloads

**Risks:**
- SCC conflicts with notebook execution pods
- Resource contention on worker nodes
- Git credential handling issues

**Mitigation:**
- Use `anyuid` SCC for validation pods if needed
- Set resource requests/limits on validation pods
- Test credential injection thoroughly before production use

### Stage 3: Testing Framework Implementation (Weeks 5-7)

**Objectives:**
- Implement comprehensive testing strategy (Tier 1/2/3)
- Integrate with test notebook repository
- Enable golden notebook comparison
- Test build integration (S2I and Tekton)

**Tasks:**
1. Deploy test notebooks from `jupyter-notebook-validator-test-notebooks` repo
2. Implement Tier 1 tests (< 30s: hello world, basic assertions)
3. Implement Tier 2 tests (1-5 min: pandas, numpy, data analysis)
4. Implement Tier 3 tests (5-15 min: model training, ML pipelines)
5. Configure golden notebook comparison with tolerances
6. Test S2I build integration for notebooks with requirements.txt
7. Test Tekton build integration for custom Dockerfiles
8. Validate build caching and optimization
9. Test credential injection patterns (ESO, Vault if available)

**Success Criteria:**
- All 3 test tiers execute successfully
- Golden notebook comparison working with configurable tolerances
- S2I builds complete successfully on OpenShift
- Tekton pipelines create custom images
- Build artifacts cached and reused
- Credentials injected securely without leaking in logs

**Risks:**
- Build timeouts for complex dependencies
- PVC conflicts for concurrent Tekton builds
- Golden notebook comparison false positives
- Test execution time exceeding limits

**Mitigation:**
- Implement unique PVC naming for Tekton builds (ADR-040)
- Configure appropriate timeouts per test tier
- Use numeric tolerances for floating-point comparisons
- Implement smart retry logic for transient failures

### Stage 4: Production Readiness & Observability (Weeks 8-10)

**Objectives:**
- Implement monitoring and alerting
- Deploy OpenShift Console dashboards
- Configure Prometheus metrics
- Implement production-grade security controls
- Enable audit logging

**Tasks:**
1. Deploy Prometheus ServiceMonitor for operator metrics
2. Create OpenShift Console dashboard for validation jobs
3. Configure Grafana dashboards (if available)
4. Implement alerting rules for operator health
5. Configure log aggregation (OpenShift Logging)
6. Enable audit logging for security events
7. Implement secret rotation policies
8. Configure Pod Security Standards
9. Set up RBAC for multi-tenant access
10. Document runbooks for common issues

**Success Criteria:**
- Prometheus metrics exposed and scraped
- OpenShift Console dashboard showing job status
- Alerts firing for operator failures
- Logs aggregated and searchable
- Secrets rotated automatically
- RBAC configured for data science teams
- Runbooks documented and tested

**Risks:**
- Metric cardinality explosion
- Dashboard performance issues
- Alert fatigue from false positives
- Log volume overwhelming storage

**Mitigation:**
- Use metric labels judiciously
- Implement dashboard caching
- Tune alert thresholds based on baseline
- Configure log retention policies

### Stage 5: Advanced Features & Optimization (Weeks 11-13)

**Objectives:**
- Implement model-aware validation with KServe
- Enable GPU-accelerated notebook execution
- Optimize build performance
- Implement advanced comparison strategies

**Tasks:**
1. Deploy sample KServe InferenceService for testing
2. Implement model discovery for KServe and ModelMesh
3. Configure notebook validation against model endpoints
4. Enable GPU scheduling for validation pods
5. Test GPU-accelerated notebooks (PyTorch, TensorFlow)
6. Implement build optimization (layer caching, multi-stage builds)
7. Configure advanced comparison strategies (semantic diff, metric-based)
8. Implement smart error messages and user feedback (ADR-030)
9. Test concurrent validation jobs at scale
10. Optimize resource utilization and pod scheduling

**Success Criteria:**
- Notebooks validate against deployed KServe models
- GPU-accelerated notebooks execute successfully
- Build times reduced by 50% through optimization
- Advanced comparison strategies working
- 10+ concurrent validation jobs without resource contention
- Smart error messages guide users to fixes

**Risks:**
- GPU resource exhaustion
- Model endpoint discovery failures
- Build optimization breaking compatibility
- Comparison strategy false negatives

**Mitigation:**
- Implement GPU resource quotas
- Fallback to CPU execution if GPU unavailable
- Test build optimization thoroughly before rollout
- Provide multiple comparison strategies with user selection

### Stage 6: Documentation & Knowledge Transfer (Weeks 14-16)

**Objectives:**
- Create comprehensive documentation
- Train data science and platform teams
- Establish support processes
- Plan for continuous improvement

**Tasks:**
1. Create user guide for data scientists
2. Create operator guide for platform engineers
3. Document integration patterns with RHODS
4. Create video tutorials and demos
5. Conduct training sessions for teams
6. Establish support channels (Slack, tickets)
7. Document troubleshooting procedures
8. Create FAQ and common issues guide
9. Plan roadmap for future enhancements
10. Establish feedback loop with users

**Success Criteria:**
- Documentation complete and published
- 80%+ of data science team trained
- Support channels established and staffed
- Troubleshooting guide covers 90% of common issues
- Roadmap approved by stakeholders
- Feedback mechanism in place

**Risks:**
- Documentation becoming outdated
- Low training attendance
- Support burden overwhelming team
- Feature requests exceeding capacity

**Mitigation:**
- Automate documentation generation where possible
- Record training sessions for on-demand viewing
- Establish tiered support model
- Prioritize feature requests based on impact

## Consequences

### Positive

1. **Risk Mitigation**: Phased approach allows early detection and resolution of issues
2. **Iterative Learning**: Each stage builds on lessons learned from previous stages
3. **Minimal Disruption**: Gradual rollout prevents impact to existing RHODS workloads
4. **Production Readiness**: Comprehensive testing and monitoring before full adoption
5. **Team Enablement**: Training and documentation ensure successful adoption
6. **Scalability**: Architecture supports growth in notebook validation workloads
7. **Integration**: Seamless integration with existing RHODS, KServe, and Tekton infrastructure

### Negative

1. **Timeline**: 12-16 week rollout may be slower than aggressive deployment
2. **Resource Investment**: Requires dedicated time from platform and data science teams
3. **Complexity**: Multi-stage plan requires coordination across teams
4. **Maintenance Overhead**: Ongoing monitoring and support required post-deployment

### Neutral

1. **Flexibility**: Plan can be adjusted based on feedback and changing requirements
2. **Reversibility**: Each stage can be rolled back if issues arise
3. **Documentation Burden**: Comprehensive documentation required throughout

## Alternatives Considered

### Alternative 1: Big Bang Deployment
**Description**: Deploy all features at once in a single release

**Pros**:
- Faster time to full functionality
- Single deployment event
- Less coordination overhead

**Cons**:
- Higher risk of production issues
- Difficult to isolate problems
- No opportunity for iterative learning
- Potential for widespread disruption

**Rejected**: Too risky for production environment with active RHODS workloads

### Alternative 2: Minimal Viable Product (MVP) Only
**Description**: Deploy only basic notebook execution without advanced features

**Pros**:
- Fastest time to initial value
- Minimal complexity
- Lower resource investment

**Cons**:
- Missing key features (model validation, GPU support)
- Limited adoption without advanced capabilities
- Requires future major upgrades

**Rejected**: Doesn't meet strategic objectives for model-aware validation and GPU support

### Alternative 3: External Managed Service
**Description**: Use external SaaS platform for notebook validation

**Pros**:
- No operational overhead
- Managed updates and support
- Potentially faster deployment

**Cons**:
- Data sovereignty concerns
- Limited customization
- Ongoing subscription costs
- No integration with internal KServe models

**Rejected**: Doesn't align with on-premises RHODS deployment and security requirements

## Implementation Details

### Stage Gating Criteria

Each stage must meet its success criteria before proceeding to the next stage. Stage gates include:

1. **Technical Validation**: All tasks completed and tested
2. **Security Review**: Security controls validated and documented
3. **Stakeholder Approval**: Platform and data science teams approve progression
4. **Documentation**: Stage-specific documentation complete
5. **Rollback Plan**: Documented procedure to revert changes if needed

### Resource Requirements

**Platform Team**:
- 1 senior platform engineer (50% allocation, Weeks 1-16)
- 1 platform engineer (25% allocation, Weeks 3-16)

**Data Science Team**:
- 1 ML engineer (25% allocation, Weeks 5-16 for testing and feedback)
- Data scientists (10% allocation, Weeks 14-16 for training)

**Infrastructure**:
- Dedicated namespace in OpenShift cluster
- 4-8 CPU cores for operator and validation pods
- 8-16 GB memory for validation workloads
- 50-100 GB storage for build artifacts and logs
- GPU access for Stage 5 testing

### Key Milestones

| Week | Milestone | Deliverable |
|------|-----------|-------------|
| 2 | Stage 1 Complete | Cluster assessment and ADR-045 |
| 4 | Stage 2 Complete | Operator deployed, Tier 1 tests passing |
| 7 | Stage 3 Complete | All test tiers passing, builds working |
| 10 | Stage 4 Complete | Monitoring and security controls active |
| 13 | Stage 5 Complete | Model validation and GPU support working |
| 16 | Stage 6 Complete | Documentation and training complete |

### Success Metrics

**Technical Metrics**:
- Operator uptime: > 99.5%
- Notebook validation success rate: > 95%
- Average validation time: < 5 minutes (Tier 1/2), < 15 minutes (Tier 3)
- Build success rate: > 90%
- GPU utilization: > 60% when scheduled

**Adoption Metrics**:
- Number of validation jobs per week: > 50 by Week 16
- Number of active users: > 10 data scientists by Week 16
- Documentation page views: > 100 by Week 16
- Training completion rate: > 80% of data science team

**Business Metrics**:
- Reduction in notebook debugging time: > 30%
- Increase in notebook reliability: > 40%
- Time to deploy validated notebooks: < 1 hour

## Related ADRs

- ADR-001: Operator Framework and SDK Version
- ADR-002: Platform Version Support Strategy
- ADR-008: Notebook Testing Strategy and Complexity Levels
- ADR-020: Model-Aware Validation Strategy
- ADR-023: S2I Build Integration (OpenShift)
- ADR-028: Tekton Task Strategy (Custom vs Cluster Tasks)
- ADR-030: Smart Error Messages and User Feedback
- ADR-040: Unique Build PVCs for Concurrent Tekton Builds

## References

- OpenShift 4.18 Documentation: https://docs.openshift.com/container-platform/4.18/
- Red Hat OpenShift AI 2.22 Documentation: https://access.redhat.com/documentation/en-us/red_hat_openshift_ai_self-managed/2.22
- KServe Documentation: https://kserve.github.io/website/
- Operator SDK Documentation: https://sdk.operatorframework.io/

## Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2025-12-02 | 1.0 | Platform Team | Initial strategic plan based on cluster assessment |


