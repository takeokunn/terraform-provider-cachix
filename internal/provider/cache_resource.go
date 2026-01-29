// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &CacheResource{}
	_ resource.ResourceWithConfigure   = &CacheResource{}
	_ resource.ResourceWithImportState = &CacheResource{}
)

// NewCacheResource creates a new cache resource instance.
func NewCacheResource() resource.Resource {
	return &CacheResource{}
}

// CacheResource defines the resource implementation.
type CacheResource struct {
	client *CachixClient
}

// CacheResourceModel describes the resource data model.
type CacheResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	IsPublic          types.Bool   `tfsdk:"is_public"`
	URI               types.String `tfsdk:"uri"`
	PublicSigningKeys types.List   `tfsdk:"public_signing_keys"`
}

// Metadata returns the resource type name.
func (r *CacheResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cache"
}

// Schema defines the schema for the resource.
func (r *CacheResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Cachix binary cache.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The identifier of the cache (same as name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the cache. Must be lowercase alphanumeric with hyphens, starting with a letter.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z][a-z0-9-]*$`),
						"must start with a lowercase letter and contain only lowercase letters, numbers, and hyphens",
					),
				},
			},
			"is_public": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the cache is publicly readable. Defaults to `true`.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"uri": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full URI of the cache (e.g., `https://my-cache.cachix.org`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_signing_keys": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of public signing keys for use in nix.conf.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *CacheResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*CachixClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *CachixClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates a new cache resource.
func (r *CacheResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CacheResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Cachix Client",
			"Expected configured Cachix client. Please report this issue to the provider developers.",
		)
		return
	}

	tflog.Debug(ctx, "Creating cache", map[string]any{
		"name":      data.Name.ValueString(),
		"is_public": data.IsPublic.ValueBool(),
	})

	cache, err := r.client.CreateCache(ctx, data.Name.ValueString(), data.IsPublic.ValueBool())
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			switch apiErr.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden:
				resp.Diagnostics.AddError(
					"Authentication Error",
					fmt.Sprintf("Unable to create cache: authentication failed. Please verify your CACHIX_AUTH_TOKEN is valid. Details: %s", apiErr.Message),
				)
			default:
				resp.Diagnostics.AddError(
					"Unable to Create Cache",
					fmt.Sprintf("An error occurred while creating the cache: %s", apiErr.Message),
				)
			}
		} else {
			resp.Diagnostics.AddError(
				"Unable to Create Cache",
				fmt.Sprintf("An unexpected error occurred while creating the cache. Please retry the operation or report this issue to the provider developers.\n\nError: %s", err.Error()),
			)
		}
		return
	}

	data.ID = types.StringValue(cache.Name)
	data.Name = types.StringValue(cache.Name)
	data.IsPublic = types.BoolValue(cache.IsPublic)
	data.URI = types.StringValue(cache.URI)

	publicSigningKeys, diags := types.ListValueFrom(ctx, types.StringType, cache.PublicSigningKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PublicSigningKeys = publicSigningKeys

	tflog.Trace(ctx, "Created cache", map[string]any{
		"name": data.Name.ValueString(),
		"uri":  data.URI.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data from the API.
func (r *CacheResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CacheResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Cachix Client",
			"Expected configured Cachix client. Please report this issue to the provider developers.",
		)
		return
	}

	tflog.Debug(ctx, "Reading cache", map[string]any{
		"name": data.Name.ValueString(),
	})

	cache, err := r.client.GetCache(ctx, data.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			switch apiErr.StatusCode {
			case http.StatusNotFound:
				tflog.Warn(ctx, "Cache not found, removing from state", map[string]any{
					"name": data.Name.ValueString(),
				})
				resp.State.RemoveResource(ctx)
				return
			case http.StatusUnauthorized, http.StatusForbidden:
				resp.Diagnostics.AddError(
					"Authentication Error",
					fmt.Sprintf("Unable to read cache: authentication failed. Please verify your CACHIX_AUTH_TOKEN is valid. Details: %s", apiErr.Message),
				)
			default:
				resp.Diagnostics.AddError(
					"Unable to Read Cache",
					fmt.Sprintf("An error occurred while reading the cache: %s", apiErr.Message),
				)
			}
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Cache",
				fmt.Sprintf("An unexpected error occurred while reading the cache. Please retry the operation or report this issue to the provider developers.\n\nError: %s", err.Error()),
			)
		}
		return
	}

	data.ID = types.StringValue(cache.Name)
	data.Name = types.StringValue(cache.Name)
	data.IsPublic = types.BoolValue(cache.IsPublic)
	data.URI = types.StringValue(cache.URI)

	publicSigningKeys, diags := types.ListValueFrom(ctx, types.StringType, cache.PublicSigningKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PublicSigningKeys = publicSigningKeys

	tflog.Trace(ctx, "Read cache", map[string]any{
		"name": data.Name.ValueString(),
		"uri":  data.URI.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not supported for cache resources as all attributes require replacement.
func (r *CacheResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"The cachix_cache resource does not support updates. All attribute changes require resource replacement (ForceNew).",
	)
}

// Delete removes the cache resource.
func (r *CacheResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CacheResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Cachix Client",
			"Expected configured Cachix client. Please report this issue to the provider developers.",
		)
		return
	}

	tflog.Debug(ctx, "Deleting cache", map[string]any{
		"name": data.Name.ValueString(),
	})

	err := r.client.DeleteCache(ctx, data.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			switch apiErr.StatusCode {
			case http.StatusNotFound:
				tflog.Warn(ctx, "Cache already deleted", map[string]any{
					"name": data.Name.ValueString(),
				})
				return
			case http.StatusUnauthorized, http.StatusForbidden:
				resp.Diagnostics.AddError(
					"Authentication Error",
					fmt.Sprintf("Unable to delete cache: authentication failed. Please verify your CACHIX_AUTH_TOKEN is valid. Details: %s", apiErr.Message),
				)
			default:
				if apiErr.StatusCode >= 500 {
					resp.Diagnostics.AddError(
						"Server Error",
						fmt.Sprintf("A server error occurred while deleting the cache. Please retry the operation. Details: %s", apiErr.Message),
					)
				} else {
					resp.Diagnostics.AddError(
						"Unable to Delete Cache",
						fmt.Sprintf("An error occurred while deleting the cache: %s", apiErr.Message),
					)
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Unable to Delete Cache",
				fmt.Sprintf("An unexpected error occurred while deleting the cache. Please retry the operation or report this issue to the provider developers.\n\nError: %s", err.Error()),
			)
		}
		return
	}

	tflog.Trace(ctx, "Deleted cache", map[string]any{
		"name": data.Name.ValueString(),
	})
}

// ImportState imports an existing cache into Terraform state.
func (r *CacheResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing cache", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
