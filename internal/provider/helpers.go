// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// cacheNameValidator is a shared validator for cache name fields.
var cacheNameValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[a-z][a-z0-9-]*$`),
	"must start with a lowercase letter and contain only lowercase letters, numbers, and hyphens",
)

// CacheNameValidators returns the validators for cache name attributes.
func CacheNameValidators() []validator.String {
	return []validator.String{cacheNameValidator}
}

// getClientFromProviderData extracts the CachixClient from provider data.
// Returns nil if provider data is nil (during early configuration).
// Adds an error diagnostic if the type assertion fails.
func getClientFromProviderData(providerData any, diags *diag.Diagnostics, resourceType string) *CachixClient {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*CachixClient)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Unexpected %s Configure Type", resourceType),
			fmt.Sprintf("Expected *CachixClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

// requireClient checks if the client is configured and adds an error if not.
// Returns true if the client is available, false otherwise.
func requireClient(client *CachixClient, diags *diag.Diagnostics) bool {
	if client == nil {
		diags.AddError(
			"Unconfigured Cachix Client",
			"Expected configured Cachix client. Please report this issue to the provider developers.",
		)
		return false
	}
	return true
}

// APIErrorHandler handles API errors and adds appropriate diagnostics.
type APIErrorHandler struct {
	Diagnostics  *diag.Diagnostics
	ResourceType string
	ResourceName string
	Operation    string
}

// Handle processes an API error and adds appropriate diagnostics.
// Returns true if an error was handled (caller should return), false otherwise.
func (h *APIErrorHandler) Handle(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			h.Diagnostics.AddError(
				"Authentication Error",
				fmt.Sprintf("Unable to %s %s: authentication failed. Please verify your CACHIX_AUTH_TOKEN is valid. Details: %s",
					h.Operation, h.ResourceType, apiErr.Message),
			)
		case http.StatusNotFound:
			h.Diagnostics.AddError(
				fmt.Sprintf("%s Not Found", h.ResourceType),
				fmt.Sprintf("The %s '%s' was not found. Please verify the name exists.", h.ResourceType, h.ResourceName),
			)
		default:
			if apiErr.StatusCode >= 500 {
				h.Diagnostics.AddError(
					"Server Error",
					fmt.Sprintf("A server error occurred while %s the %s (HTTP %d). Please retry the operation. Details: %s",
						getOperationGerund(h.Operation), h.ResourceType, apiErr.StatusCode, apiErr.Message),
				)
			} else {
				h.Diagnostics.AddError(
					fmt.Sprintf("Unable to %s %s", capitalize(h.Operation), h.ResourceType),
					fmt.Sprintf("An error occurred while %s the %s: %s",
						getOperationGerund(h.Operation), h.ResourceType, apiErr.Message),
				)
			}
		}
	} else {
		h.Diagnostics.AddError(
			fmt.Sprintf("Unable to %s %s", capitalize(h.Operation), h.ResourceType),
			fmt.Sprintf("An unexpected error occurred while %s the %s. Please retry the operation or report this issue to the provider developers.\n\nError: %s",
				getOperationGerund(h.Operation), h.ResourceType, err.Error()),
		)
	}

	return true
}

// HandleNotFoundAsRemoved handles API errors, treating 404 as resource removal.
// Returns: shouldReturn, wasNotFound
func (h *APIErrorHandler) HandleNotFoundAsRemoved(err error) (shouldReturn bool, wasNotFound bool) {
	if err == nil {
		return false, false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return true, true
	}

	return h.Handle(err), false
}

// CacheModel is a common interface for cache data models.
type CacheModel interface {
	SetID(types.String)
	SetName(types.String)
	SetIsPublic(types.Bool)
	SetURI(types.String)
	SetPublicSigningKeys(types.List)
}

// mapCacheToState maps a Cache API response to the Terraform state model.
func mapCacheToState(ctx context.Context, cache *Cache, diags *diag.Diagnostics) (
	id, name types.String, isPublic types.Bool, uri types.String, publicSigningKeys types.List,
) {
	id = types.StringValue(cache.Name)
	name = types.StringValue(cache.Name)
	isPublic = types.BoolValue(cache.IsPublic)
	uri = types.StringValue(cache.URI)

	keys, d := types.ListValueFrom(ctx, types.StringType, cache.PublicSigningKeys)
	diags.Append(d...)
	publicSigningKeys = keys

	return
}

// getOperationGerund returns the gerund form of an operation verb.
func getOperationGerund(operation string) string {
	switch operation {
	case "create":
		return "creating"
	case "read":
		return "reading"
	case "delete":
		return "deleting"
	default:
		return operation + "ing"
	}
}

// capitalize returns the string with the first letter capitalized.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
