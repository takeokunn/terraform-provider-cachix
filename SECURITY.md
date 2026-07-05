# Security Policy

## Supported Versions

Only the latest released version of the provider receives security fixes.
Please upgrade to the most recent release before reporting an issue.

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Instead, report them privately using GitHub's
[private vulnerability reporting](https://github.com/takeokunn/terraform-provider-cachix/security/advisories/new).

When reporting, include as much of the following as possible:

- A description of the vulnerability and its impact.
- Steps to reproduce or a proof of concept.
- The provider version and Terraform/OpenTofu version affected.

You can expect an initial acknowledgement within a few business days. Once the
issue is confirmed, a fix will be prepared and released, and you will be
credited unless you prefer to remain anonymous.

## Handling of Credentials

This provider authenticates to the Cachix API with a bearer token. The token is:

- Marked `sensitive` in the provider schema so it is redacted from Terraform output.
- Never written to logs; only non-sensitive metadata (such as the API host) is logged.

Always supply the token via the `CACHIX_AUTH_TOKEN` environment variable or a
sensitive variable rather than committing it to version control.
