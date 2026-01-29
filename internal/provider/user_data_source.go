// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

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
				Description:         "The identifier of the user (same as username).",
				MarkdownDescription: "The identifier of the user (same as username).",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				Description:         "The username of the authenticated user.",
				MarkdownDescription: "The username of the authenticated user.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				Description:         "The email address of the authenticated user.",
				MarkdownDescription: "The email address of the authenticated user.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading user data source")

	// Call the API to get user information
	user, err := d.client.GetUser(ctx)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			switch apiErr.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden:
				resp.Diagnostics.AddError(
					"Authentication Error",
					fmt.Sprintf("Failed to authenticate with Cachix API: %s. Please verify your auth_token is valid.", err),
				)
			case http.StatusNotFound:
				resp.Diagnostics.AddError(
					"User Not Found",
					"Unable to retrieve user information. The authenticated user may not exist or the token may be invalid.",
				)
			default:
				if apiErr.StatusCode >= 500 {
					resp.Diagnostics.AddError(
						"Cachix API Error",
						fmt.Sprintf("The Cachix API returned an error (HTTP %d): %s. Please try again later.", apiErr.StatusCode, err),
					)
				} else {
					resp.Diagnostics.AddError(
						"Error Reading User",
						fmt.Sprintf("Unable to read user: %s", err),
					)
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Error Reading User",
				fmt.Sprintf("Unable to read user: %s", err),
			)
		}
		return
	}

	tflog.Trace(ctx, "Successfully read user data", map[string]any{
		"username": user.Username,
	})

	// Map response to model
	data.ID = types.StringValue(user.Username)
	data.Username = types.StringValue(user.Username)
	if user.Email != "" {
		data.Email = types.StringValue(user.Email)
	} else {
		data.Email = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
