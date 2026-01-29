# Contributing to Terraform Provider Cachix

Thank you for your interest in contributing to the Terraform Provider for Cachix. This document provides guidelines and instructions for contributing.

## Requirements

- [Go](https://golang.org/doc/install) 1.24+
- [Terraform](https://www.terraform.io/downloads.html) 1.0+
- A [Cachix](https://cachix.org) account with an API token (for acceptance tests)

## Building the Provider

Clone the repository:

```bash
git clone https://github.com/takeokunn/terraform-provider-cachix.git
cd terraform-provider-cachix
```

Build the provider:

```bash
go build -v ./...
```

Install the provider locally:

```bash
go install .
```

## Testing

### Unit Tests

Run unit tests with:

```bash
go test -v -cover ./...
```

### Acceptance Tests

Acceptance tests create real resources against the Cachix API. You must set the `CACHIX_AUTH_TOKEN` environment variable:

```bash
export CACHIX_AUTH_TOKEN="your-token-here"
TF_ACC=1 go test -v ./internal/provider/... -timeout 120m
```

**Note:** Acceptance tests may incur costs or create real resources. Use a dedicated test account when possible.

### Linting

This project uses [golangci-lint](https://golangci-lint.run/) for linting:

```bash
golangci-lint run ./...
```

## Documentation

Provider documentation is generated using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs).

### Structure

- `templates/` - Documentation templates
- `examples/` - Example configurations used in documentation
  - `examples/provider/` - Provider configuration examples
  - `examples/resources/` - Resource examples
  - `examples/data-sources/` - Data source examples

### Generating Documentation

```bash
go generate ./...
```

Or using tfplugindocs directly:

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
tfplugindocs generate
```

## Submitting Changes

### Pull Request Process

1. Fork the repository and create a feature branch from `main`
2. Make your changes with appropriate tests
3. Ensure all tests pass and linting is clean
4. Update documentation if adding or modifying resources/data sources
5. Submit a pull request to the `main` branch

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature or resource
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring without behavior changes
- `chore`: Maintenance tasks

Examples:
```
feat: add cachix_agent resource
fix: handle nil response in cache data source
docs: update authentication examples
```

### Code Style

- Follow standard Go conventions and [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Add tests for new functionality
- Keep functions focused and well-documented

## License

By contributing, you agree that your contributions will be licensed under the [Mozilla Public License 2.0](LICENSE).
