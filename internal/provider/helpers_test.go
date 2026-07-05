// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestOptionalString(t *testing.T) {
	t.Run("empty string maps to null", func(t *testing.T) {
		v := optionalString("")
		if !v.IsNull() {
			t.Errorf("expected null value for empty string, got %v", v)
		}
	})

	t.Run("non-empty string maps to value", func(t *testing.T) {
		v := optionalString("hello")
		if v.IsNull() {
			t.Fatal("expected non-null value")
		}
		if v.ValueString() != "hello" {
			t.Errorf("expected 'hello', got %q", v.ValueString())
		}
	})
}

func TestMapCacheToState(t *testing.T) {
	t.Run("maps all fields", func(t *testing.T) {
		cache := &Cache{
			Name:                       "my-cache",
			URI:                        "https://my-cache.cachix.org",
			IsPublic:                   true,
			PublicSigningKeys:          []string{"my-cache.cachix.org-1:abc="},
			GithubUsername:             "takeokunn",
			Permission:                 "Admin",
			PreferredCompressionMethod: "ZSTD",
		}

		var diags diag.Diagnostics
		s := mapCacheToState(context.Background(), cache, &diags)

		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if s.ID.ValueString() != "my-cache" {
			t.Errorf("expected ID 'my-cache', got %q", s.ID.ValueString())
		}
		if s.Name.ValueString() != "my-cache" {
			t.Errorf("expected Name 'my-cache', got %q", s.Name.ValueString())
		}
		if !s.IsPublic.ValueBool() {
			t.Error("expected IsPublic true")
		}
		if s.URI.ValueString() != "https://my-cache.cachix.org" {
			t.Errorf("unexpected URI %q", s.URI.ValueString())
		}
		if s.GithubUsername.ValueString() != "takeokunn" {
			t.Errorf("expected GithubUsername 'takeokunn', got %q", s.GithubUsername.ValueString())
		}
		if s.Permission.ValueString() != "Admin" {
			t.Errorf("expected Permission 'Admin', got %q", s.Permission.ValueString())
		}
		if s.PreferredCompressionMethod.ValueString() != "ZSTD" {
			t.Errorf("expected PreferredCompressionMethod 'ZSTD', got %q", s.PreferredCompressionMethod.ValueString())
		}
		if s.PublicSigningKeys.IsNull() {
			t.Error("expected non-null PublicSigningKeys")
		}
	})

	t.Run("optional fields absent map to null", func(t *testing.T) {
		cache := &Cache{
			Name:              "bare-cache",
			URI:               "https://bare-cache.cachix.org",
			IsPublic:          false,
			PublicSigningKeys: []string{},
		}

		var diags diag.Diagnostics
		s := mapCacheToState(context.Background(), cache, &diags)

		if !s.GithubUsername.IsNull() {
			t.Error("expected GithubUsername null when absent")
		}
		if !s.Permission.IsNull() {
			t.Error("expected Permission null when absent")
		}
		if !s.PreferredCompressionMethod.IsNull() {
			t.Error("expected PreferredCompressionMethod null when absent")
		}
	})
}
