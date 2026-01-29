# Reference an existing cache not managed by Terraform
data "cachix_cache" "nixpkgs" {
  name = "nixpkgs"
}

# Use the signing keys in your nix configuration
output "nixpkgs_substituter" {
  value = data.cachix_cache.nixpkgs.uri
}

output "nixpkgs_public_keys" {
  value = data.cachix_cache.nixpkgs.public_signing_keys
}
