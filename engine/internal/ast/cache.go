// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type CachedFunction struct {
	FunctionName string `json:"function_name"`
	FilePath     string `json:"file_path"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	Snippet      string `json:"snippet"`
}

type fileFunctionCache struct {
	mu        sync.RWMutex
	functions []CachedFunction
}

type ASTCache struct {
	exactStore sync.Map // direct key -> snippet for fast direct lookups
	fileStore  sync.Map // owner/repo@commit:file -> *fileFunctionCache
}

func NewASTCache() *ASTCache {
	return &ASTCache{}
}

func generateCacheKey(owner, repo, commit, filePath string, line int) string {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(filePath), "/")
	return fmt.Sprintf("%s/%s@%s:%s:%d", owner, repo, commit, cleanPath, line)
}

func generateFileKey(owner, repo, commit, filePath string) string {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(filePath), "/")
	return fmt.Sprintf("%s/%s@%s:%s", owner, repo, commit, cleanPath)
}

// Get retrieves a cached AST snippet using exact line key or function range lookup.
func (c *ASTCache) Get(owner, repo, commit, filePath string, line int) (string, bool) {
	// 1. Direct exact-line lookup
	key := generateCacheKey(owner, repo, commit, filePath, line)
	if val, ok := c.exactStore.Load(key); ok {
		if snippet, ok := val.(string); ok && snippet != "" {
			return snippet, true
		}
	}

	// 2. Function-wise range lookup across the file
	fileKey := generateFileKey(owner, repo, commit, filePath)
	if val, ok := c.fileStore.Load(fileKey); ok {
		fc := val.(*fileFunctionCache)
		fc.mu.RLock()
		defer fc.mu.RUnlock()
		for _, fn := range fc.functions {
			if line >= fn.StartLine && line <= fn.EndLine {
				// Memoize in exactStore for instant O(1) next lookup
				c.exactStore.Store(key, fn.Snippet)
				return fn.Snippet, true
			}
		}
	}

	return "", false
}

// Set stores a cached AST snippet for an exact line.
func (c *ASTCache) Set(owner, repo, commit, filePath string, line int, snippet string) {
	if snippet == "" {
		return
	}
	key := generateCacheKey(owner, repo, commit, filePath, line)
	c.exactStore.Store(key, snippet)
}

// SetFunction stores a function with its [startLine, endLine] boundary for function-wise range caching.
func (c *ASTCache) SetFunction(owner, repo, commit, filePath, fnName string, startLine, endLine int, snippet string) {
	if snippet == "" {
		return
	}

	// Store in exactStore for startLine
	c.Set(owner, repo, commit, filePath, startLine, snippet)

	fileKey := generateFileKey(owner, repo, commit, filePath)
	val, _ := c.fileStore.LoadOrStore(fileKey, &fileFunctionCache{})
	fc := val.(*fileFunctionCache)

	fc.mu.Lock()
	defer fc.mu.Unlock()

	for i, fn := range fc.functions {
		if fn.FunctionName == fnName && fn.StartLine == startLine {
			fc.functions[i].Snippet = snippet
			fc.functions[i].EndLine = endLine
			return
		}
	}

	fc.functions = append(fc.functions, CachedFunction{
		FunctionName: fnName,
		FilePath:     filePath,
		StartLine:    startLine,
		EndLine:      endLine,
		Snippet:      snippet,
	})
}
