# E2E ROSA Cluster Setup Runbook

**Owner**: jupyter-notebook-validator-operator maintainers
**Risk Level**: Medium
**Last Updated**: 2026-09-14
**Version**: 1.0.0

---

## Quick Reference

| Attribute | Value |
|-----------|-------|
| **Execution Time** | ~45 minutes (full setup) or ~10 minutes (deploy only) |
| **Impact Window** | No downtime (new cluster) |
| **Rollback Time** | ~15 minutes (terraform destroy) |
| **Prerequisites** | AWS account with ROSA enabled, OCM token, GitHub repo access |

---

## Scope and Use Case

### When to Use This Runbook

Use this runbook to provision a ROSA HCP (Hosted Control Plane) cluster and deploy the operator for end-to-end testing. Use it before you trigger the `e2e-openshift.yaml` GitHub Actions workflow.

**Triggers**:
- You need to run the OpenShift E2E test suite
- You need to validate a release candidate on a real cluster
- The `e2e-openshift.yaml` workflow uses the `provision_cluster` option

### Expected Outcome

A ROSA HCP cluster runs with all prerequisites installed:
- cert-manager (webhook TLS)
- Tekton Pipelines (Tier 2, 3, 5 builds)
- Operator deployed with webhooks
- GitHub secrets configured for CI

### What This Does NOT Cover

- KServe or model serving setup (Tier 3 model validation requires manual KServe install)
- Production cluster hardening
- Multi-cluster federation

---

## Test Tier Dependency Matrix

Each E2E tier requires different cluster components. Use this matrix to determine which components you must install.

| Tier | Name | Build Strategy | cert-manager | Tekton | S2I (BuildConfig) | pipelines-scc | Notes |
|------|------|----------------|:------------:|:------:|:-----------------:|:-------------:|-------|
| 1 | Simple | None | Yes | No | No | No | Basic notebook execution |
| 2 | Intermediate | `tekton` | Yes | **Yes** | No | **Yes** | Data analysis with pandas/numpy |
| 3 | Complex | `tekton` | Yes | **Yes** | No | **Yes** | ML training, credential injection |
| 4 | S2I | `s2i` | Yes | No | **Yes** | **Yes** | OpenShift Source-to-Image builds |
| 5 | Volumes | `tekton` | Yes | **Yes** | No | **Yes** | PVC-mounted notebook execution |

**Minimum for Tier 1 only**: cert-manager + operator.
**Full coverage (all tiers)**: cert-manager + Tekton Pipelines + operator + pipelines-scc.

---

## Prerequisites

### Required Access and Permissions

- [ ] AWS account with ROSA enabled ([enable ROSA billing](https://console.aws.amazon.com/rosa/home#/get-started))
- [ ] OCM token from <https://console.redhat.com/openshift/token>
- [ ] AWS CLI configured (`aws sts get-caller-identity` succeeds)
- [ ] OCM CLI logged in (`ocm whoami` succeeds)
- [ ] GitHub repository write access for secrets

### Required Tools

The setup script installs these automatically with `--tools-only`:

- [ ] Go 1.25+
- [ ] ROSA CLI 1.2.47+
- [ ] Terraform 1.9+
- [ ] `oc` (OpenShift CLI)
- [ ] `gh` (GitHub CLI, authenticated)

**Install all tools**:
```bash
./scripts/setup-rosa-e2e.sh --tools-only
```

### GitHub Secrets

The E2E workflow requires these secrets. Run `./scripts/setup-rosa-e2e.sh --secrets` to see current values.

| Secret | Purpose | Required For |
|--------|---------|-------------|
| `OPENSHIFT_SERVER` | Cluster API URL | All tiers |
| `OPENSHIFT_TOKEN` | Cluster auth token | All tiers |
| `QUAY_USERNAME` | Quay.io login for pushing operator images | Workflow image build |
| `QUAY_PASSWORD` | Quay.io robot token or password | Workflow image build |
| `TEST_REPO_TOKEN` | GitHub PAT with repo scope for test notebooks | Private test repo access |

**Terraform provisioning secrets** (only when `provision_cluster=true`):

| Secret | Purpose |
|--------|---------|
| `AWS_ACCESS_KEY_ID` | AWS credentials |
| `AWS_SECRET_ACCESS_KEY` | AWS credentials |
| `AWS_ACCOUNT_ID` | AWS account ID |
| `ROSA_TOKEN` | OCM offline token |
| `ROSA_INSTALLER_ROLE_ARN` | Installer role ARN |
| `ROSA_SUPPORT_ROLE_ARN` | Support role ARN |
| `ROSA_WORKER_ROLE_ARN` | Worker role ARN |
| `ROSA_OIDC_CONFIG_ID` | OIDC config ID |
| `ROSA_CREATOR_ARN` | Creator IAM ARN |

---

## Pre-Flight Checks

**STOP**: Do NOT proceed unless ALL checks pass.

### Check 1: AWS Identity

```bash
aws sts get-caller-identity
```

Pass criteria: Returns your AWS account ID and ARN.
Fail action: Run `aws configure` or check your AWS credentials.

### Check 2: OCM Login

```bash
ocm whoami
```

Pass criteria: Returns your OCM user details.
Fail action: Run `ocm login --use-device-code` and follow the browser prompt.

### Check 3: ROSA Billing

```bash
aws service-quotas get-service-quota \
  --service-code rosa \
  --quota-code L-FC39B1B5 \
  --query 'Quota.Value'
```

Pass criteria: Returns a value greater than 0.
Fail action: Enable ROSA at <https://console.aws.amazon.com/rosa/home#/get-started>.

---

## Procedure

### Option A: Full Setup (New Cluster)

This option provisions a new ROSA HCP cluster with all dependencies.

```bash
./scripts/setup-rosa-e2e.sh
```

The script runs these steps in order:
1. Preflight checks (AWS, OCM)
2. Tool installation (Go, ROSA CLI, Terraform, oc, gh)
3. Sandbox detection (auto-detects OpenTLC environments)
4. ROSA login and prerequisites (account roles, OIDC config, operator roles)
5. Cluster provisioning (ROSA HCP with 2 workers)
6. cert-manager installation
7. Tekton Pipelines installation (OpenShift Pipelines operator)
8. Operator deployment (CRDs, controller, webhooks)
9. Smoke test
10. GitHub secrets output

**Expected duration**: 30-45 minutes (cluster creation takes 20-30 minutes).

### Option B: Deploy to Existing Cluster

Use this option when a ROSA cluster already exists and you need to deploy or redeploy the operator with all prerequisites.

```bash
oc login -u cluster-admin -p "$PASSWORD" "$SERVER"
./scripts/setup-rosa-e2e.sh --deploy
```

The `--deploy` flag installs cert-manager, Tekton Pipelines, CRDs, the operator, and runs the smoke test.

### Option C: Audit Cluster Readiness

Use this option to verify that all components are installed before you trigger the workflow.

```bash
./scripts/setup-rosa-e2e.sh --audit
```

The audit checks:
1. Cluster connection
2. cert-manager deployment status
3. Tekton Pipelines CRDs and controller
4. S2I BuildConfig API availability
5. pipelines-scc SecurityContextConstraint
6. Operator deployment health
7. Webhook CA bundle injection
8. GitHub secrets (requires `gh` CLI)

**Expected output** (fully ready cluster):
```
═══════════════════════════════════════════════════════════════
  Audit Summary
═══════════════════════════════════════════════════════════════
  Passed: 13
  Failed: 0
  Warnings: 0

  Tier Readiness:
  Tier 1 (Simple):       READY
  Tier 2 (Intermediate): READY
  Tier 3 (Complex):      READY
  Tier 4 (S2I):          READY
  Tier 5 (Volumes):      READY
═══════════════════════════════════════════════════════════════
```

---

## Set GitHub Secrets

After the cluster is ready, set the CI secrets.

### Automatic (requires gh CLI with secrets scope)

```bash
./scripts/setup-rosa-e2e.sh --set-secrets
```

### Manual

1. Get the cluster API URL:
   ```bash
   oc whoami --show-server
   ```

2. Get an authentication token:
   ```bash
   oc whoami -t
   ```

3. Set the secrets in the GitHub UI at:
   `https://github.com/tosin2013/jupyter-notebook-validator-operator/settings/secrets/actions`

---

## Trigger E2E Workflow

### From the GitHub UI

1. Go to **Actions** > **OpenShift E2E Tests**
2. Click **Run workflow**
3. Select the branch
4. Leave `provision_cluster` unchecked (you already have a cluster)
5. Click **Run workflow**

### From the command line

```bash
gh workflow run e2e-openshift.yaml \
  --ref main \
  -f provision_cluster=false \
  -f destroy_cluster=false
```

---

## Verification

### Post-Deployment Checks

#### Operator Health

```bash
oc get deployment notebook-validator-controller-manager \
  -n jupyter-notebook-validator-operator
```

Pass criteria: `READY` column shows `1/1`.

#### Webhook CA Bundle

```bash
oc get mutatingwebhookconfiguration \
  notebook-validator-mutating-webhook-configuration \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | head -c 20
```

Pass criteria: Returns a non-empty base64 string (not `Cg==`).

#### Tekton Controller

```bash
oc get pods -n openshift-pipelines -l app=tekton-pipelines-controller
```

Pass criteria: Pod status is `Running`.

#### Create a Test Validation Job

```bash
oc apply -f config/samples/mlops_v1alpha1_notebookvalidationjob.yaml -n e2e-tests
oc get notebookvalidationjobs -n e2e-tests -w
```

Pass criteria: Job reaches `Succeeded` phase.

---

## Rollback Procedure

### When to Roll Back

Roll back if:
- The cluster failed to provision
- The operator deployment is stuck
- The workflow fails repeatedly

### Destroy Cluster

```bash
./scripts/setup-rosa-e2e.sh --destroy
```

This command deletes the ROSA cluster and cleans up the VPC. IAM roles are preserved for reuse.

### Manual Cleanup

If the script fails to destroy resources:

```bash
cd test/e2e/terraform
terraform destroy -auto-approve

rosa delete cluster --cluster jnvo-e2e --yes --watch
rosa delete oidc-config --oidc-config-id "$OIDC_CONFIG_ID" --yes
rosa delete operator-roles --prefix jnvo-e2e --yes
```

---

## Troubleshooting

### Issue 1: OCM Token Expired

**Symptoms**: `rosa` or `ocm` commands return `401 Unauthorized`.

**Solution**:
```bash
ocm login --use-device-code
```

Follow the browser prompt to re-authenticate. OCM tokens expire after a short period in sandbox environments.

### Issue 2: ROSA Billing Not Enabled

**Symptoms**: Cluster creation fails with "No valid billing account associated".

**Solution**: Enable ROSA at <https://console.aws.amazon.com/rosa/home#/get-started>. Click **Enable ROSA** and wait for confirmation.

### Issue 3: Operator Role Trust Policy Mismatch

**Symptoms**: Cluster creation fails with "Operator role does not have trusted relationship to OIDC issuer URL".

**Diagnosis**:
```bash
rosa list oidc-config
rosa list operator-roles --prefix jnvo-e2e
```

**Solution**: Delete the stale operator roles and recreate them:
```bash
rosa delete operator-roles --prefix jnvo-e2e --yes
rosa create operator-roles --prefix jnvo-e2e \
  --oidc-config-id "$OIDC_CONFIG_ID" \
  --installer-role-arn "$INSTALLER_ROLE_ARN" \
  --hosted-cp --yes --mode auto
```

### Issue 4: Support Role Trust Policy Missing Principal

**Symptoms**: Cluster creation fails with a trust policy error for the support role.

**Diagnosis**:
```bash
aws iam get-role --role-name ManagedOpenShift-HCP-ROSA-Support-Role \
  --query 'Role.AssumeRolePolicyDocument'
```

**Solution**: Add the missing Red Hat support principal. The `setup-rosa-e2e.sh` script handles this automatically in the `rosa_prerequisites` function.

### Issue 5: Quay.io Read-Only Mode

**Symptoms**: `docker push` fails with a read-only or maintenance message.

**Solution**: Wait for Quay.io maintenance to complete. Use an existing image as a workaround:
```bash
export OPERATOR_IMG=quay.io/takinosh/jupyter-notebook-validator-operator:latest
./scripts/setup-rosa-e2e.sh --deploy
```

### Issue 6: cert-manager Pods Not Ready

**Symptoms**: `kubectl get pods -n cert-manager` shows pods in `Pending` or `CrashLoopBackOff`.

**Diagnosis**:
```bash
kubectl describe pods -n cert-manager
kubectl get events -n cert-manager --sort-by='.lastTimestamp'
```

**Solution**: Delete and reinstall cert-manager:
```bash
kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.2/cert-manager.yaml
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.2/cert-manager.yaml
```

### Issue 7: Admin Credentials Not Propagating

**Symptoms**: `oc login` fails with `Unauthorized` immediately after cluster creation.

**Solution**: Wait 2-3 minutes for the admin user to propagate. The setup script retries `oc login` automatically with a 30-second interval.

### Issue 8: Tekton CRDs Not Available After Install

**Symptoms**: `oc api-resources | grep tekton` returns nothing after the OpenShift Pipelines subscription was created.

**Diagnosis**:
```bash
oc get subscription openshift-pipelines-operator -n openshift-operators -o yaml
oc get csv -n openshift-operators | grep pipelines
```

**Solution**: Wait up to 5 minutes for the operator to install. Check the install plan:
```bash
oc get installplan -n openshift-operators
```

If the install plan is in `RequiresApproval` state, approve it:
```bash
oc patch installplan $(oc get installplan -n openshift-operators -o name | head -1) \
  -n openshift-operators --type merge -p '{"spec":{"approved":true}}'
```

---

## Escalation Path

| Severity | First Contact | Response Time |
|----------|---------------|---------------|
| Cluster provisioning failure | Repository maintainer | 1 business day |
| AWS quota or billing issue | AWS Support | Per support plan |
| OCM or ROSA platform issue | Red Hat Support | Per subscription |

---

## Post-Execution Tasks

### Immediate (Within 5 Minutes)
- [ ] Run `./scripts/setup-rosa-e2e.sh --audit` to confirm readiness
- [ ] Set GitHub secrets if not already set
- [ ] Trigger the E2E workflow

### After E2E Tests Complete
- [ ] Review test results in GitHub Actions
- [ ] Download test artifacts if failures occurred
- [ ] Destroy the cluster to avoid ongoing costs:
  ```bash
  ./scripts/setup-rosa-e2e.sh --destroy
  ```

### Cost Awareness

A ROSA HCP cluster with 2 workers costs approximately $2-5 per hour:
- Worker nodes: 2x m5.xlarge (~$0.384/hr each)
- NAT Gateway: ~$0.045/hr
- ROSA HCP control plane: included in ROSA pricing

Destroy the cluster when you finish testing.

---

## Script Reference

| Command | Description |
|---------|-------------|
| `./scripts/setup-rosa-e2e.sh` | Full setup: tools + cluster + cert-manager + Tekton + operator + smoke test |
| `./scripts/setup-rosa-e2e.sh --tools-only` | Install tools (Go, ROSA CLI, Terraform, oc, gh) |
| `./scripts/setup-rosa-e2e.sh --deploy` | Deploy to existing cluster (cert-manager + Tekton + operator + smoke test) |
| `./scripts/setup-rosa-e2e.sh --audit` | Audit cluster readiness for all E2E tiers |
| `./scripts/setup-rosa-e2e.sh --secrets` | Print GitHub secrets needed for CI |
| `./scripts/setup-rosa-e2e.sh --set-secrets` | Auto-set GitHub repo secrets |
| `./scripts/setup-rosa-e2e.sh --destroy` | Tear down cluster and clean up resources |
| `./scripts/setup-rosa-e2e.sh --status` | Check cluster and operator status |
| `./scripts/setup-rosa-e2e.sh --terraform-validate` | Validate Terraform configs (dry-run) |

---

## Related Documentation

- [TEKTON_BUILD_SETUP.md](TEKTON_BUILD_SETUP.md): Tekton prerequisites and setup for build tasks
- [TESTING_GUIDE.md](../reference/TESTING_GUIDE.md): Test tier descriptions and notebook categories
- [Terraform README](../../test/e2e/terraform/README.md): ROSA HCP Terraform configuration details
- [RELEASE.md](RELEASE.md): Release process and OLM bundle generation

---

## Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-09-14 | tosin2013 | Initial version |

---

**Last Reviewed**: 2026-09-14
**Next Review**: 2026-12-14
**Feedback**: Open an issue at [jupyter-notebook-validator-operator](https://github.com/tosin2013/jupyter-notebook-validator-operator/issues)
