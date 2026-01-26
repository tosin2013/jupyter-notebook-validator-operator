# Operator Release Guide

This document outlines the standards and procedures for releasing the Jupyter Notebook Validator Operator to community operator catalogs.

## Metadata Standards

All operator releases **MUST** use the following consistent metadata:

### Provider Information

```yaml
provider:
  name: Decision Crafters
  url: https://www.decisioncrafters.com/
```

### Maintainer Information

```yaml
maintainers:
- email: takinosh@redhat.com
  name: Tosin Akinosho
```

### Icon/Thumbnail

Use the **standard operator icon** from version 1.0.3-ocp4.19. The icon base64 data starts with:

```
iVBORw0KGgoAAAANSUhEUgAAAIAAAACACAIAAABMXPacAAAABG...
```

**Important:** Do NOT use personal GitHub avatars or other icons. Always use the Decision Crafters branded icon.

---

## Git Commit Standards

### Author Configuration

Before making commits for operator releases:

```bash
git config user.email "takinosh@redhat.com"
git config user.name "Tosin Akinosho"
```

### DCO Sign-off

All commits **MUST** include a DCO (Developer Certificate of Origin) sign-off:

```bash
git commit -s -m "Your commit message"
```

The sign-off line must match the author email:
```
Signed-off-by: Tosin Akinosho <takinosh@redhat.com>
```

---

## Community Operators Submission

### Repository: k8s-operatorhub/community-operators

1. Fork the repository
2. Create a branch: `add-jupyter-validator-{VERSION}`
3. Copy bundle to: `operators/jupyter-notebook-validator-operator/{VERSION}/`
4. Include `ci.yaml` in the operator directory
5. PR Title: `operator jupyter-notebook-validator-operator ({VERSION})`

### Repository: redhat-openshift-ecosystem/community-operators-prod

**Important:** This repository requires **ONE bundle per PR**.

1. Fork the repository
2. Create a **separate branch for each version**: `add-jupyter-validator-{VERSION}`
3. Copy bundle to: `operators/jupyter-notebook-validator-operator/{VERSION}/`
4. Include `ci.yaml` in the operator directory
5. PR Title: `operator jupyter-notebook-validator-operator ({VERSION})`

**Do NOT submit multiple versions in a single PR to community-operators-prod.**

---

## Pre-Release Checklist

Before submitting to community operators, verify:

- [ ] `provider.name` is "Decision Crafters"
- [ ] `provider.url` is "https://www.decisioncrafters.com/"
- [ ] `maintainers[0].email` is "takinosh@redhat.com"
- [ ] `maintainers[0].name` is "Tosin Akinosho"
- [ ] Icon matches the standard icon (from 1.0.3-ocp4.19)
- [ ] `containerImage` annotation matches the actual operator image
- [ ] Git author email is "takinosh@redhat.com"
- [ ] Commit includes DCO sign-off

### Verification Commands

```bash
# Check provider
grep -A2 "provider:" bundle/manifests/*.clusterserviceversion.yaml

# Check maintainers
grep -A2 "maintainers:" bundle/manifests/*.clusterserviceversion.yaml

# Check icon (first 50 chars)
grep "base64data:" bundle/manifests/*.clusterserviceversion.yaml | sed 's/.*base64data: //' | cut -c1-50

# Check containerImage annotation
grep "containerImage:" bundle/manifests/*.clusterserviceversion.yaml
```

---

## Release Scripts

### Full Pre-submission Test

```bash
./scripts/full-pre-submission-test.sh
```

### Local E2E Test

```bash
./scripts/local-e2e-test.sh [tier1|tier2|tier3|metrics|webhook|all|basic]
```

### Pre-submission Validation

```bash
./scripts/pre-submission-validate.sh
```

---

## Version History

| Version | OpenShift Compatibility | Notes |
|---------|------------------------|-------|
| 1.0.2 | 4.18+ | Initial release |
| 1.0.3-ocp4.19 | 4.19 | OCP 4.19 specific |
| 1.0.4 | 4.18+ | Feature updates |
| 1.0.5 | 4.18, 4.19, 4.20 | Universal release |

---

## Troubleshooting

### DCO Check Failed

If you see "The sign-off is missing":

```bash
# Rebase and add sign-off
git rebase HEAD~1 --signoff
git push --force-with-lease origin your-branch
```

### containerImage Annotation Mismatch

Ensure the `containerImage` annotation in the CSV matches the actual operator image tag:

```yaml
metadata:
  annotations:
    containerImage: quay.io/takinosh/jupyter-notebook-validator-operator:1.0.5
```

### Multiple Bundles in community-operators-prod

If you see "The PR affects more than one bundle", create separate PRs for each version.
