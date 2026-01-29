# Terraform Provider for Cachix

A Terraform provider for managing [Cachix](https://cachix.org) binary caches as infrastructure-as-code.

## Requirements

- [Go](https://golang.org/doc/install) 1.23+ (for building from source)
- [Terraform](https://www.terraform.io/downloads.html) 1.0+
- A [Cachix](https://cachix.org) account with an API token

## Installation

### From Terraform Registry

```hcl
terraform {
  required_providers {
    cachix = {
      source  = "takeokunn/cachix"
      version = "~> 0.1"
    }
  }
}

provider "cachix" {}
```

### Local Development

Clone the repository and build the provider:

```bash
git clone https://github.com/takeokunn/terraform-provider-cachix.git
cd terraform-provider-cachix
make install
```

Configure Terraform to use the local provider by creating a `~/.terraformrc` file:

```hcl
provider_installation {
  dev_overrides {
    "takeokunn/cachix" = "/path/to/go/bin"
  }
  direct {}
}
```

## Quick Start

1. Set your Cachix authentication token:

```bash
export CACHIX_AUTH_TOKEN="your-token-here"
```

2. Create a Terraform configuration:

```hcl
terraform {
  required_providers {
    cachix = {
      source  = "takeokunn/cachix"
      version = "~> 0.1"
    }
  }
}

provider "cachix" {}

resource "cachix_cache" "my_project" {
  name      = "my-project"
  is_public = true
}

output "cache_uri" {
  value = cachix_cache.my_project.uri
}

output "nix_conf" {
  value = <<-EOT
    extra-substituters = ${cachix_cache.my_project.uri}
    extra-trusted-public-keys = ${join(" ", cachix_cache.my_project.public_signing_keys)}
  EOT
}
```

3. Initialize and apply:

```bash
terraform init
terraform apply
```

## Resources

### cachix_cache

Manages a Cachix binary cache.

#### Example Usage

```hcl
resource "cachix_cache" "example" {
  name      = "my-cache"
  is_public = true
}
```

#### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | The name of the cache. Immutable after creation. |
| `is_public` | bool | No | Whether the cache is publicly readable. Default: `true` |

#### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | Cache identifier |
| `uri` | string | Full cache URI (e.g., `https://my-cache.cachix.org`) |
| `public_signing_keys` | list(string) | Public signing keys for nix.conf configuration |
| `created_at` | string | Creation timestamp |

#### Import

Existing caches can be imported using the cache name:

```bash
terraform import cachix_cache.example my-cache
```

## Data Sources

### cachix_cache

Read-only access to cache information.

#### Example Usage

```hcl
data "cachix_cache" "nixpkgs" {
  name = "nixpkgs"
}

output "nixpkgs_uri" {
  value = data.cachix_cache.nixpkgs.uri
}
```

#### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | The name of the cache to look up |

#### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | Cache identifier |
| `name` | string | Cache name |
| `uri` | string | Full cache URI |
| `is_public` | bool | Whether the cache is publicly readable |
| `public_signing_keys` | list(string) | Public signing keys for nix.conf |

### cachix_user

Retrieves information about the currently authenticated Cachix user.

#### Example Usage

```hcl
data "cachix_user" "current" {}

output "username" {
  value = data.cachix_user.current.username
}
```

#### Argument Reference

This data source has no required arguments. It retrieves the user associated with the configured authentication token.

#### Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | The user identifier (same as username) |
| `username` | String | The username of the authenticated user |
| `email` | String | The email address of the authenticated user (sensitive) |

## Authentication

The provider requires a Cachix authentication token. You can configure it in two ways:

### Environment Variable (Recommended)

```bash
export CACHIX_AUTH_TOKEN="your-token-here"
```

### Provider Configuration

```hcl
provider "cachix" {
  auth_token = var.cachix_token
}
```

### Optional: Custom API Host

For self-hosted Cachix instances:

```hcl
provider "cachix" {
  auth_token = var.cachix_token
  api_host   = "https://your-cachix-instance.example.com/api/v1"
}
```

## Development

### Building

```bash
make build
```

### Testing

Run unit tests:

```bash
make test
```

Run acceptance tests (requires `CACHIX_AUTH_TOKEN`):

```bash
make testacc
```

### Generating Documentation

```bash
make docs
```

### Release

Releases are automated via GitHub Actions and GoReleaser. To create a new release:

1. Tag the commit: `git tag v0.1.0`
2. Push the tag: `git push origin v0.1.0`

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.
