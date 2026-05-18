// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
