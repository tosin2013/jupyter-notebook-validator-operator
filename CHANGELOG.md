# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned — v1.0.10 (OCP 4.21 stream)

- Create `release-4.21` branch from main
- Run scorecard validation on OCP 4.21 / Kubernetes 1.34
- Generate bundle for v1.0.10 with `replaces: v1.0.9`
- Submit OperatorHub PR for `1.0.10/` after v1.0.9 PR merges

---

## [1.0.9] - 2026-09-15

**OCP Stream:** OCP 4.20+ (Kubernetes 1.33)
**Certified window (ADR-056):** OCP 4.20 / 4.21 / 4.22
**Status:** Released. E2E validated on ROSA HCP (OCP 4.20).
**Bundle:** `olm.skipRange: >=1.0.2 <1.0.9` | `com.redhat.openshift.versions: v4.20-v4.22`

> **OCP-stream versioning**: v1.0.7 → OCP 4.18 | v1.0.8 → OCP 4.19 | v1.0.9 → OCP 4.20 | v1.0.10 → OCP 4.21

### Added

- ADR-038: Requirements.txt auto-detection with 6-source fallback chain — [#17](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/17)
- ADR-041: Exit code validation framework with learning/development/staging/production levels — [#18](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/18)
- OperatorHub bundle submission automation script (`scripts/submit-to-operatorhub.sh`) — [#39](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/39)
- File Based Catalog (FBC) support: `make catalog-render` target and FBC documentation — [#40](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/40)
- Diataxis documentation restructure with four quadrant directories (`tutorials/`, `how-to/`, `reference/`, `explanation/`) — [#87](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/87)
- Laptop-first data scientist tutorial: `docs/tutorials/LOCAL_VALIDATION.md` — [#87](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/87)
- Dual-audience value messaging in README.md, DESIGN_DOC.md, and CSV description — [#85](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/85)
- Data scientist onboarding path in CONTRIBUTING.md — [#42](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/42)
- ADR-056: OpenShift support window 4.20-4.22 (supersedes ADR-002)
- ADR-058: OpenShift E2E as manual/release-only CI strategy
- On-demand ROSA HCP cluster provisioning for E2E tests (`scripts/setup-rosa-e2e.sh`, Terraform) — [#68](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/68)
- Tekton Pipelines install and cluster readiness audit in `setup-rosa-e2e.sh` — [#94](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/94)
- E2E ROSA runbook: `docs/how-to/E2E_ROSA_RUNBOOK.md` — [#94](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/94)
- VHS terminal demo tapes for data scientist and platform engineer audiences — [#93](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/93)
  - `demo/local-validation.tape` (data scientist local workflow)
  - `demo/openshift-deploy.tape` (platform engineer deployment)
  - `demo/laptop-to-cluster.tape` (full bridge workflow)
- GitHub Pages documentation site with MkDocs Material theme — [#99](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/99)
  - Full Diataxis navigation, search, Mermaid rendering, dark/light themes
  - Deployed at `https://tosin2013.github.io/jupyter-notebook-validator-operator/`

### Fixed

- ADR-050: S2I build status monitoring uses `OutputDockerImageReference` — [#19](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/19)
- Failing unit/integration tests in CI (BeforeSuite setup + GO_VERSION mismatch) — [#15](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/15)
- OperatorHub submission backlog: v1.0.7 and v1.0.8 submitted to both upstream repos — [#22](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/22)
- golangci-lint v2 config compatibility and Helm timeout on Kind — [#51](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/51), [#52](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/52)
- E2E OpenShift workflow hardened to skip gracefully when no cluster configured — [#20](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/20)
- OperatorHub listing blank icon (truncated PNG) and stale description — [#89](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/89)
- golangci-lint version pinned, typecheck suppressions resolved — [#24](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/24)
- E2E workflow: OpenShift internal registry fallback for image push when Quay.io is unavailable — [#96](https://github.com/tosin2013/jupyter-notebook-validator-operator/pull/96)
- E2E workflow: internal registry push permissions and Docker flag fix — [#97](https://github.com/tosin2013/jupyter-notebook-validator-operator/pull/97)

### Changed

- `com.redhat.openshift.versions` narrowed from `v4.19-v4.22` to `v4.20-v4.22` (ADR-056)
- `minKubeVersion` updated from `1.28.0` to `1.31.0`
- Go upgraded from 1.24 to 1.25; dependency bumps for controller-runtime, Tekton, CEL, gRPC
- Minimum test coverage threshold established — [#25](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/25)
- CSV description rewritten with dual-audience value messaging and Diataxis doc links
- 35 documentation files reorganized into Diataxis quadrant directories
- E2E OpenShift workflow updated with Terraform-based ROSA provisioning option

### Documentation

- Complete RELEASE.md runbook with per-version instructions — [#23](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/23)
- OpenShift cluster registration and OCP validation strategy — [#21](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/21)
- ADR-021 dashboard verification and OBSERVABILITY.md — [#26](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/26)
- ADR-022 community observability contribution pipeline verified — [#27](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/27)
- GitHub Secrets setup guide: `docs/how-to/GITHUB_SECRETS_SETUP.md` — [#94](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/94)

### Bundle Metadata

- `olm.skipRange: ">=1.0.2 <1.0.9"`
- `com.redhat.openshift.versions: "v4.20-v4.22"`
- `containerImage: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.9`

---

## [1.0.8] - 2026-04-21

**OCP Stream:** OCP 4.19+ (Kubernetes 1.32)  
**Status:** Released — CI builds operator and bundle images on tag push.  
**OperatorHub (community-operators-prod):** PR open — [tosin2013/community-operators-prod branch `add-jupyter-notebook-validator-operator-1.0.8`](https://github.com/tosin2013/community-operators-prod/tree/add-jupyter-notebook-validator-operator-1.0.8) → open PR at https://github.com/redhat-openshift-ecosystem/community-operators-prod/compare/main...tosin2013:add-jupyter-notebook-validator-operator-1.0.8  
**OperatorHub (community-operators):** PR open — [tosin2013/community-operators branch `add-jupyter-notebook-validator-operator-1.0.8`](https://github.com/tosin2013/community-operators/tree/add-jupyter-notebook-validator-operator-1.0.8) → open PR at https://github.com/k8s-operatorhub/community-operators/compare/main...tosin2013:add-jupyter-notebook-validator-operator-1.0.8

> **OCP-stream versioning**: v1.0.7 → OCP 4.18 | v1.0.8 → OCP 4.19 | v1.0.9 → OCP 4.20 | v1.0.10 → OCP 4.21  
> **Certified support window (ADR-056)**: OCP 4.20 / 4.21 / 4.22. OCP 4.18 dropped from certified support (maintenance ended 2026-08-25). OCP 4.19 best-effort.

### Added
- OpenShift-native dashboard Kustomize overlays (ADR-021) — [#7](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/7)
  - `config/monitoring/kustomization.yaml` wiring all 5 OpenShift Console dashboard ConfigMaps
  - `config/monitoring/openshift-console/kustomization.yaml` for standalone deployment
  - `config/monitoring/grafana/kustomization.yaml` with ConfigMapGenerator for Grafana Operator
- Community observability contribution framework (ADR-022) — [#8](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/8)
  - `config/monitoring/community/` directory with `kustomization.yaml` placeholder for contributors
  - `docs/how-to/DASHBOARD_CONTRIBUTING.md` — full guide with formats, metrics table, doc template, review criteria
- Comprehensive ADR-030 Level 1/2/3 test coverage — [#9](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/9)
  - `TestADR030_AllThreeLevels`: validates accurate status, root cause, and actionable guidance
  - `TestADR030_SeverityEscalation`: validates severity defaults per category

### Changed
- `Makefile VERSION` corrected from `1.1.0` → `1.0.8` (OCP-stream convention)
- `IMAGE_TAG_BASE` standardized to `quay.io/takinosh/jupyter-notebook-validator-operator`
- Helm `Chart.yaml` `version` and `appVersion` set to `1.0.8`
- Helm chart `kube-rbac-proxy` migrated `gcr.io/kubebuilder` → `quay.io/brancz:v0.18.0`
- CI `GO_VERSION` bumped `1.22` → `1.24` across all four workflows
- `go.mod` `go` directive updated `1.22.3` → `1.24`
- `bundle-validation.yaml`: added `release-4.21` to push/PR branch triggers
- `bundle-validation.yaml`: extended upgrade chain and version table to include `v1.0.10 (OCP 4.21)`
- `docs/OPENSHIFT_SUPPORT_MATRIX_AND_STRATEGY.md`: OCP 4.21 marked GA (April 2026); active window updated to 4.19/4.20/4.21
- `docs/adrs/021-openshift-native-dashboard-strategy.md`: status Proposed → Implemented
- `docs/adrs/022-community-observability-contributions.md`: status Proposed → Implemented

### Bundle Metadata
- Replaces: `jupyter-notebook-validator-operator.v1.0.7`
- `com.redhat.openshift.versions: "v4.19-v4.22"`
- Images built and pushed by CI on `v1.0.8` tag push

---

## [1.0.7] - 2026-03-19

**Status:** Ready for submission to Community Operators.
See full release notes: [`docs/RELEASE-NOTES-v1.0.7.md`](./docs/RELEASE-NOTES-v1.0.7.md)

### Added
- Extended OpenShift support range to **4.18-4.22** (previously 4.18-4.20)
- `com.redhat.openshift.versions` annotation for proper catalog filtering
- Kubernetes 1.34 (OCP 4.21) and 1.35 (OCP 4.22) support

### Fixed
- **kube-rbac-proxy image**: migrated from deprecated `gcr.io/kubebuilder/kube-rbac-proxy:v0.16.0`
  to `quay.io/brancz/kube-rbac-proxy:v0.18.0` — resolves `ImagePullBackOff` errors
- **Channel mismatch**: updated bundle metadata from `alpha` to `stable` channel

### Images
- `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.7`
- `quay.io/takinosh/jupyter-notebook-validator-operator-bundle:1.0.7`

### Bundle Metadata
- Replaces: `jupyter-notebook-validator-operator.v1.0.6`
- Channel: `stable`
- minKubeVersion: `1.28.0`

---

## [1.0.5] - 2026-01-xx

### Added
- OKD 4.22 support and validation
- Expanded test notebook suite

### Fixed
- RBAC scope corrections for webhook permissions

---

## [1.0.4] - 2025-11-xx

*OCP-stream variants: `v1.0.4-ocp4.18`, `v1.0.4-ocp4.19`, `v1.0.4-ocp4.20`*

### Added
- Multi-OCP-version parallel release stream (`ocp4.18`, `ocp4.19`, `ocp4.20` tags)
- Webhook certificate rotation via cert-manager

### Fixed
- CSV upgrade chain: `v1.0.3 → v1.0.4`

---

## [1.0.3] - 2025-10-xx

*OCP-stream variants: `v1.0.3-ocp4.18`, `v1.0.3-ocp4.19`*

### Fixed
- Operator icon standardized to Decision Crafters branding (removed personal GitHub avatars)
- Provider information corrected to `Decision Crafters / decisioncrafters.com`

---

## [1.0.2] - 2025-09-xx

*OCP-stream variant: `v1.0.2-ocp4.18`*

### Added
- Initial OpenShift 4.18 (Kubernetes 1.31) support

---

## [1.0.1] - 2025-08-xx

*OCP-stream variant: `v1.0.1-ocp4.18`*

### Fixed
- Initial bug fixes post initial release

---

## [1.0.0] - 2025-07-xx

*OCP-stream variants: `v1.0.0-ocp4.18`, `v1.0.0-ocp4.19`, `v1.0.0-ocp4.20`*

See release notes: [`docs/RELEASE_NOTES_v1.0.0-ocp4.20.md`](./docs/RELEASE_NOTES_v1.0.0-ocp4.20.md)

### Added
- Initial release of Jupyter Notebook Validator Operator
- Kubernetes operator for validating Jupyter notebooks via Tekton Pipelines
- Admission webhook for notebook submission validation
- OLM bundle for OperatorHub submission (stable channel)
- Support for OpenShift 4.18-4.20 / Kubernetes 1.28+
- Tekton Pipelines v0.68+ integration (OpenShift Pipelines 1.20+)
- E2E test suite with Kind cluster (Tier 1)
- cert-manager integration for webhook TLS
- `quay.io/takinosh/jupyter-notebook-validator-operator:1.0.0`
