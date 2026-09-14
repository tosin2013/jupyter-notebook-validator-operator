# Outputs for E2E workflow consumption

output "cluster_api_url" {
  description = "OpenShift API server URL"
  value       = rhcs_cluster_rosa_hcp.cluster.api_url
}

output "cluster_console_url" {
  description = "OpenShift console URL"
  value       = rhcs_cluster_rosa_hcp.cluster.console_url
}

output "cluster_id" {
  description = "ROSA cluster ID"
  value       = rhcs_cluster_rosa_hcp.cluster.id
}

output "admin_username" {
  description = "Cluster admin username"
  value       = rhcs_cluster_rosa_hcp_admin.admin.username
}

output "admin_password" {
  description = "Cluster admin password"
  value       = rhcs_cluster_rosa_hcp_admin.admin.password
  sensitive   = true
}

output "vpc_id" {
  description = "VPC ID (for cleanup verification)"
  value       = module.vpc.vpc_id
}
