# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-07-05

### Added

- `cachix_cache` resource and data source now expose additional read-only
  attributes: `github_username`, `permission` (`Read`/`Write`/`Admin`), and
  `preferred_compression_method` (`XZ`/`ZSTD`).
- `cachix_user` data source now exposes `fullname`, `subscription_plan`,
  `subscription_storage_limit`, and `subscription_storage_usage`.
- Nix flake (`flake.nix`) providing a pinned development shell, a buildable
  provider package (`nix build`), and `nix flake check` (build, unit tests,
  `gofmt`, `golangci-lint`).

### Changed

- CI consolidated to a single `nix flake check` workflow.
- Upgraded all Go dependencies, including `terraform-plugin-framework` to
  v1.19.0, `terraform-plugin-go` to v0.31.0, and `terraform-plugin-testing`
  to v1.16.0.
- Development environment migrated from devenv to a standalone Nix flake.

## [1.0.1] - 2025

### Fixed

- Include the account ID when creating a cache and improve test coverage.

## [1.0.0] - 2025

### Added

- Initial release: `cachix_cache` resource, `cachix_cache` and `cachix_user`
  data sources, provider authentication via `auth_token`/`CACHIX_AUTH_TOKEN`,
  import support, and an HTTP client with retry and exponential backoff.

[1.1.0]: https://github.com/takeokunn/terraform-provider-cachix/releases/tag/v1.1.0
[1.0.1]: https://github.com/takeokunn/terraform-provider-cachix/releases/tag/v1.0.1
[1.0.0]: https://github.com/takeokunn/terraform-provider-cachix/releases/tag/v1.0.0
