// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCachixClient(t *testing.T) {
	t.Run("creates client with correct values", func(t *testing.T) {
		baseURL := "https://api.cachix.org"
		authToken := "test-token-123"
		version := "1.0.0"

		client := NewCachixClient(baseURL, authToken, version)

		if client.baseURL != baseURL {
			t.Errorf("expected baseURL %q, got %q", baseURL, client.baseURL)
		}
		if client.authToken != authToken {
			t.Errorf("expected authToken %q, got %q", authToken, client.authToken)
		}
		expectedUserAgent := "terraform-provider-cachix/1.0.0"
		if client.userAgent != expectedUserAgent {
			t.Errorf("expected userAgent %q, got %q", expectedUserAgent, client.userAgent)
		}
	})

	t.Run("uses default retry settings", func(t *testing.T) {
		client := NewCachixClient("", "", "")

		if client.retryMax != DefaultRetryMax {
			t.Errorf("expected retryMax %d, got %d", DefaultRetryMax, client.retryMax)
		}
	})

	t.Run("creates HTTP client with timeout", func(t *testing.T) {
		client := NewCachixClient("", "", "")

		if client.httpClient == nil {
			t.Fatal("expected httpClient to be initialized")
		}
		expectedTimeout := 30 * time.Second
		if client.httpClient.Timeout != expectedTimeout {
			t.Errorf("expected timeout %v, got %v", expectedTimeout, client.httpClient.Timeout)
		}
	})

	t.Run("handles empty version", func(t *testing.T) {
		client := NewCachixClient("https://api.cachix.org", "token", "")

		expectedUserAgent := "terraform-provider-cachix/"
		if client.userAgent != expectedUserAgent {
			t.Errorf("expected userAgent %q, got %q", expectedUserAgent, client.userAgent)
		}
	})
}

func TestCachixClient_shouldRetry(t *testing.T) {
	client := NewCachixClient("", "", "")

	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		// Success responses - should NOT retry
		{"200 OK", 200, false},
		{"201 Created", 201, false},
		{"204 No Content", 204, false},

		// Client errors - should NOT retry
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"403 Forbidden", 403, false},
		{"404 Not Found", 404, false},
		{"409 Conflict", 409, false},
		{"422 Unprocessable Entity", 422, false},

		// Rate limiting - SHOULD retry
		{"429 Too Many Requests", 429, true},

		// Server errors - SHOULD retry
		{"500 Internal Server Error", 500, true},
		{"501 Not Implemented", 501, true},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
		{"504 Gateway Timeout", 504, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d_%s", tt.statusCode, tt.name), func(t *testing.T) {
			got := client.shouldRetry(tt.statusCode)
			if got != tt.expected {
				t.Errorf("shouldRetry(%d) = %v, want %v", tt.statusCode, got, tt.expected)
			}
		})
	}
}

func TestCachixClient_calculateBackoff(t *testing.T) {
	client := NewCachixClient("", "", "")

	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
		description string
	}{
		{
			attempt:     1,
			expectedMin: 1 * time.Second,
			expectedMax: 1 * time.Second,
			description: "first retry: 1 second",
		},
		{
			attempt:     2,
			expectedMin: 2 * time.Second,
			expectedMax: 2 * time.Second,
			description: "second retry: 2 seconds",
		},
		{
			attempt:     3,
			expectedMin: 4 * time.Second,
			expectedMax: 4 * time.Second,
			description: "third retry: 4 seconds",
		},
		{
			attempt:     4,
			expectedMin: 8 * time.Second,
			expectedMax: 8 * time.Second,
			description: "fourth retry: 8 seconds",
		},
		{
			attempt:     5,
			expectedMin: 16 * time.Second,
			expectedMax: 16 * time.Second,
			description: "fifth retry: 16 seconds",
		},
		{
			attempt:     6,
			expectedMin: 30 * time.Second,
			expectedMax: 30 * time.Second,
			description: "sixth retry: capped at 30 seconds",
		},
		{
			attempt:     10,
			expectedMin: 30 * time.Second,
			expectedMax: 30 * time.Second,
			description: "large attempt: capped at 30 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got := client.calculateBackoff(tt.attempt)
			if got < tt.expectedMin || got > tt.expectedMax {
				t.Errorf("calculateBackoff(%d) = %v, want between %v and %v",
					tt.attempt, got, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCachixClient_handleErrorResponse(t *testing.T) {
	client := NewCachixClient("", "", "")

	tests := []struct {
		name           string
		statusCode     int
		body           []byte
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "401 with JSON error field",
			statusCode:     http.StatusUnauthorized,
			body:           []byte(`{"error": "invalid token"}`),
			expectedStatus: 401,
			expectedMsg:    "invalid token",
		},
		{
			name:           "401 with JSON message field",
			statusCode:     http.StatusUnauthorized,
			body:           []byte(`{"message": "token expired"}`),
			expectedStatus: 401,
			expectedMsg:    "token expired",
		},
		{
			name:           "401 with empty body",
			statusCode:     http.StatusUnauthorized,
			body:           []byte(``),
			expectedStatus: 401,
			expectedMsg:    "authentication failed: invalid or expired token",
		},
		{
			name:           "401 with invalid JSON",
			statusCode:     http.StatusUnauthorized,
			body:           []byte(`not json`),
			expectedStatus: 401,
			expectedMsg:    "authentication failed: invalid or expired token",
		},
		{
			name:           "403 with empty body",
			statusCode:     http.StatusForbidden,
			body:           []byte(``),
			expectedStatus: 403,
			expectedMsg:    "access denied: insufficient permissions",
		},
		{
			name:           "403 with JSON error",
			statusCode:     http.StatusForbidden,
			body:           []byte(`{"error": "permission denied for resource"}`),
			expectedStatus: 403,
			expectedMsg:    "permission denied for resource",
		},
		{
			name:           "404 with empty body",
			statusCode:     http.StatusNotFound,
			body:           []byte(``),
			expectedStatus: 404,
			expectedMsg:    "resource not found",
		},
		{
			name:           "404 with JSON message",
			statusCode:     http.StatusNotFound,
			body:           []byte(`{"message": "cache my-cache not found"}`),
			expectedStatus: 404,
			expectedMsg:    "cache my-cache not found",
		},
		{
			name:           "500 with empty body",
			statusCode:     http.StatusInternalServerError,
			body:           []byte(``),
			expectedStatus: 500,
			expectedMsg:    "",
		},
		{
			name:           "500 with plain text body",
			statusCode:     http.StatusInternalServerError,
			body:           []byte(`Internal Server Error`),
			expectedStatus: 500,
			expectedMsg:    "",
		},
		{
			name:           "400 with JSON error",
			statusCode:     http.StatusBadRequest,
			body:           []byte(`{"error": "invalid cache name"}`),
			expectedStatus: 400,
			expectedMsg:    "invalid cache name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.handleErrorResponse(tt.statusCode, tt.body)

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %T", err)
			}

			if apiErr.StatusCode != tt.expectedStatus {
				t.Errorf("expected StatusCode %d, got %d", tt.expectedStatus, apiErr.StatusCode)
			}

			if apiErr.Message != tt.expectedMsg {
				t.Errorf("expected Message %q, got %q", tt.expectedMsg, apiErr.Message)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError *APIError
		expected string
	}{
		{
			name: "with message",
			apiError: &APIError{
				StatusCode: 401,
				Message:    "invalid token",
				Body:       `{"error": "invalid token"}`,
			},
			expected: "cachix API error (status 401): invalid token",
		},
		{
			name: "without message uses body",
			apiError: &APIError{
				StatusCode: 500,
				Message:    "",
				Body:       "Internal Server Error",
			},
			expected: "cachix API error (status 500): Internal Server Error",
		},
		{
			name: "empty message and body",
			apiError: &APIError{
				StatusCode: 404,
				Message:    "",
				Body:       "",
			},
			expected: "cachix API error (status 404): ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apiError.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "404 APIError returns true",
			err: &APIError{
				StatusCode: http.StatusNotFound,
				Message:    "not found",
			},
			expected: true,
		},
		{
			name: "401 APIError returns false",
			err: &APIError{
				StatusCode: http.StatusUnauthorized,
				Message:    "unauthorized",
			},
			expected: false,
		},
		{
			name: "403 APIError returns false",
			err: &APIError{
				StatusCode: http.StatusForbidden,
				Message:    "forbidden",
			},
			expected: false,
		},
		{
			name: "500 APIError returns false",
			err: &APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "server error",
			},
			expected: false,
		},
		{
			name:     "generic error returns false",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundError(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFoundError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	t.Run("DefaultRetryMax is 3", func(t *testing.T) {
		if DefaultRetryMax != 3 {
			t.Errorf("expected DefaultRetryMax to be 3, got %d", DefaultRetryMax)
		}
	})

	t.Run("DefaultRetryWaitMin is 1 second", func(t *testing.T) {
		expected := 1 * time.Second
		if DefaultRetryWaitMin != expected {
			t.Errorf("expected DefaultRetryWaitMin to be %v, got %v", expected, DefaultRetryWaitMin)
		}
	})

	t.Run("DefaultRetryWaitMax is 30 seconds", func(t *testing.T) {
		expected := 30 * time.Second
		if DefaultRetryWaitMax != expected {
			t.Errorf("expected DefaultRetryWaitMax to be %v, got %v", expected, DefaultRetryWaitMax)
		}
	})
}

func TestCacheStruct(t *testing.T) {
	t.Run("Cache fields are accessible", func(t *testing.T) {
		cache := Cache{
			Name:              "my-cache",
			URI:               "https://my-cache.cachix.org",
			IsPublic:          true,
			PublicSigningKeys: []string{"key1", "key2"},
			CreatedAt:         "2024-01-01T00:00:00Z",
		}

		if cache.Name != "my-cache" {
			t.Errorf("expected Name %q, got %q", "my-cache", cache.Name)
		}
		if cache.URI != "https://my-cache.cachix.org" {
			t.Errorf("expected URI %q, got %q", "https://my-cache.cachix.org", cache.URI)
		}
		if !cache.IsPublic {
			t.Error("expected IsPublic to be true")
		}
		if len(cache.PublicSigningKeys) != 2 {
			t.Errorf("expected 2 signing keys, got %d", len(cache.PublicSigningKeys))
		}
		if cache.CreatedAt != "2024-01-01T00:00:00Z" {
			t.Errorf("expected CreatedAt %q, got %q", "2024-01-01T00:00:00Z", cache.CreatedAt)
		}
	})
}

func TestCreateCacheRequestStruct(t *testing.T) {
	t.Run("CreateCacheRequest fields are accessible", func(t *testing.T) {
		req := CreateCacheRequest{
			IsPublic: true,
		}

		if !req.IsPublic {
			t.Error("expected IsPublic to be true")
		}
	})
}

// =============================================================================
// HTTP Mock Tests using httptest.Server
// =============================================================================

func TestCachixClient_GetCache_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// Verify request path
		if r.URL.Path != "/cache/test-cache" {
			t.Errorf("expected /cache/test-cache, got %s", r.URL.Path)
		}
		// Verify Authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or invalid Authorization header: got %q", r.Header.Get("Authorization"))
		}
		// Verify Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}
		// Verify User-Agent header
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "terraform-provider-cachix/") {
			t.Errorf("expected User-Agent to start with terraform-provider-cachix/, got %q", r.Header.Get("User-Agent"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Cache{
			Name:              "test-cache",
			URI:               "https://test-cache.cachix.org",
			IsPublic:          true,
			PublicSigningKeys: []string{"test-cache.cachix.org-1:xxxx="},
			CreatedAt:         "2024-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.GetCache(context.Background(), "test-cache")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.Name != "test-cache" {
		t.Errorf("expected name 'test-cache', got '%s'", cache.Name)
	}
	if cache.URI != "https://test-cache.cachix.org" {
		t.Errorf("expected URI 'https://test-cache.cachix.org', got '%s'", cache.URI)
	}
	if !cache.IsPublic {
		t.Error("expected IsPublic to be true")
	}
	if len(cache.PublicSigningKeys) != 1 {
		t.Errorf("expected 1 signing key, got %d", len(cache.PublicSigningKeys))
	}
}

func TestCachixClient_GetCache_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "cache 'nonexistent-cache' not found",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.GetCache(context.Background(), "nonexistent-cache")

	if cache != nil {
		t.Error("expected cache to be nil")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if !IsNotFoundError(err) {
		t.Error("expected IsNotFoundError to return true")
	}
}

func TestCachixClient_GetCache_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid token",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "invalid-token", "1.0.0")
	cache, err := client.GetCache(context.Background(), "test-cache")

	if cache != nil {
		t.Error("expected cache to be nil")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid token" {
		t.Errorf("expected message 'invalid token', got '%s'", apiErr.Message)
	}
}

func TestCachixClient_GetCache_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return malformed JSON
		_, _ = w.Write([]byte(`{"name": "test-cache", "uri": `))
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.GetCache(context.Background(), "test-cache")

	if cache != nil {
		t.Error("expected cache to be nil")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestCachixClient_CreateCache_Success(t *testing.T) {
	var postCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user":
			// Return user info for accountID
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(User{
				ID:       12345,
				Username: "testuser",
				Email:    "testuser@example.com",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/cache/new-cache":
			postCalled = true
			// Verify request body
			var reqBody CreateCacheRequest
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			if !reqBody.IsPublic {
				t.Error("expected IsPublic to be true in request")
			}
			if !reqBody.GenerateSigningKey {
				t.Error("expected GenerateSigningKey to be true in request")
			}
			if reqBody.AccountID != 12345 {
				t.Errorf("expected AccountID 12345, got %d", reqBody.AccountID)
			}
			// API returns empty body on success
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/cache/new-cache":
			// Return cache details after creation
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Cache{
				Name:              "new-cache",
				URI:               "https://new-cache.cachix.org",
				IsPublic:          true,
				PublicSigningKeys: []string{"new-cache.cachix.org-1:yyyy="},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.CreateCache(context.Background(), "new-cache", true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !postCalled {
		t.Error("expected POST to /cache/new-cache to be called")
	}
	if cache.Name != "new-cache" {
		t.Errorf("expected name 'new-cache', got '%s'", cache.Name)
	}
	if cache.URI != "https://new-cache.cachix.org" {
		t.Errorf("expected URI 'https://new-cache.cachix.org', got '%s'", cache.URI)
	}
	if !cache.IsPublic {
		t.Error("expected IsPublic to be true")
	}
}

func TestCachixClient_CreateCache_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(User{
				ID:       12345,
				Username: "testuser",
				Email:    "testuser@example.com",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/cache/existing-cache":
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "cache 'existing-cache' already exists",
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.CreateCache(context.Background(), "existing-cache", true)

	if cache != nil {
		t.Error("expected cache to be nil")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "cache 'existing-cache' already exists" {
		t.Errorf("expected conflict message, got '%s'", apiErr.Message)
	}
}

func TestCachixClient_DeleteCache_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// Verify request path
		if r.URL.Path != "/cache/delete-me" {
			t.Errorf("expected /cache/delete-me, got %s", r.URL.Path)
		}
		// Verify Authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or invalid Authorization header")
		}

		// Return 204 No Content
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	err := client.DeleteCache(context.Background(), "delete-me")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCachixClient_DeleteCache_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "cache 'nonexistent-cache' not found",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	err := client.DeleteCache(context.Background(), "nonexistent-cache")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if !IsNotFoundError(err) {
		t.Error("expected IsNotFoundError to return true")
	}
}

func TestCachixClient_GetUser_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// Verify request path
		if r.URL.Path != "/user" {
			t.Errorf("expected /user, got %s", r.URL.Path)
		}
		// Verify Authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or invalid Authorization header")
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{
			ID:       12345,
			Username: "testuser",
			Email:    "testuser@example.com",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	user, err := client.GetUser(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 12345 {
		t.Errorf("expected id 12345, got %d", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", user.Username)
	}
	if user.Email != "testuser@example.com" {
		t.Errorf("expected email 'testuser@example.com', got '%s'", user.Email)
	}
}

func TestCachixClient_GetUser_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "token has expired",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "expired-token", "1.0.0")
	user, err := client.GetUser(context.Background())

	if user != nil {
		t.Error("expected user to be nil")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "token has expired" {
		t.Errorf("expected message 'token has expired', got '%s'", apiErr.Message)
	}
}

func TestCachixClient_doRequest_RetryOn5xx(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)

		// Return 503 for first 2 attempts, then 200
		if attempt <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error": "service temporarily unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Cache{
			Name:     "test-cache",
			URI:      "https://test-cache.cachix.org",
			IsPublic: true,
		})
	}))
	defer server.Close()

	// Create client with custom retry settings for faster test
	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	// Note: The client uses default retry settings; test will be slower but accurate

	cache, err := client.GetCache(context.Background(), "test-cache")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.Name != "test-cache" {
		t.Errorf("expected name 'test-cache', got '%s'", cache.Name)
	}

	// Verify retry happened (should be 3 attempts: initial + 2 retries)
	finalAttempts := atomic.LoadInt32(&attemptCount)
	if finalAttempts != 3 {
		t.Errorf("expected 3 attempts, got %d", finalAttempts)
	}
}

func TestCachixClient_doRequest_RetryOn429(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)

		// Return 429 for first attempt, then 200
		if attempt == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "rate limit exceeded"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{
			ID:       12345,
			Username: "testuser",
			Email:    "testuser@example.com",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	user, err := client.GetUser(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", user.Username)
	}

	// Verify retry happened (should be 2 attempts: initial + 1 retry)
	finalAttempts := atomic.LoadInt32(&attemptCount)
	if finalAttempts != 2 {
		t.Errorf("expected 2 attempts, got %d", finalAttempts)
	}
}

func TestCachixClient_doRequest_MaxRetriesExceeded(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		// Always return 500 to exhaust retries
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	_, err := client.GetCache(context.Background(), "test-cache")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("expected 'max retries exceeded' error, got: %v", err)
	}

	// Verify all retries were attempted (initial + 3 retries = 4 total)
	finalAttempts := atomic.LoadInt32(&attemptCount)
	expectedAttempts := int32(DefaultRetryMax + 1)
	if finalAttempts != expectedAttempts {
		t.Errorf("expected %d attempts, got %d", expectedAttempts, finalAttempts)
	}
}

func TestCachixClient_doRequest_ContextCancellation(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		// Always return 503 to trigger retry
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "service unavailable"}`))
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start the request in a goroutine
	errCh := make(chan error, 1)
	go func() {
		_, err := client.GetCache(ctx, "test-cache")
		errCh <- err
	}()

	// Wait for at least one attempt
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the result
	err := <-errCh

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestCachixClient_doRequest_NoRetryOn4xx(t *testing.T) {
	var attemptCount int32

	tests := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
		{"404 Not Found", http.StatusNotFound},
		{"409 Conflict", http.StatusConflict},
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atomic.StoreInt32(&attemptCount, 0)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attemptCount, 1)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"error": "client error"}`))
			}))
			defer server.Close()

			client := NewCachixClient(server.URL, "test-token", "1.0.0")
			_, err := client.GetCache(context.Background(), "test-cache")

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Verify no retry happened (should be exactly 1 attempt)
			finalAttempts := atomic.LoadInt32(&attemptCount)
			if finalAttempts != 1 {
				t.Errorf("expected 1 attempt (no retry for %d), got %d", tt.statusCode, finalAttempts)
			}
		})
	}
}

func TestCachixClient_RequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all expected headers
		headers := map[string]string{
			"Authorization": "Bearer my-auth-token",
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			"User-Agent":    "terraform-provider-cachix/2.0.0",
		}

		for name, expected := range headers {
			if got := r.Header.Get(name); got != expected {
				t.Errorf("expected %s header %q, got %q", name, expected, got)
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(User{Username: "test"})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "my-auth-token", "2.0.0")
	_, err := client.GetUser(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCachixClient_GetCache_PrivateCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Cache{
			Name:              "private-cache",
			URI:               "https://private-cache.cachix.org",
			IsPublic:          false,
			PublicSigningKeys: []string{},
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.GetCache(context.Background(), "private-cache")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.IsPublic {
		t.Error("expected IsPublic to be false")
	}
	if len(cache.PublicSigningKeys) != 0 {
		t.Errorf("expected 0 signing keys for private cache, got %d", len(cache.PublicSigningKeys))
	}
}

func TestCachixClient_CreateCache_PrivateCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(User{
				ID:       12345,
				Username: "testuser",
				Email:    "testuser@example.com",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/cache/private-cache":
			// Verify request body has IsPublic: false
			var reqBody CreateCacheRequest
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			if reqBody.IsPublic {
				t.Error("expected IsPublic to be false in request")
			}
			if !reqBody.GenerateSigningKey {
				t.Error("expected GenerateSigningKey to be true")
			}
			// API returns empty body on success
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/cache/private-cache":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Cache{
				Name:     "private-cache",
				URI:      "https://private-cache.cachix.org",
				IsPublic: false,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.CreateCache(context.Background(), "private-cache", false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.IsPublic {
		t.Error("expected IsPublic to be false")
	}
}

func TestCachixClient_DeleteCache_WithOKStatus(t *testing.T) {
	// Some APIs return 200 OK instead of 204 No Content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	err := client.DeleteCache(context.Background(), "test-cache")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCachixClient_GetCache_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "you do not have permission to access this cache",
		})
	}))
	defer server.Close()

	client := NewCachixClient(server.URL, "test-token", "1.0.0")
	cache, err := client.GetCache(context.Background(), "restricted-cache")

	if cache != nil {
		t.Error("expected cache to be nil")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "you do not have permission to access this cache" {
		t.Errorf("expected permission error message, got '%s'", apiErr.Message)
	}
}
