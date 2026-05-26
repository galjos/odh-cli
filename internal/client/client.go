// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/cache"
)

const (
	defaultUserAgent  = "odh-cli/0.2"
	defaultMaxRetries = 3
	defaultBaseDelay  = 1 * time.Second
)

// Client is a small context-aware HTTP client for Open Data Hub API calls.
type Client struct {
	httpClient *http.Client
	userAgent  string
	maxRetries int
	baseDelay  time.Duration
	cacheStore *cache.Store
}

// Response contains the raw response body and selected metadata.
type Response struct {
	URL         string
	StatusCode  int
	ContentType string
	Body        []byte
	FromCache   bool
}

// HTTPError captures a non-2xx response.
type HTTPError struct {
	URL         string
	StatusCode  int
	ContentType string
	Body        string
}

func (e *HTTPError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("GET %s failed with HTTP %d", e.URL, e.StatusCode)
	}
	if len(body) > 300 {
		body = body[:300] + "..."
	}
	return fmt.Sprintf("GET %s failed with HTTP %d: %s", e.URL, e.StatusCode, body)
}

// New creates a client with the provided timeout and default retries.
func New(timeout time.Duration) *Client {
	return NewWithHTTPClient(&http.Client{Timeout: timeout}, defaultUserAgent)
}

// NewWithHTTPClient creates a client around an injected HTTP client.
func NewWithHTTPClient(httpClient *http.Client, userAgent string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = defaultUserAgent
	}
	return &Client{
		httpClient: httpClient,
		userAgent:  userAgent,
		maxRetries: defaultMaxRetries,
		baseDelay:  defaultBaseDelay,
		cacheStore: cache.New(cache.DefaultDir()),
	}
}

// WithTransport returns a copy of the client using a specific RoundTripper.
func (c *Client) WithTransport(transport http.RoundTripper) *Client {
	if c == nil {
		return NewWithHTTPClient(&http.Client{Transport: transport, Timeout: 30 * time.Second}, "")
	}
	httpClient := *c.httpClient
	httpClient.Transport = transport
	return &Client{
		httpClient: &httpClient,
		userAgent:  c.userAgent,
		maxRetries: c.maxRetries,
		baseDelay:  c.baseDelay,
		cacheStore: c.cacheStore,
	}
}

// WithTimeout returns a copy of the client with a different HTTP timeout.
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	if c == nil {
		return New(timeout)
	}
	if timeout <= 0 {
		return c
	}
	httpClient := *c.httpClient
	httpClient.Timeout = timeout
	return &Client{
		httpClient: &httpClient,
		userAgent:  c.userAgent,
		maxRetries: c.maxRetries,
		baseDelay:  c.baseDelay,
		cacheStore: c.cacheStore,
	}
}

// WithRetryPolicy returns a copy of the client with custom retry behavior.
func (c *Client) WithRetryPolicy(maxRetries int, baseDelay time.Duration) *Client {
	if maxRetries < 0 {
		maxRetries = 0
	}
	if baseDelay < 0 {
		baseDelay = 0
	}
	if c == nil {
		c = New(30 * time.Second)
	}
	return &Client{
		httpClient: c.httpClient,
		userAgent:  c.userAgent,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		cacheStore: c.cacheStore,
	}
}

// Get performs an HTTP GET and returns the response body for 2xx responses.
func (c *Client) Get(ctx context.Context, url string) (Response, error) {
	return c.GetWithLimit(ctx, url, 50*1024*1024)
}

// GetCached performs an HTTP GET with local file caching.
func (c *Client) GetCached(ctx context.Context, url string, ttl time.Duration) (Response, error) {
	if c.cacheStore != nil {
		if data, ok := c.cacheStore.Get(url, ttl); ok {
			return Response{
				URL:        url,
				StatusCode: 200,
				Body:       data,
				FromCache:  true,
			}, nil
		}
	}

	resp, err := c.Get(ctx, url)
	if err == nil && c.cacheStore != nil {
		_ = c.cacheStore.Set(url, resp.Body)
	}
	return resp, nil
}

// GetWithLimit performs an HTTP GET and reads at most limitBytes bytes.
func (c *Client) GetWithLimit(ctx context.Context, url string, limitBytes int64) (Response, error) {
	if strings.TrimSpace(url) == "" {
		return Response{}, errors.New("url is required")
	}
	if limitBytes < 1 {
		return Response{}, errors.New("limitBytes must be greater than zero")
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.baseDelay * (1 << (uint(attempt) - 1))
			select {
			case <-ctx.Done():
				return Response{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return Response{}, err
		}
		req.Header.Set("Accept", "application/json, application/yaml, text/yaml, */*")
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return Response{}, ctx.Err()
			}
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, limitBytes+1))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if int64(len(body)) > limitBytes {
			return Response{
				URL:         url,
				StatusCode:  resp.StatusCode,
				ContentType: resp.Header.Get("Content-Type"),
				Body:        body[:limitBytes],
			}, fmt.Errorf("GET %s response exceeded %d byte limit", url, limitBytes)
		}

		result := Response{
			URL:         url,
			StatusCode:  resp.StatusCode,
			ContentType: resp.Header.Get("Content-Type"),
			Body:        body,
		}

		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return result, nil
		}

		lastErr = &HTTPError{
			URL:         url,
			StatusCode:  resp.StatusCode,
			ContentType: result.ContentType,
			Body:        string(body),
		}

		// Only retry on 429 (Rate Limit) or 5xx (Server Error)
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return result, lastErr
		}
	}

	return Response{}, lastErr
}
