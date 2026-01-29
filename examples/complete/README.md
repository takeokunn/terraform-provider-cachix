# Complete Cachix Provider Example

This example demonstrates a complete setup for managing Cachix binary caches with Terraform.

## Features

- Provider configuration with variables
- Creating multiple caches for different environments (dev, staging, prod)
- Referencing existing public caches (nix-community, devenv)
- Generating nix.conf configuration
- Outputting values for CI/CD integration

## Prerequisites

1. A Cachix account with API token
2. Terraform >= 1.0

## Usage

### 1. Set up authentication

```bash
export CACHIX_AUTH_TOKEN="your-token-here"
```

Or create a `terraform.tfvars` file:

```hcl
cachix_token = "your-token-here"
project_name = "my-awesome-project"
environments = ["dev", "staging", "prod"]
```

### 2. Initialize and apply

```bash
terraform init
terraform plan
terraform apply
```

### 3. Use the outputs

Get the nix.conf configuration:

```bash
terraform output -raw nix_extra_config >> ~/.config/nix/nix.conf
```

Get GitHub Actions environment:

```bash
terraform output -json github_actions_env
```

## Outputs

| Output | Description |
|--------|-------------|
| `authenticated_user` | Current Cachix username |
| `main_cache` | Main project cache details |
| `environment_caches` | Map of environment cache URIs |
| `nix_substituters` | Substituters line for nix.conf |
| `nix_trusted_public_keys` | Trusted public keys for nix.conf |
| `nix_extra_config` | Complete nix.conf configuration block |
| `github_actions_env` | Environment variables for GitHub Actions |

## Customization

Modify `environments` variable to create different sets of caches:

```hcl
environments = ["feature-x", "feature-y"]
```

Set `is_public` logic in the resource based on your needs.
