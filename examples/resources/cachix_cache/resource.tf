# Create a public binary cache
resource "cachix_cache" "my_project" {
  name      = "my-project"
  is_public = true
}

# Create a private binary cache
resource "cachix_cache" "private_cache" {
  name      = "my-private-cache"
  is_public = false
}

# Output for nix.conf configuration
output "nix_conf" {
  value = <<-EOT
    extra-substituters = ${cachix_cache.my_project.uri}
    extra-trusted-public-keys = ${join(" ", cachix_cache.my_project.public_signing_keys)}
  EOT
}
