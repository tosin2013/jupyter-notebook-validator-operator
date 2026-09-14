# ROSA HCP Cluster for E2E Tests

This Terraform configuration provisions an ephemeral ROSA HCP (Hosted Control Plane)
cluster for running operator E2E tests. It is designed to be invoked from
GitHub Actions via the `e2e-openshift.yaml` workflow.

## Architecture

- Single-AZ VPC with one public and one private subnet
- ROSA HCP cluster with 2 worker nodes (minimum for operator testing)
- Cluster admin user created for `oc login`
- All AWS resources tagged for traceability

## Prerequisites

Before using this configuration, you need:

1. **AWS account** with ROSA enabled
2. **OCM token** from https://console.redhat.com/openshift/token
3. **ROSA account roles** created via `rosa create account-roles`
4. **OIDC config** created via `rosa create oidc-config`
5. **Operator roles** created via `rosa create operator-roles`

## GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS credentials (or use OIDC role) |
| `AWS_SECRET_ACCESS_KEY` | AWS credentials |
| `AWS_ACCOUNT_ID` | AWS account ID |
| `ROSA_TOKEN` | OCM offline token |
| `ROSA_INSTALLER_ROLE_ARN` | Installer role ARN |
| `ROSA_SUPPORT_ROLE_ARN` | Support role ARN |
| `ROSA_WORKER_ROLE_ARN` | Worker role ARN |
| `ROSA_OIDC_CONFIG_ID` | OIDC config ID |
| `ROSA_CREATOR_ARN` | Creator IAM ARN |

## Cost

- Estimated cost: **$2-5 per E2E run** (1-2 hours of ROSA HCP runtime)
- Worker nodes: 2x m5.xlarge (~$0.384/hr each)
- NAT Gateway: ~$0.045/hr
- The workflow always destroys infrastructure after tests complete

## Local Usage

```bash
cd test/e2e/terraform

export TF_VAR_rhcs_token="your-ocm-token"
export TF_VAR_aws_account_id="123456789012"
export TF_VAR_installer_role_arn="arn:aws:iam::..."
export TF_VAR_support_role_arn="arn:aws:iam::..."
export TF_VAR_worker_role_arn="arn:aws:iam::..."
export TF_VAR_oidc_config_id="..."
export TF_VAR_rosa_creator_arn="arn:aws:iam::..."

terraform init
terraform apply -auto-approve

# Get cluster credentials
export OPENSHIFT_SERVER=$(terraform output -raw cluster_api_url)
export OPENSHIFT_PASSWORD=$(terraform output -raw admin_password)
oc login -u cluster-admin -p "$OPENSHIFT_PASSWORD" "$OPENSHIFT_SERVER"

# Run tests...

terraform destroy -auto-approve
```
