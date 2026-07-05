# Terraform Provider for Cachix

[![CI](https://github.com/takeokunn/terraform-provider-cachix/actions/workflows/ci.yml/badge.svg)](https://github.com/takeokunn/terraform-provider-cachix/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/takeokunn/terraform-provider-cachix)](https://goreportcard.com/report/github.com/takeokunn/terraform-provider-cachix)
[![Terraform Registry](https://img.shields.io/badge/registry-takeokunn%2Fcachix-blue?logo=terraform)](https://registry.terraform.io/providers/takeokunn/cachix/latest)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg)](LICENSE)

Manage [Cachix](https://cachix.org) binary caches as infrastructure-as-code with Terraform or OpenTofu.

## Features

- **`cachix_cache` resource** — create, read, import, and delete binary caches.
- **`cachix_cache` data source** — reference existing caches (managed elsewhere or public).
- **`cachix_user` data source** — look up the authenticated account, including subscription and storage usage.
- Automatic signing-key generation, exposed as `public_signing_keys` for `nix.conf`.
- Robust HTTP client with retries and exponential backoff on `429`/`5xx` responses.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 or [OpenTofu](https://opentofu.org) >= 1.6
- A [Cachix](https://cachix.org) account and API token

## Usage

```hcl
terraform {
  required_providers {
    cachix = {
      source  = "takeokunn/cachix"
      version = "~> 1.0"
    }
  }
}

provider "cachix" {
  # auth_token = var.cachix_auth_token   # or set CACHIX_AUTH_TOKEN
}

# Create and manage a cache.
resource "cachix_cache" "example" {
  name      = "my-cache"
  is_public = true
}

# Reference an existing public cache.
data "cachix_cache" "nixpkgs" {
  name = "nixpkgs"
}

# Look up the authenticated user.
data "cachix_user" "current" {}

output "trusted_public_keys" {
  value = cachix_cache.example.public_signing_keys
}
```

### Authentication

The provider reads the API token from, in order of precedence:

1. The `auth_token` argument in the `provider` block.
2. The `CACHIX_AUTH_TOKEN` environment variable.

```bash
export CACHIX_AUTH_TOKEN="your-token-here"
```

Full documentation, including every attribute, is published on the
[Terraform Registry](https://registry.terraform.io/providers/takeokunn/cachix/latest/docs).

## Development

This repository ships a [Nix flake](https://nixos.wiki/wiki/Flakes) that provides
the entire toolchain (Go, Terraform, golangci-lint, goreleaser, terraform-docs).

```bash
nix develop            # Enter a dev shell with every tool pinned
nix build              # Build the provider binary
nix flake check        # Run build, unit tests, gofmt, and golangci-lint
```

If you use [direnv](https://direnv.net), running `direnv allow` loads the dev
shell automatically via the checked-in `.envrc` (`use flake`).

Common tasks are also wrapped in the `Makefile`:

```bash
make build     # go build ./...
make test      # unit tests
make testacc   # acceptance tests (requires CACHIX_AUTH_TOKEN)
make lint      # golangci-lint
make generate  # regenerate registry documentation
```

CI runs a single `nix flake check`, so a green `nix flake check` locally
mirrors exactly what runs on pull requests.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full contributor guide.

## License

[MPL-2.0](LICENSE)
