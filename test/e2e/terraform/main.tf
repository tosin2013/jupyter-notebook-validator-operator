# ROSA HCP cluster provisioning for E2E tests
#
# This Terraform configuration provisions a minimal ROSA HCP cluster
# for running operator E2E tests. It is designed to be used from
# GitHub Actions via the e2e-openshift.yaml workflow.
#
# Usage:
#   terraform init
#   terraform apply -auto-approve
#   # ... run tests ...
#   terraform destroy -auto-approve
#
# Required environment variables:
#   RHCS_TOKEN          - OCM offline token (from console.redhat.com/openshift/token)
#   AWS_REGION          - AWS region (default: us-east-2)
#   AWS_ACCESS_KEY_ID   - AWS credentials (or use OIDC role assumption)
#   AWS_SECRET_ACCESS_KEY

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    rhcs = {
      source  = "terraform-redhat/rhcs"
      version = ">= 1.6.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    null = {
      source  = "hashicorp/null"
      version = ">= 3.0"
    }
  }
}

provider "rhcs" {
  token = var.rhcs_token
  url   = var.ocm_url
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      "app-code"      = "jupyter-notebook-validator-operator"
      "service-phase"  = "e2e-test"
      "owner"          = "tosin2013"
      "repo"           = "tosin2013/jupyter-notebook-validator-operator"
      "managed-by"     = "terraform"
      "github-run-id"  = var.github_run_id
    }
  }
}

# --------------------------------------------------------------------------
# VPC (single-AZ for cost optimization)
# --------------------------------------------------------------------------

data "aws_availability_zones" "available" {
  state = "available"
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.cluster_name}-vpc"
  cidr = "10.0.0.0/16"

  azs             = [data.aws_availability_zones.available.names[0]]
  private_subnets = ["10.0.1.0/24"]
  public_subnets  = ["10.0.101.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true
  enable_dns_support   = true

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = "1"
  }

  public_subnet_tags = {
    "kubernetes.io/role/elb" = "1"
  }
}

# --------------------------------------------------------------------------
# ROSA HCP Cluster
# --------------------------------------------------------------------------

resource "rhcs_cluster_rosa_hcp" "cluster" {
  name                = var.cluster_name
  cloud_region        = var.aws_region
  aws_account_id      = var.aws_account_id
  aws_subnet_ids      = concat(module.vpc.private_subnets, module.vpc.public_subnets)
  availability_zones  = [data.aws_availability_zones.available.names[0]]
  replicas            = var.worker_count
  version             = var.ocp_version

  sts = {
    role_arn         = var.installer_role_arn
    support_role_arn = var.support_role_arn
    instance_iam_roles = {
      worker_role_arn = var.worker_role_arn
    }
    operator_role_prefix = var.cluster_name
    oidc_config_id       = var.oidc_config_id
  }

  wait_for_create_complete            = true
  wait_for_std_compute_nodes_complete = true

  properties = {
    rosa_creator_arn = var.rosa_creator_arn
  }
}

# --------------------------------------------------------------------------
# Cluster credentials (admin user for E2E tests)
# --------------------------------------------------------------------------

resource "rhcs_cluster_rosa_hcp_admin" "admin" {
  cluster_id = rhcs_cluster_rosa_hcp.cluster.id
}
