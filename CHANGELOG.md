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

## [1.0.8] - 2026-04-21

**OCP Stream:** OCP 4.19+ (Kubernetes 1.32)
**Status:** Released — CI builds operator and bundle images on tag push.
**OperatorHub:** Pending PR submission to `community-operators-prod` after v1.0.7 chain confirms.

> **OCP-stream versioning**: v1.0.7 → OCP 4.18 | v1.0.8 → OCP 4.19 | v1.0.9 → OCP 4.20 | v1.0.10 → OCP 4.21

### Added
- OpenShift-native dashboard Kustomize overlays (ADR-021) — [#7](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/7)
  - `config/monitoring/kustomization.yaml` wiring all 5 OpenShift Console dashboard ConfigMaps
  - `config/monitoring/openshift-console/kustomization.yaml` for standalone deployment
  - `config/monitoring/grafana/kustomization.yaml` with ConfigMapGenerator for Grafana Operator
- Community observability contribution framework (ADR-022) — [#8](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues/8)
  - `config/monitoring/community/` directory with `kustomization.yaml` placeholder for contributors
  - `docs/dashboards/CONTRIBUTING.md` — full guide with formats, metrics table, doc template, review criteria
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
