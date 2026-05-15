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

func TestNewAppliesTimeout(t *testing.T) {
	c := New(3 * time.Second)
	if c.httpClient.Timeout != 3*time.Second {
		t.Fatalf("unexpected timeout %s", c.httpClient.Timeout)
	}
}
