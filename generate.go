// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package main

// Documentation generation. Run with `go generate ./...`. tfplugindocs is
// pinned via the `tool` directive in go.mod and invoked through `go tool`
// (Go 1.24+), so no separate tools module is required.

//go:generate terraform fmt -recursive ./examples/
//go:generate go tool github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir . --provider-name cachix
