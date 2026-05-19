// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package cache

import (
	"os"
	"testing"
	"time"
)

func TestStoreGetReturnsFreshEntry(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Set("https://example.test/a", []byte("payload")); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, ok := store.Get("https://example.test/a", time.Hour)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != "payload" {
		t.Fatalf("unexpected payload %q", string(got))
	}
}

func TestStoreGetRejectsExpiredEntry(t *testing.T) {
	store := New(t.TempDir())
	key := "https://example.test/a"
	if err := store.Set(key, []byte("payload")); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(store.path(key), old, old); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}

	if _, ok := store.Get(key, time.Hour); ok {
		t.Fatal("expected expired cache miss")
	}
}
