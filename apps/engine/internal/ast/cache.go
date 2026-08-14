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

// generateCacheKey creates a legacy cache key using owner/repo@commit:file:line.
// Used as fallback when blob SHA is not available.
func generateCacheKey(owner, repo, commit, filePath string, line int) string {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(filePath), "/")
	return fmt.Sprintf("%s/%s@%s:%s:%d", owner, repo, commit, cleanPath, line)
}

// generateBlobCacheKey creates a cache key using the Git blob SHA.
// Blob SHA is a content hash — identical file content across commits/branches
// produces the same key, enabling cross-commit deduplication.
func generateBlobCacheKey(blobSHA string, line int) string {
	return fmt.Sprintf("blob:%s:%d", blobSHA, line)
}

// Get retrieves a cached AST snippet using the legacy owner/repo@commit key.
func (c *ASTCache) Get(owner, repo, commit, filePath string, line int) (string, bool) {
	key := generateCacheKey(owner, repo, commit, filePath, line)
	val, ok := c.store.Load(key)
	if !ok {
		return "", false
	}
	snippet, ok := val.(string)
	return snippet, ok
}

// Set stores a cached AST snippet using the legacy owner/repo@commit key.
func (c *ASTCache) Set(owner, repo, commit, filePath string, line int, snippet string) {
	if snippet == "" {
		return
	}
	key := generateCacheKey(owner, repo, commit, filePath, line)
	c.store.Store(key, snippet)
}

// GetByBlobSHA retrieves a cached AST snippet using the Git blob SHA.
func (c *ASTCache) GetByBlobSHA(blobSHA string, line int) (string, bool) {
	if blobSHA == "" {
		return "", false
	}
	key := generateBlobCacheKey(blobSHA, line)
	val, ok := c.store.Load(key)
	if !ok {
		return "", false
	}
	snippet, ok := val.(string)
	return snippet, ok
}

// SetByBlobSHA stores a cached AST snippet using the Git blob SHA.
func (c *ASTCache) SetByBlobSHA(blobSHA string, line int, snippet string) {
	if blobSHA == "" || snippet == "" {
		return
	}
	key := generateBlobCacheKey(blobSHA, line)
	c.store.Store(key, snippet)
}
