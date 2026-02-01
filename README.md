# Terraform Provider for Cachix

[![Tests](https://github.com/takeokunn/terraform-provider-cachix/actions/workflows/test.yml/badge.svg)](https://github.com/takeokunn/terraform-provider-cachix/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/takeokunn/terraform-provider-cachix)](https://goreportcard.com/report/github.com/takeokunn/terraform-provider-cachix)

A Terraform provider for managing [Cachix](https://cachix.org) binary caches.

## Usage

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

resource "cachix_cache" "example" {
  name      = "my-cache"
  is_public = true
}
```

Set `CACHIX_AUTH_TOKEN` environment variable for authentication.

Full documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/takeokunn/cachix/latest/docs).

## Development

```bash
make build    # Build
make test     # Unit tests
make testacc  # Acceptance tests (requires CACHIX_AUTH_TOKEN)
make generate # Generate documentation
```

## License

[MPL-2.0](LICENSE)
