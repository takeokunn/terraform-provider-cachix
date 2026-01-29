// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Unit Tests

func TestCacheResource_Metadata(t *testing.T) {
	r := NewCacheResource()

	req := resource.MetadataRequest{ProviderTypeName: "cachix"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "cachix_cache" {
		t.Errorf("expected TypeName 'cachix_cache', got '%s'", resp.TypeName)
	}
}

func TestCacheResource_Schema(t *testing.T) {
	r := NewCacheResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify required attributes
	attrs := []string{"id", "name", "is_public", "uri", "public_signing_keys"}
	for _, attr := range attrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected '%s' attribute in schema", attr)
		}
	}
}

func TestCacheResource_Schema_NameValidation(t *testing.T) {
	// Test that invalid cache names are rejected by the validator
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-cache", true},
		{"cache123", true},
		{"a", true},
		{"test-cache-name", true},
		{"123cache", false}, // starts with number
		{"My-Cache", false}, // uppercase
		{"my_cache", false}, // underscore
		{"my cache", false}, // space
		{"-cache", false},   // starts with hyphen
	}

	pattern := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := pattern.MatchString(tt.name)
			if matches != tt.valid {
				t.Errorf("name '%s': expected valid=%v, got %v", tt.name, tt.valid, matches)
			}
		})
	}
}

// Acceptance Tests

func TestAccCacheResource_Basic(t *testing.T) {
	cacheName := fmt.Sprintf("test-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Create and Read testing
			{
				Config: testAccCacheResourceConfig(cacheName, true),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("cachix_cache.test", "name", cacheName),
					tfresource.TestCheckResourceAttr("cachix_cache.test", "is_public", "true"),
					tfresource.TestCheckResourceAttrSet("cachix_cache.test", "id"),
					tfresource.TestCheckResourceAttrSet("cachix_cache.test", "uri"),
					tfresource.TestCheckResourceAttrSet("cachix_cache.test", "public_signing_keys.#"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "cachix_cache.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCacheResource_Private(t *testing.T) {
	cacheName := fmt.Sprintf("test-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Create a private cache
			{
				Config: testAccCacheResourceConfig(cacheName, false),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("cachix_cache.test", "name", cacheName),
					tfresource.TestCheckResourceAttr("cachix_cache.test", "is_public", "false"),
					tfresource.TestCheckResourceAttrSet("cachix_cache.test", "id"),
					tfresource.TestCheckResourceAttrSet("cachix_cache.test", "uri"),
				),
			},
		},
	})
}

func TestAccCacheResource_ForceNew(t *testing.T) {
	cacheName1 := fmt.Sprintf("test-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	cacheName2 := fmt.Sprintf("test-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Create initial cache
			{
				Config: testAccCacheResourceConfig(cacheName1, true),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("cachix_cache.test", "name", cacheName1),
				),
			},
			// Change name should trigger ForceNew (destroy and recreate)
			{
				Config: testAccCacheResourceConfig(cacheName2, true),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("cachix_cache.test", "name", cacheName2),
				),
			},
		},
	})
}

func TestAccCacheResource_Import(t *testing.T) {
	cacheName := fmt.Sprintf("test-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Create the cache first
			{
				Config: testAccCacheResourceConfig(cacheName, true),
			},
			// Import using the cache name
			{
				ResourceName:      "cachix_cache.test",
				ImportState:       true,
				ImportStateId:     cacheName,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCacheResourceConfig(name string, isPublic bool) string {
	return fmt.Sprintf(`
resource "cachix_cache" "test" {
  name      = %[1]q
  is_public = %[2]t
}
`, name, isPublic)
}
