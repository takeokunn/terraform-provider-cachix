# Retrieve the current authenticated user
data "cachix_user" "current" {}

output "username" {
  value = data.cachix_user.current.username
}

output "email" {
  value     = data.cachix_user.current.email
  sensitive = true
}

output "subscription_plan" {
  value = data.cachix_user.current.subscription_plan
}

# Storage limit and usage are reported in bytes.
output "storage_usage_bytes" {
  value = data.cachix_user.current.subscription_storage_usage
}
