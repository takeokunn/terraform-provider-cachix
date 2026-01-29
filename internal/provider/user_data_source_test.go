// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Unit Tests

func TestUserDataSource_Metadata(t *testing.T) {
	d := NewUserDataSource()

	req := datasource.MetadataRequest{ProviderTypeName: "cachix"}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	if resp.TypeName != "cachix_user" {
		t.Errorf("expected TypeName 'cachix_user', got '%s'", resp.TypeName)
	}
}

func TestUserDataSource_Schema(t *testing.T) {
	d := NewUserDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	attrs := []string{"id", "username", "email"}
	for _, attr := range attrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected '%s' attribute in schema", attr)
		}
	}

	// Verify email is marked as sensitive
	emailAttr := resp.Schema.Attributes["email"]
	if stringAttr, ok := emailAttr.(schema.StringAttribute); ok {
		if !stringAttr.Sensitive {
			t.Error("expected 'email' attribute to be marked as sensitive")
		}
	} else {
		t.Error("expected 'email' attribute to be a StringAttribute")
	}
}

// Acceptance Tests

func TestAccUserDataSource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cachix_user.current", "id"),
					resource.TestCheckResourceAttrSet("data.cachix_user.current", "username"),
				),
			},
		},
	})
}

func testAccUserDataSourceConfig() string {
	return `
data "cachix_user" "current" {}
`
}
