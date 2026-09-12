# ADR-058: OpenShift E2E as Manual/Release-Only CI

**Status**: Accepted
**Date**: 2026-09-12
**Authors**: Tosin Akinosho
**Related**: ADR-033 (E2E Testing), ADR-034 (Dual Testing Strategy), ADR-056 (OCP 4.20-4.22)

## Context

The operator certifies for OpenShift 4.20, 4.21, and 4.22 (ADR-056). The CI pipeline includes an E2E workflow (`e2e-openshift.yaml`) that runs tests on a live OpenShift cluster. This workflow was triggered on every push to `main` and on PRs with the `e2e-test` label.

The problem: the maintainer does not always have a standing OpenShift cluster available. Developer Sandbox clusters expire. The configured cluster DNS record becomes stale, and the workflow fails with:

```
error: dial tcp: lookup api.cluster-xxx.sandbox.opentlc.com: no such host
```

This failure blocks CI on `main` even though the change has nothing to do with OpenShift. The existing guard (`check-cluster` job) only verified that the `OPENSHIFT_SERVER` secret was non-empty. It did not verify the cluster was reachable.

ADR-034 established a dual Kind/OpenShift strategy but did not specify whether OpenShift E2E should be a push-to-main gate.

## Decision

OpenShift E2E tests are **not a gate for day-to-day development on main**. They are required only for:

1. **Release certification**: pushes to `release-*` branches.
2. **Manual trigger**: `workflow_dispatch` for on-demand validation.
3. **Labeled PRs**: PRs with the `e2e-test` label (unchanged).

The `check-cluster` job now tests actual cluster reachability with a `curl` probe (5-second timeout) instead of only checking if the secret is non-empty. If the cluster is unreachable, the workflow skips gracefully.

Day-to-day CI on `main` uses Kind for Tier 1 E2E tests (`e2e-kind.yaml`). Kind catches approximately 90% of issues (controller logic, CRD validation, webhooks, basic notebook execution).

### Future enhancement

A follow-up issue tracks adding on-demand ROSA/IPI cluster provisioning to the manual workflow. When no cluster is available, the workflow would deploy one on AWS, run tests, and tear it down.

## Alternatives considered

### MicroShift in GitHub Actions
Single-node OpenShift on a GitHub Actions runner. Rejected: MicroShift requires 4 vCPU and 16 GB RAM, which exceeds the default runner limits (2 vCPU, 7 GB).

### CRC (CodeReady Containers) in GitHub Actions
Desktop OpenShift distribution. Rejected: CRC requires nested virtualization, which is not available on GitHub Actions runners.

### Red Hat Developer Sandbox API
Free ephemeral OpenShift clusters. Rejected: no stable programmatic API for provisioning clusters from CI. The web-based signup flow cannot be automated.

### Always-on OpenShift cluster
Maintain a persistent cluster for CI. Rejected: cost prohibitive for an open-source project. A ROSA cluster costs approximately $500/month.

### Kind-only (no OpenShift testing)
Run all tests on Kind and skip OpenShift entirely. Rejected: OpenShift-specific features (S2I builds, SCCs, ImageStreams, OpenShift Pipelines) cannot be tested on Kind. These features must be validated before release.

## Consequences

### Positive

- CI on `main` no longer fails when no OpenShift cluster is available.
- Contributors get fast feedback from Kind E2E without needing an OpenShift cluster.
- The workflow skips gracefully instead of failing hard on DNS errors.
- Release certification still requires full OpenShift validation.

### Negative

- OpenShift-specific regressions may not be caught until release certification.
- The maintainer must remember to run manual OpenShift E2E before releases.

### Neutral

- Dependabot minor version upgrades for platform dependencies (k8s.io, controller-runtime) require manual OpenShift E2E validation before merging.

## Changes

- `.github/workflows/e2e-openshift.yaml`: removed `main` from push trigger, added `curl` reachability check to `check-cluster` job.
- `.github/dependabot.yml`: updated to allow minor bumps (block major only) for platform deps. Minor bumps are validated by manual OpenShift E2E before merge.
