// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &CacheDataSource{}

// NewCacheDataSource creates a new cache data source instance.
func NewCacheDataSource() datasource.DataSource {
	return &CacheDataSource{}
}

// CacheDataSource defines the data source implementation.
type CacheDataSource struct {
	client *CachixClient
}

// CacheDataSourceModel describes the data source data model.
type CacheDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	URI               types.String `tfsdk:"uri"`
	IsPublic          types.Bool   `tfsdk:"is_public"`
	PublicSigningKeys types.List   `tfsdk:"public_signing_keys"`
}

// Metadata returns the data source type name.
func (d *CacheDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cache"
}

// Schema defines the schema for the data source.
func (d *CacheDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetches information about a Cachix binary cache.",
		MarkdownDescription: "Fetches information about a Cachix binary cache. Use this data source to reference existing caches that are not managed by Terraform.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The identifier of the cache (same as name).",
				MarkdownDescription: "The identifier of the cache (same as name).",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The name of the Cachix cache to look up.",
				MarkdownDescription: "The name of the Cachix cache to look up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z][a-z0-9-]*$`),
						"must start with a lowercase letter and contain only lowercase letters, numbers, and hyphens",
					),
				},
			},
			"uri": schema.StringAttribute{
				Description:         "The full URI of the cache (e.g., https://my-cache.cachix.org).",
				MarkdownDescription: "The full URI of the cache (e.g., `https://my-cache.cachix.org`).",
				Computed:            true,
			},
			"is_public": schema.BoolAttribute{
				Description:         "Whether the cache is publicly readable.",
				MarkdownDescription: "Whether the cache is publicly readable.",
				Computed:            true,
			},
			"public_signing_keys": schema.ListAttribute{
				Description:         "List of public signing keys for use in nix.conf trusted-public-keys.",
				MarkdownDescription: "List of public signing keys for use in nix.conf `trusted-public-keys`.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *CacheDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CachixClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *CachixClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data from the API.
func (d *CacheDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CacheDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	cacheName := data.Name.ValueString()

	tflog.Debug(ctx, "Reading cache data source", map[string]any{
		"cache_name": cacheName,
	})

	// Call the API to get cache information
	cache, err := d.client.GetCache(ctx, cacheName)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			switch apiErr.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden:
				resp.Diagnostics.AddError(
					"Authentication Error",
					fmt.Sprintf("Failed to authenticate with Cachix API: %s. Please verify your auth_token is valid and has permission to access this cache.", err),
				)
			case http.StatusNotFound:
				resp.Diagnostics.AddError(
					"Cache Not Found",
					fmt.Sprintf("The cache '%s' was not found. Please verify the cache name exists.", cacheName),
				)
			default:
				if apiErr.StatusCode >= 500 {
					resp.Diagnostics.AddError(
						"Cachix API Error",
						fmt.Sprintf("The Cachix API returned an error (HTTP %d): %s. Please try again later.", apiErr.StatusCode, err),
					)
				} else {
					resp.Diagnostics.AddError(
						"Error Reading Cache",
						fmt.Sprintf("Unable to read cache '%s': %s", cacheName, err),
					)
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Error Reading Cache",
				fmt.Sprintf("Unable to read cache '%s': %s", cacheName, err),
			)
		}
		return
	}

	tflog.Trace(ctx, "Successfully read cache data", map[string]any{
		"cache_name": cacheName,
		"uri":        cache.URI,
		"is_public":  cache.IsPublic,
	})

	// Map response to model
	data.ID = types.StringValue(cache.Name)
	data.Name = types.StringValue(cache.Name)
	data.URI = types.StringValue(cache.URI)
	data.IsPublic = types.BoolValue(cache.IsPublic)

	// Convert signing keys to types.List
	publicSigningKeys, diags := types.ListValueFrom(ctx, types.StringType, cache.PublicSigningKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PublicSigningKeys = publicSigningKeys

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
