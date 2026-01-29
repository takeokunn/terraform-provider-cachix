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
var _ datasource.DataSource = &UserDataSource{}

// NewUserDataSource creates a new user data source instance.
func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	client *CachixClient
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
}

// Metadata returns the data source type name.
func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source.
func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetches information about the current authenticated Cachix user.",
		MarkdownDescription: "Fetches information about the current authenticated Cachix user. Use this data source to retrieve details about the user associated with the configured API token.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the user (same as username).",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the authenticated user.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the authenticated user.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = getClientFromProviderData(req.ProviderData, &resp.Diagnostics, "Data Source")
}

// Read refreshes the Terraform state with the latest data from the API.
func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading user data source")

	user, err := d.client.GetUser(ctx)
	errorHandler := &APIErrorHandler{
		Diagnostics:  &resp.Diagnostics,
		ResourceType: "User",
		ResourceName: "current",
		Operation:    "read",
	}
	if errorHandler.Handle(err) {
		return
	}

	tflog.Trace(ctx, "Successfully read user data", map[string]any{
		"username": user.Username,
	})

	data.ID = types.StringValue(user.Username)
	data.Username = types.StringValue(user.Username)
	if user.Email != "" {
		data.Email = types.StringValue(user.Email)
	} else {
		data.Email = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
