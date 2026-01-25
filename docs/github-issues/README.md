# GitHub Issues for jupyter-notebook-validator-operator

This directory contains templates for GitHub issues that would add significant value to the jupyter-notebook-validator-operator project. Each issue is fully documented with problem statements, implementation details, use cases, and timelines.

## How to Use These Templates

1. **Read the issue template** to understand the feature request
2. **Copy the content** to create a new GitHub issue
3. **Adjust as needed** based on community feedback
4. **Add appropriate labels** as suggested in each template

## Available Feature Requests

### Priority: High

#### 1. [ArgoCD Integration for GitOps Workflows](./argocd-integration-feature-request.md)

**Status:** Proposed (ADR-048)
**Timeline:** 8-10 weeks (4 phases)
**Labels:** `enhancement`, `argocd`, `gitops`, `integration`, `help wanted`

**Summary:**
Enable complete GitOps workflows by integrating with ArgoCD. Includes health assessment, post-success resource hooks, sync wave coordination, application status integration, and notifications.

**Key Features:**
- ✅ **Feature 1 (Ready Now):** Health assessment configuration for ArgoCD UI
- 🔄 **Feature 2:** Auto-restart InferenceServices when models are trained
- 🔄 **Feature 3:** Sync wave blocking to prevent race conditions
- 🔄 **Feature 4:** Aggregated status reporting in ArgoCD
- 🔄 **Feature 5:** Notification hooks for failures

**Why This Matters:**
Solves real production issue in openshift-aiops-platform where model training completes but InferenceServices don't reload models. Eliminates manual intervention and enables fully declarative ML workflows.

**Related:**
- ADR-048: ArgoCD Integration Strategy
- docs/ARGOCD_INTEGRATION.md

---

#### 2. [Model-Aware Validation for ML Workflows](./model-aware-validation-feature-request.md)

**Status:** Proposed (ADR-020)
**Timeline:** 12 weeks
**Labels:** `enhancement`, `model-validation`, `kserve`, `ml-ops`, `help wanted`

**Summary:**
Add optional validation of notebooks against deployed ML models (KServe, OpenShift AI, vLLM, etc.). Validates platform readiness, model health, and prediction consistency.

**Key Features:**
- ✅ Platform readiness checks (KServe, OpenShift AI installed)
- ✅ Model health validation (InferenceService status)
- ✅ Prediction consistency testing (actual vs expected outputs)
- ✅ Support for 8+ model serving platforms
- ✅ Two-phase validation (clean + existing environments)

**Why This Matters:**
Closes gap where notebooks execute successfully in isolation but fail when interacting with production model serving infrastructure. Enables automated integration testing for ML workflows.

**Related:**
- ADR-020: Model-Aware Validation Strategy
- Metrics TODO in internal/controller/metrics.go:172-180

---

### Priority: Medium-High

#### 3. [Smart Pod Recovery with Build Strategy Fallback](./smart-pod-recovery-feature-request.md)

**Status:** Proposed (ADR-026)
**Timeline:** 6 weeks
**Labels:** `enhancement`, `reliability`, `pod-recovery`, `build-strategy`, `help wanted`

**Summary:**
Implement intelligent pod failure detection with automatic recovery and build strategy fallback (S2I → Tekton → Pre-built image).

**Key Features:**
- ✅ Classify failures (ImagePullBackOff, SCC violations, CrashLoopBackOff, etc.)
- ✅ Smart recovery actions (fallback images, remove init containers)
- ✅ Build strategy fallback chain
- ✅ Clear error messages with suggested actions
- ✅ Reduced manual intervention

**Why This Matters:**
Current simple retry logic (delete + recreate with same config) wastes attempts on persistent failures. Smart recovery reduces operational burden and improves success rates in production.

**Related:**
- ADR-026: Smart Validation Pod Recovery
- ADR-011: Error Handling and Retry Strategy
- ADR-023: S2I Build Strategy
- ADR-031: Tekton Build Strategy

---

### Priority: Medium

#### 4. [Automatic Tekton Git Credentials Synchronization](./tekton-git-credentials-sync.md)

**Status:** TODO (ADR-042)
**Timeline:** 4 weeks
**Labels:** `enhancement`, `tekton`, `credentials`, `automation`, `good first issue`

**Summary:**
Add automatic synchronization of Tekton secrets when source git credentials are updated. Fixes stale credentials in Tekton builds.

**Key Features:**
- ✅ Watch source secrets for changes
- ✅ Auto-update Tekton secrets when source changes
- ✅ GitOps-friendly credential management
- ✅ Supports External Secrets Operator
- ✅ Enables secure credential rotation

**Why This Matters:**
Currently, updating git credentials requires manually updating both the source secret AND the Tekton secret. This breaks GitOps workflows and causes build failures from stale credentials.

**Related:**
- ADR-042: Automatic Tekton Git Credentials Conversion
- Code TODO: pkg/build/tekton_strategy.go:252

---

## Implementation Priority Recommendation

Based on impact, effort, and dependencies:

### Phase 1: Quick Wins (Weeks 1-4)
1. **ArgoCD Integration - Feature 1** (Health Assessment)
   - ⏱️ 1-2 weeks
   - 💡 Documentation-only, no code changes
   - 🎯 Immediate value for users

2. **Tekton Git Credentials Sync**
   - ⏱️ 4 weeks
   - 💡 Clear TODO, well-scoped
   - 🎯 Good first issue for contributors

### Phase 2: High-Impact Features (Weeks 5-12)
3. **ArgoCD Integration - Feature 2** (Resource Hooks)
   - ⏱️ 2-3 weeks
   - 💡 Solves real production issue
   - 🎯 Enables auto-reload of InferenceServices

4. **Smart Pod Recovery**
   - ⏱️ 6 weeks
   - 💡 Improves reliability across the board
   - 🎯 Reduces operational burden

### Phase 3: Advanced Features (Weeks 13-24)
5. **Model-Aware Validation**
   - ⏱️ 12 weeks
   - 💡 Most complex, highest value for ML workflows
   - 🎯 Differentiator for enterprise adoption

6. **ArgoCD Integration - Features 3, 4, 5** (Sync Waves, Status, Notifications)
   - ⏱️ 4-6 weeks
   - 💡 Completes GitOps story
   - 🎯 Enterprise-grade observability

---

## Related Documentation

- **ADRs (Architecture Decision Records):** `docs/adrs/`
  - ADR-020: Model-Aware Validation Strategy
  - ADR-026: Smart Validation Pod Recovery
  - ADR-042: Automatic Tekton Git Credentials Conversion
  - ADR-048: ArgoCD Integration Strategy

- **User Guides:**
  - docs/ARGOCD_INTEGRATION.md - ArgoCD integration guide (Feature 1 ready to use)
  - docs/END_TO_END_ML_WORKFLOW.md - ML workflow examples

- **Development Guides:**
  - docs/DEVELOPMENT.md - Development setup
  - docs/TESTING_GUIDE.md - Testing strategy

---

## Contributing

Interested in implementing one of these features?

1. **Read the issue template** to understand scope and design
2. **Check related ADRs** for architectural context
3. **Comment on the GitHub issue** (once created) to claim it
4. **Follow the implementation plan** outlined in the issue
5. **Submit a PR** with tests and documentation

---

## Questions or Feedback?

- Open a discussion in the GitHub repository
- Tag @tosin2013 (operator maintainer)
- Join community meetings (if available)

---

## Issue Status Legend

- ✅ **Ready to implement** - Design approved, can start coding
- 🔄 **Proposed** - Design complete, awaiting approval
- 💡 **Good first issue** - Well-scoped, good for new contributors
- ⏱️ **Estimated timeline** - Based on ADR implementation plans

---

**Last Updated:** 2026-01-24
