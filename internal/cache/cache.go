// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// Store is a simple file-based cache.
type Store struct {
	Dir string
}

// New creates a store in the provided directory.
func New(dir string) *Store {
	return &Store{Dir: dir}
}

// DefaultDir returns the default user cache directory for odh-cli.
func DefaultDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "odh-cli", "http")
}

// Get retrieves a cached response if it exists and is not expired.
func (s *Store) Get(key string, ttl time.Duration) ([]byte, bool) {
	if ttl <= 0 {
		return nil, false
	}
	path := s.path(key)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, false
	}
	if time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Set stores a response in the cache.
func (s *Store) Set(key string, data []byte) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path(key), data, 0o644)
}

func (s *Store) path(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(s.Dir, hex.EncodeToString(hash[:]))
}
