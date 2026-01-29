// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure CachixProvider satisfies various provider interfaces.
var _ provider.Provider = &CachixProvider{}

// CachixProvider defines the provider implementation.
type CachixProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// CachixProviderModel describes the provider data model.
type CachixProviderModel struct {
	AuthToken types.String `tfsdk:"auth_token"`
	APIHost   types.String `tfsdk:"api_host"`
}

// Metadata returns the provider type name.
func (p *CachixProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cachix"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *CachixProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Cachix provider allows you to manage Cachix binary caches as infrastructure-as-code.",
		MarkdownDescription: `
The Cachix provider allows you to manage Cachix binary caches as infrastructure-as-code.

## Authentication

The provider supports authentication via:
1. Explicit ` + "`auth_token`" + ` in the provider block
2. ` + "`CACHIX_AUTH_TOKEN`" + ` environment variable

## Example Usage

` + "```hcl" + `
provider "cachix" {
  # Token can be set via CACHIX_AUTH_TOKEN environment variable
  auth_token = var.cachix_auth_token
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"auth_token": schema.StringAttribute{
				Description:         "The Cachix API authentication token. Can also be set via the CACHIX_AUTH_TOKEN environment variable.",
				MarkdownDescription: "The Cachix API authentication token. Can also be set via the `CACHIX_AUTH_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_host": schema.StringAttribute{
				Description:         "The Cachix API host URL. Defaults to https://app.cachix.org/api/v1",
				MarkdownDescription: "The Cachix API host URL. Defaults to `https://app.cachix.org/api/v1`",
				Optional:            true,
			},
		},
	}
}

// Configure prepares a Cachix API client for data sources and resources.
func (p *CachixProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Cachix client")

	var config CachixProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values
	authToken := os.Getenv("CACHIX_AUTH_TOKEN")
	apiHost := "https://app.cachix.org/api/v1"

	// Override with explicit configuration if provided
	if !config.AuthToken.IsNull() && !config.AuthToken.IsUnknown() {
		authToken = config.AuthToken.ValueString()
	}

	if !config.APIHost.IsNull() && !config.APIHost.IsUnknown() {
		apiHost = config.APIHost.ValueString()
	}

	// Validate that auth token is provided
	if authToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_token"),
			"Missing Cachix API Token",
			"The provider cannot create the Cachix API client as there is a missing or empty value for the Cachix API token. "+
				"Set the auth_token value in the configuration or use the CACHIX_AUTH_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		return
	}

	tflog.Debug(ctx, "Creating Cachix client", map[string]any{
		"api_host": apiHost,
	})

	// Create the Cachix client
	client := NewCachixClient(apiHost, authToken, p.version)

	// Make the client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured Cachix client", map[string]any{
		"api_host": apiHost,
	})
}

// DataSources defines the data sources implemented in the provider.
func (p *CachixProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCacheDataSource,
		NewUserDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *CachixProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCacheResource,
	}
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CachixProvider{
			version: version,
		}
	}
}
