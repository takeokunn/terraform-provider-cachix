// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cachix": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates the necessary test API keys exist in the testing
// environment.
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CACHIX_AUTH_TOKEN"); v == "" {
		t.Fatal("CACHIX_AUTH_TOKEN must be set for acceptance tests")
	}
}

func TestProvider_Metadata(t *testing.T) {
	p := New("1.0.0")()

	req := provider.MetadataRequest{}
	resp := &provider.MetadataResponse{}

	p.Metadata(context.Background(), req, resp)

	if resp.TypeName != "cachix" {
		t.Errorf("expected TypeName 'cachix', got '%s'", resp.TypeName)
	}
	if resp.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", resp.Version)
	}
}

func TestProvider_Metadata_DevVersion(t *testing.T) {
	p := New("dev")()

	req := provider.MetadataRequest{}
	resp := &provider.MetadataResponse{}

	p.Metadata(context.Background(), req, resp)

	if resp.TypeName != "cachix" {
		t.Errorf("expected TypeName 'cachix', got '%s'", resp.TypeName)
	}
	if resp.Version != "dev" {
		t.Errorf("expected Version 'dev', got '%s'", resp.Version)
	}
}

func TestProvider_Schema(t *testing.T) {
	p := New("test")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	if resp.Schema.Attributes == nil {
		t.Fatal("expected schema attributes, got nil")
	}

	// Check auth_token attribute exists
	if _, ok := resp.Schema.Attributes["auth_token"]; !ok {
		t.Error("expected 'auth_token' attribute in schema")
	}

	// Check api_host attribute exists
	if _, ok := resp.Schema.Attributes["api_host"]; !ok {
		t.Error("expected 'api_host' attribute in schema")
	}
}

func TestProvider_Schema_AuthTokenProperties(t *testing.T) {
	p := New("test")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	authToken, ok := resp.Schema.Attributes["auth_token"]
	if !ok {
		t.Fatal("expected 'auth_token' attribute in schema")
	}

	// auth_token should be optional (since env var can be used)
	if authToken.IsRequired() {
		t.Error("expected 'auth_token' to be optional, not required")
	}

	// auth_token should be sensitive
	if !authToken.IsSensitive() {
		t.Error("expected 'auth_token' to be sensitive")
	}
}

func TestProvider_Schema_APIHostProperties(t *testing.T) {
	p := New("test")()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	apiHost, ok := resp.Schema.Attributes["api_host"]
	if !ok {
		t.Fatal("expected 'api_host' attribute in schema")
	}

	// api_host should be optional
	if apiHost.IsRequired() {
		t.Error("expected 'api_host' to be optional, not required")
	}

	// api_host should NOT be sensitive
	if apiHost.IsSensitive() {
		t.Error("expected 'api_host' to not be sensitive")
	}
}

func TestProvider_Configure_WithEnvVar(t *testing.T) {
	// Set the environment variable for the test
	t.Setenv("CACHIX_AUTH_TOKEN", "test-env-token")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create an empty config (no explicit token)
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, nil),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify the client was created and set
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProvider_Configure_WithExplicitToken(t *testing.T) {
	// Unset environment variable to ensure explicit token is used
	t.Setenv("CACHIX_AUTH_TOKEN", "")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config with explicit token
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, "explicit-test-token"),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify the client was created
	if resp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
	if resp.ResourceData == nil {
		t.Error("expected ResourceData to be set")
	}
}

func TestProvider_Configure_TokenPrecedence(t *testing.T) {
	// Set environment variable
	t.Setenv("CACHIX_AUTH_TOKEN", "env-token")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config with explicit token (should override env var)
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, "explicit-token"),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify the client was created with explicit token
	client, ok := resp.DataSourceData.(*CachixClient)
	if !ok {
		t.Fatal("expected DataSourceData to be *CachixClient")
	}

	// The explicit token should be used (not the env var)
	if client.authToken != "explicit-token" {
		t.Errorf("expected auth token 'explicit-token', got '%s'", client.authToken)
	}
}

func TestProvider_Configure_CustomAPIHost(t *testing.T) {
	t.Setenv("CACHIX_AUTH_TOKEN", "test-token")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config with custom API host
	customHost := "https://custom.cachix.org/api/v2"
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, nil),
		"api_host":   tftypes.NewValue(tftypes.String, customHost),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify the client was created with custom host
	client, ok := resp.DataSourceData.(*CachixClient)
	if !ok {
		t.Fatal("expected DataSourceData to be *CachixClient")
	}

	if client.baseURL != customHost {
		t.Errorf("expected base URL '%s', got '%s'", customHost, client.baseURL)
	}
}

func TestProvider_Configure_DefaultAPIHost(t *testing.T) {
	t.Setenv("CACHIX_AUTH_TOKEN", "test-token")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config without custom API host (should use default)
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, nil),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify the client was created with default host
	client, ok := resp.DataSourceData.(*CachixClient)
	if !ok {
		t.Fatal("expected DataSourceData to be *CachixClient")
	}

	expectedHost := "https://app.cachix.org/api/v1"
	if client.baseURL != expectedHost {
		t.Errorf("expected base URL '%s', got '%s'", expectedHost, client.baseURL)
	}
}

func TestProvider_Configure_MissingToken(t *testing.T) {
	// Ensure no token is available
	t.Setenv("CACHIX_AUTH_TOKEN", "")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config without token
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, nil),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	// Should have an error about missing token
	if !resp.Diagnostics.HasError() {
		t.Error("expected error for missing token, got none")
	}

	// Verify the error message mentions the token
	found := false
	for _, diag := range resp.Diagnostics.Errors() {
		if diag.Summary() == "Missing Cachix API Token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error summary 'Missing Cachix API Token'")
	}

	// Client should not be set when there's an error
	if resp.DataSourceData != nil {
		t.Error("expected DataSourceData to be nil on error")
	}
	if resp.ResourceData != nil {
		t.Error("expected ResourceData to be nil on error")
	}
}

func TestProvider_Configure_EmptyExplicitToken(t *testing.T) {
	// Ensure no token from env
	t.Setenv("CACHIX_AUTH_TOKEN", "")

	p := New("test")()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	// Create config with empty string token
	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, ""),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	// Should have an error about missing/empty token
	if !resp.Diagnostics.HasError() {
		t.Error("expected error for empty token, got none")
	}
}

func TestProvider_DataSources(t *testing.T) {
	p := New("test")()

	dataSources := p.DataSources(context.Background())

	// Verify expected number of data sources
	expectedCount := 2 // cache and user
	if len(dataSources) != expectedCount {
		t.Errorf("expected %d data sources, got %d", expectedCount, len(dataSources))
	}

	// Verify each factory returns a valid data source
	for i, factory := range dataSources {
		ds := factory()
		if ds == nil {
			t.Errorf("data source factory %d returned nil", i)
		}
	}
}

func TestProvider_DataSources_TypeNames(t *testing.T) {
	p := New("test")()

	dataSources := p.DataSources(context.Background())

	// We just verify that the expected number of data sources is returned
	// Type name verification is covered by individual data source tests
	expectedCount := 2 // cache and user
	if len(dataSources) != expectedCount {
		t.Errorf("expected %d data sources, got %d", expectedCount, len(dataSources))
	}
}

func TestProvider_Resources(t *testing.T) {
	p := New("test")()

	resources := p.Resources(context.Background())

	// Verify expected number of resources
	expectedCount := 1 // cache
	if len(resources) != expectedCount {
		t.Errorf("expected %d resources, got %d", expectedCount, len(resources))
	}

	// Verify each factory returns a valid resource
	for i, factory := range resources {
		r := factory()
		if r == nil {
			t.Errorf("resource factory %d returned nil", i)
		}
	}
}

func TestNew_ReturnsProviderFactory(t *testing.T) {
	factory := New("1.2.3")

	if factory == nil {
		t.Fatal("expected factory function, got nil")
	}

	p := factory()
	if p == nil {
		t.Fatal("expected provider instance, got nil")
	}

	// Verify it's a CachixProvider
	cachixProvider, ok := p.(*CachixProvider)
	if !ok {
		t.Fatal("expected *CachixProvider type")
	}

	if cachixProvider.version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got '%s'", cachixProvider.version)
	}
}

func TestNew_MultipleFactoryCalls(t *testing.T) {
	factory := New("test")

	// Multiple calls should return independent instances
	p1 := factory()
	p2 := factory()

	if p1 == p2 {
		t.Error("expected independent provider instances")
	}
}

func TestProvider_ImplementsInterface(t *testing.T) {
	p := New("test")()

	// Verify the provider implements the required interface
	var _ provider.Provider = p
}

func TestProvider_Configure_ClientVersion(t *testing.T) {
	t.Setenv("CACHIX_AUTH_TOKEN", "test-token")

	version := "1.5.0"
	p := New(version)()

	// First get the schema
	schemaReq := provider.SchemaRequest{}
	schemaResp := &provider.SchemaResponse{}
	p.Schema(context.Background(), schemaReq, schemaResp)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"auth_token": tftypes.String,
			"api_host":   tftypes.String,
		},
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, nil),
		"api_host":   tftypes.NewValue(tftypes.String, nil),
	})

	config := tfsdk.Config{
		Raw:    configValue,
		Schema: schemaResp.Schema,
	}

	req := provider.ConfigureRequest{
		Config: config,
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}

	// Verify the client has correct user agent with version
	client, ok := resp.DataSourceData.(*CachixClient)
	if !ok {
		t.Fatal("expected DataSourceData to be *CachixClient")
	}

	expectedUserAgent := "terraform-provider-cachix/1.5.0"
	if client.userAgent != expectedUserAgent {
		t.Errorf("expected user agent '%s', got '%s'", expectedUserAgent, client.userAgent)
	}
}
