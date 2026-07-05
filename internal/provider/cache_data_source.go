// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	URI                        types.String `tfsdk:"uri"`
	IsPublic                   types.Bool   `tfsdk:"is_public"`
	PublicSigningKeys          types.List   `tfsdk:"public_signing_keys"`
	GithubUsername             types.String `tfsdk:"github_username"`
	Permission                 types.String `tfsdk:"permission"`
	PreferredCompressionMethod types.String `tfsdk:"preferred_compression_method"`
}

// applyCacheState copies mapped API values into the data source model.
func (m *CacheDataSourceModel) applyCacheState(s cacheState) {
	m.ID = s.ID
	m.Name = s.Name
	m.IsPublic = s.IsPublic
	m.URI = s.URI
	m.PublicSigningKeys = s.PublicSigningKeys
	m.GithubUsername = s.GithubUsername
	m.Permission = s.Permission
	m.PreferredCompressionMethod = s.PreferredCompressionMethod
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
				MarkdownDescription: "The identifier of the cache (same as name).",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Cachix cache to look up.",
				Required:            true,
				Validators:          CacheNameValidators(),
			},
			"uri": schema.StringAttribute{
				MarkdownDescription: "The full URI of the cache (e.g., `https://my-cache.cachix.org`).",
				Computed:            true,
			},
			"is_public": schema.BoolAttribute{
				MarkdownDescription: "Whether the cache is publicly readable.",
				Computed:            true,
			},
			"public_signing_keys": schema.ListAttribute{
				MarkdownDescription: "List of public signing keys for use in nix.conf `trusted-public-keys`.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"github_username": schema.StringAttribute{
				MarkdownDescription: "The GitHub username of the account that owns the cache.",
				Computed:            true,
			},
			"permission": schema.StringAttribute{
				MarkdownDescription: "The permission level the authenticated token has on the cache (`Read`, `Write`, or `Admin`).",
				Computed:            true,
			},
			"preferred_compression_method": schema.StringAttribute{
				MarkdownDescription: "The compression method used for the cache (`XZ` or `ZSTD`).",
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *CacheDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = getClientFromProviderData(req.ProviderData, &resp.Diagnostics, "Data Source")
}

// Read refreshes the Terraform state with the latest data from the API.
func (d *CacheDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CacheDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cacheName := data.Name.ValueString()

	tflog.Debug(ctx, "Reading cache data source", map[string]any{
		"cache_name": cacheName,
	})

	cache, err := d.client.GetCache(ctx, cacheName)
	errorHandler := &APIErrorHandler{
		Diagnostics:  &resp.Diagnostics,
		ResourceType: "Cache",
		ResourceName: cacheName,
		Operation:    "read",
	}
	if errorHandler.Handle(err) {
		return
	}

	tflog.Trace(ctx, "Successfully read cache data", map[string]any{
		"cache_name": cacheName,
		"uri":        cache.URI,
		"is_public":  cache.IsPublic,
	})

	data.applyCacheState(mapCacheToState(ctx, cache, &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
