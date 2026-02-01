// Copyright (c) takeokunn
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// DefaultRetryMax is the maximum number of retries for transient errors
	DefaultRetryMax = 3
	// DefaultRetryWaitMin is the minimum wait time between retries
	DefaultRetryWaitMin = 1 * time.Second
	// DefaultRetryWaitMax is the maximum wait time between retries
	DefaultRetryWaitMax = 30 * time.Second
)

// CachixClient is the HTTP client for interacting with the Cachix API.
type CachixClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
	userAgent  string
	retryMax   int
}

// Cache represents a Cachix binary cache.
type Cache struct {
	Name              string   `json:"name"`
	URI               string   `json:"uri"`
	IsPublic          bool     `json:"isPublic"`
	PublicSigningKeys []string `json:"publicSigningKeys"`
	CreatedAt         string   `json:"createdAt,omitempty"`
}

// User represents a Cachix user.
type User struct {
	ID               int    `json:"id"`
	Username         string `json:"githubUsername"`
	Email            string `json:"email,omitempty"`
	Fullname         string `json:"fullname,omitempty"`
	SubscriptionPlan string `json:"subscriptionPlan,omitempty"`
}

// CreateCacheRequest represents the request body for creating a cache.
type CreateCacheRequest struct {
	IsPublic           bool `json:"isPublic"`
	GenerateSigningKey bool `json:"generateSigningKey"`
	AccountID          int  `json:"accountID"`
}

// APIError represents an error response from the Cachix API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("cachix API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("cachix API error (status %d): %s", e.StatusCode, e.Body)
}

// NewCachixClient creates a new Cachix API client.
func NewCachixClient(baseURL, authToken, version string) *CachixClient {
	return &CachixClient{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: fmt.Sprintf("terraform-provider-cachix/%s", version),
		retryMax:  DefaultRetryMax,
	}
}

// doRequest performs an HTTP request with retry logic for transient errors.
func (c *CachixClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var lastErr error
	for attempt := 0; attempt <= c.retryMax; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff
			wait := c.calculateBackoff(attempt)
			tflog.Debug(ctx, "Retrying request after transient error", map[string]any{
				"attempt": attempt,
				"wait":    wait.String(),
				"url":     url,
			})
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(wait):
			}

			// Reset body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewBuffer(jsonBody)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		tflog.Debug(ctx, "Making Cachix API request", map[string]any{
			"method":  method,
			"url":     url,
			"attempt": attempt,
		})

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Check if we should retry based on status code
		if c.shouldRetry(resp.StatusCode) {
			lastErr = &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(respBody),
			}
			continue
		}

		return resp, respBody, nil
	}

	return nil, nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// shouldRetry determines if a request should be retried based on status code.
func (c *CachixClient) shouldRetry(statusCode int) bool {
	// Retry on server errors (5xx) and rate limiting (429)
	return statusCode >= 500 || statusCode == 429
}

// calculateBackoff calculates the backoff duration for a retry attempt.
func (c *CachixClient) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: min * 2^attempt, capped at max
	wait := float64(DefaultRetryWaitMin) * math.Pow(2, float64(attempt-1))
	if wait > float64(DefaultRetryWaitMax) {
		wait = float64(DefaultRetryWaitMax)
	}
	return time.Duration(wait)
}

// handleErrorResponse converts an HTTP response to an appropriate error.
func (c *CachixClient) handleErrorResponse(statusCode int, body []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       string(body),
	}

	// Try to parse error message from response
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			apiErr.Message = errResp.Error
		} else if errResp.Message != "" {
			apiErr.Message = errResp.Message
		}
	}

	switch statusCode {
	case http.StatusUnauthorized:
		if apiErr.Message == "" {
			apiErr.Message = "authentication failed: invalid or expired token"
		}
	case http.StatusForbidden:
		if apiErr.Message == "" {
			apiErr.Message = "access denied: insufficient permissions"
		}
	case http.StatusNotFound:
		if apiErr.Message == "" {
			apiErr.Message = "resource not found"
		}
	}

	return apiErr
}

// GetCache retrieves a cache by name.
func (c *CachixClient) GetCache(ctx context.Context, name string) (*Cache, error) {
	tflog.Debug(ctx, "Getting cache", map[string]any{"name": name})

	resp, body, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/cache/%s", name), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	var cache Cache
	if err := json.Unmarshal(body, &cache); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache response: %w", err)
	}

	tflog.Debug(ctx, "Got cache", map[string]any{
		"name":      cache.Name,
		"uri":       cache.URI,
		"is_public": cache.IsPublic,
	})

	return &cache, nil
}

// CreateCache creates a new cache with the given name and visibility.
func (c *CachixClient) CreateCache(ctx context.Context, name string, isPublic bool) (*Cache, error) {
	tflog.Debug(ctx, "Creating cache", map[string]any{
		"name":      name,
		"is_public": isPublic,
	})

	// First, get the current user to obtain the accountID
	user, err := c.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user for cache creation: %w", err)
	}

	reqBody := CreateCacheRequest{
		IsPublic:           isPublic,
		GenerateSigningKey: true,
		AccountID:          user.ID,
	}

	resp, body, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/cache/%s", name), reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	// The API returns an empty body on success, so fetch the cache details
	cache, err := c.GetCache(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("cache created but failed to fetch details: %w", err)
	}

	tflog.Info(ctx, "Created cache", map[string]any{
		"name":      cache.Name,
		"uri":       cache.URI,
		"is_public": cache.IsPublic,
	})

	return cache, nil
}

// DeleteCache deletes a cache by name.
func (c *CachixClient) DeleteCache(ctx context.Context, name string) error {
	tflog.Debug(ctx, "Deleting cache", map[string]any{"name": name})

	resp, body, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/cache/%s", name), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.handleErrorResponse(resp.StatusCode, body)
	}

	tflog.Info(ctx, "Deleted cache", map[string]any{"name": name})

	return nil
}

// IsNotFoundError checks if an error is a 404 Not Found error.
func IsNotFoundError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// GetUser retrieves the current authenticated user.
func (c *CachixClient) GetUser(ctx context.Context) (*User, error) {
	tflog.Debug(ctx, "Getting current user")

	resp, body, err := c.doRequest(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user response: %w", err)
	}

	tflog.Debug(ctx, "Got user", map[string]any{
		"username": user.Username,
	})

	return &user, nil
}
