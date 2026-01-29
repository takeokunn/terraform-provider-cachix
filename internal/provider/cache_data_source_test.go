// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Unit Tests

func TestCacheDataSource_Metadata(t *testing.T) {
	d := NewCacheDataSource()

	req := datasource.MetadataRequest{ProviderTypeName: "cachix"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	if resp.TypeName != "cachix_cache" {
		t.Errorf("expected TypeName 'cachix_cache', got '%s'", resp.TypeName)
	}
}

func TestCacheDataSource_Schema(t *testing.T) {
	d := NewCacheDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	attrs := []string{"id", "name", "uri", "is_public", "public_signing_keys"}
	for _, attr := range attrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected '%s' attribute in schema", attr)
		}
	}
}

// Acceptance Tests

func TestAccCacheDataSource_Basic(t *testing.T) {
	// Use "nixpkgs" as it's a well-known public cache that always exists
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCacheDataSourceConfig("nixpkgs"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cachix_cache.test", "name", "nixpkgs"),
					resource.TestCheckResourceAttr("data.cachix_cache.test", "id", "nixpkgs"),
					resource.TestCheckResourceAttr("data.cachix_cache.test", "is_public", "true"),
					resource.TestCheckResourceAttrSet("data.cachix_cache.test", "uri"),
					resource.TestCheckResourceAttrSet("data.cachix_cache.test", "public_signing_keys.#"),
				),
			},
		},
	})
}

func TestAccCacheDataSource_WithResource(t *testing.T) {
	cacheName := fmt.Sprintf("test-acc-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a cache using resource, then read it using data source
			{
				Config: testAccCacheDataSourceWithResourceConfig(cacheName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check resource
					resource.TestCheckResourceAttr("cachix_cache.test", "name", cacheName),
					resource.TestCheckResourceAttr("cachix_cache.test", "is_public", "true"),
					// Check data source reads the same values
					resource.TestCheckResourceAttr("data.cachix_cache.test", "name", cacheName),
					resource.TestCheckResourceAttr("data.cachix_cache.test", "is_public", "true"),
					// Compare URI between resource and data source
					resource.TestCheckResourceAttrPair(
						"cachix_cache.test", "uri",
						"data.cachix_cache.test", "uri",
					),
					resource.TestCheckResourceAttrPair(
						"cachix_cache.test", "public_signing_keys.#",
						"data.cachix_cache.test", "public_signing_keys.#",
					),
				),
			},
		},
	})
}

func testAccCacheDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "cachix_cache" "test" {
  name = %[1]q
}
`, name)
}

func testAccCacheDataSourceWithResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "cachix_cache" "test" {
  name      = %[1]q
  is_public = true
}

data "cachix_cache" "test" {
  name = cachix_cache.test.name
}
`, name)
}
