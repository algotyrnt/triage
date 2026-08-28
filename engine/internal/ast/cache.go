// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type ASTCache struct {
	store sync.Map
}

func NewASTCache() *ASTCache {
	return &ASTCache{}
}

// generateCacheKey creates a cache key using owner/repo@commit:file:line.
func generateCacheKey(owner, repo, commit, filePath string, line int) string {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(filePath), "/")
	return fmt.Sprintf("%s/%s@%s:%s:%d", owner, repo, commit, cleanPath, line)
}

// Get retrieves a cached AST snippet using the owner/repo@commit key.
func (c *ASTCache) Get(owner, repo, commit, filePath string, line int) (string, bool) {
	key := generateCacheKey(owner, repo, commit, filePath, line)
	val, ok := c.store.Load(key)
	if !ok {
		return "", false
	}
	snippet, ok := val.(string)
	return snippet, ok
}

// Set stores a cached AST snippet using the owner/repo@commit key.
func (c *ASTCache) Set(owner, repo, commit, filePath string, line int, snippet string) {
	if snippet == "" {
		return
	}
	key := generateCacheKey(owner, repo, commit, filePath, line)
	c.store.Store(key, snippet)
}
