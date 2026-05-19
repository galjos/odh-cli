// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/galjos/odh-cli/internal/cache"
)

func TestGetSendsUserAgentAndReturnsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Fatalf("unexpected user agent %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := NewWithHTTPClient(server.Client(), "test-agent")
	resp, err := c.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Fatalf("unexpected body %q", string(resp.Body))
	}
}

func TestGetReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	c := NewWithHTTPClient(server.Client(), "test-agent")
	_, err := c.Get(context.Background(), server.URL)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T %[1]v", err)
	}
	if httpErr.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status %d", httpErr.StatusCode)
	}
	if !strings.Contains(httpErr.Error(), "not found") {
		t.Fatalf("error did not include response body: %v", httpErr)
	}
}

func TestGetRetriesTransientStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := NewWithHTTPClient(server.Client(), "test-agent").WithRetryPolicy(2, 0)
	resp, err := c.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Fatalf("unexpected body %q", string(resp.Body))
	}
}

func TestGetDoesNotRetryClientError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	c := NewWithHTTPClient(server.Client(), "test-agent").WithRetryPolicy(3, 0)
	_, err := c.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestGetCachedUsesFreshCache(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		_, _ = io.WriteString(w, `{"cached":true}`)
	}))
	defer server.Close()

	c := NewWithHTTPClient(server.Client(), "test-agent")
	c.cacheStore = cache.New(t.TempDir())
	first, err := c.GetCached(context.Background(), server.URL, time.Hour)
	if err != nil {
		t.Fatalf("first GetCached returned error: %v", err)
	}
	second, err := c.GetCached(context.Background(), server.URL, time.Hour)
	if err != nil {
		t.Fatalf("second GetCached returned error: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	if first.FromCache {
		t.Fatal("first response unexpectedly came from cache")
	}
	if !second.FromCache {
		t.Fatal("second response did not come from cache")
	}
}

func TestGetWithLimitRejectsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("abcdef"))
	}))
	defer server.Close()

	c := NewWithHTTPClient(server.Client(), "test-agent")
	resp, err := c.GetWithLimit(context.Background(), server.URL, 5)
	if err == nil {
		t.Fatal("expected oversized response error")
	}
	if !strings.Contains(err.Error(), "exceeded 5 byte limit") {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Body) != "abcde" {
		t.Fatalf("unexpected truncated body %q", string(resp.Body))
	}
}

func TestNewAppliesTimeout(t *testing.T) {
	c := New(3 * time.Second)
	if c.httpClient.Timeout != 3*time.Second {
		t.Fatalf("unexpected timeout %s", c.httpClient.Timeout)
	}
}

func TestWithTimeoutCopiesClient(t *testing.T) {
	base := NewWithHTTPClient(&http.Client{Timeout: 3 * time.Second}, "test-agent")
	copied := base.WithTimeout(2 * time.Minute)
	if copied == base {
		t.Fatal("WithTimeout returned the same client")
	}
	if base.httpClient.Timeout != 3*time.Second {
		t.Fatalf("base timeout changed to %s", base.httpClient.Timeout)
	}
	if copied.httpClient.Timeout != 2*time.Minute {
		t.Fatalf("unexpected copied timeout %s", copied.httpClient.Timeout)
	}
	if copied.userAgent != "test-agent" {
		t.Fatalf("unexpected user agent %q", copied.userAgent)
	}
}
