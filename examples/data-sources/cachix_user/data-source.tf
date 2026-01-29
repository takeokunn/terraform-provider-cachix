# Retrieve the current authenticated user
data "cachix_user" "current" {}

output "username" {
  value = data.cachix_user.current.username
}

output "email" {
  value     = data.cachix_user.current.email
  sensitive = true
}
