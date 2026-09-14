# Variables for ROSA HCP E2E cluster provisioning

variable "cluster_name" {
  description = "Name of the ROSA HCP cluster"
  type        = string
  default     = "jnvo-e2e"
}

variable "aws_region" {
  description = "AWS region for the cluster"
  type        = string
  default     = "us-east-2"
}

variable "aws_account_id" {
  description = "AWS account ID"
  type        = string
}

variable "ocp_version" {
  description = "OpenShift version to deploy"
  type        = string
  default     = "4.20.0"
}

variable "worker_count" {
  description = "Number of worker nodes (minimum for E2E)"
  type        = number
  default     = 2
}

variable "rhcs_token" {
  description = "OCM offline token for ROSA API"
  type        = string
  sensitive   = true
}

variable "ocm_url" {
  description = "OCM API URL"
  type        = string
  default     = "https://api.openshift.com"
}

variable "installer_role_arn" {
  description = "ARN of the ROSA installer IAM role"
  type        = string
}

variable "support_role_arn" {
  description = "ARN of the ROSA support IAM role"
  type        = string
}

variable "worker_role_arn" {
  description = "ARN of the ROSA worker IAM role"
  type        = string
}

variable "oidc_config_id" {
  description = "Pre-created OIDC config ID for HCP"
  type        = string
}

variable "rosa_creator_arn" {
  description = "ARN of the IAM entity creating the cluster"
  type        = string
}

variable "github_run_id" {
  description = "GitHub Actions run ID for resource tagging"
  type        = string
  default     = "local"
}
