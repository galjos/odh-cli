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
)

const defaultUserAgent = "odh-cli/0.1"

// Client is a small context-aware HTTP client for Open Data Hub API calls.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// Response contains the raw response body and selected metadata.
type Response struct {
	URL         string
	StatusCode  int
	ContentType string
	Body        []byte
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

// New creates a client with the provided timeout.
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
	return &Client{httpClient: httpClient, userAgent: userAgent}
}

// Get performs an HTTP GET and returns the response body for 2xx responses.
func (c *Client) Get(ctx context.Context, url string) (Response, error) {
	if strings.TrimSpace(url) == "" {
		return Response{}, errors.New("url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json, application/yaml, text/yaml, */*")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024))
	if readErr != nil {
		return Response{}, readErr
	}

	result := Response{
		URL:         url,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return result, &HTTPError{
			URL:         url,
			StatusCode:  resp.StatusCode,
			ContentType: result.ContentType,
			Body:        string(body),
		}
	}
	return result, nil
}
