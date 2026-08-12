// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileFetcher interface {
	FetchFile(ctx context.Context, owner, repo, commit, filePath string) ([]byte, error)
}

type OnDemandFetcher struct {
	HTTPClient *http.Client
}

func NewOnDemandFetcher() *OnDemandFetcher {
	return &OnDemandFetcher{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (f *OnDemandFetcher) FetchFile(ctx context.Context, owner, repo, commit, filePath string) ([]byte, error) {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(filePath), "/")

	// 1. If owner, repo, and commit are provided, attempt GitHub API / Raw URL fetch
	if owner != "" && repo != "" && commit != "" {
		content, err := f.fetchFromGitHub(ctx, owner, repo, commit, cleanPath)
		if err == nil && len(content) > 0 {
			return content, nil
		}
	}

	// 2. Fall back to reading from local workspace root context
	return f.fetchFromLocalWorkspace(cleanPath)
}

func (f *OnDemandFetcher) fetchFromGitHub(ctx context.Context, owner, repo, commit, cleanPath string) ([]byte, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, commit, cleanPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create github raw request: %w", err)
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5MB limit
		if err != nil {
			return nil, fmt.Errorf("failed to read raw content body: %w", err)
		}
		return data, nil
	}

	return nil, fmt.Errorf("github raw fetch returned HTTP %d for %s/%s@%s %s", resp.StatusCode, owner, repo, commit, cleanPath)
}

func (f *OnDemandFetcher) fetchFromLocalWorkspace(reqFile string) ([]byte, error) {
	root := os.Getenv("AST_WORKSPACE_ROOT")
	if root == "" {
		root = "."
	}

	cleanRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		cleanRoot = filepath.Clean(root)
	}
	if evalRoot, evalErr := filepath.EvalSymlinks(cleanRoot); evalErr == nil {
		cleanRoot = evalRoot
	}

	cleanReq := filepath.Clean(reqFile)
	var targetPath string

	if filepath.IsAbs(cleanReq) {
		rel, relErr := filepath.Rel(cleanRoot, cleanReq)
		if relErr == nil && !strings.HasPrefix(rel, "..") {
			targetPath = filepath.Join(cleanRoot, rel)
		} else {
			parts := strings.Split(filepath.ToSlash(cleanReq), "/")
			found := false
			for i := 0; i < len(parts); i++ {
				subPath := filepath.Join(parts[i:]...)
				candidate := filepath.Join(cleanRoot, subPath)
				if _, statErr := os.Stat(candidate); statErr == nil {
					targetPath = candidate
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("absolute path cannot be mapped to local workspace root: %s", reqFile)
			}
		}
	} else {
		targetPath = filepath.Join(cleanRoot, cleanReq)
	}

	if evalTarget, evalErr := filepath.EvalSymlinks(targetPath); evalErr == nil {
		targetPath = evalTarget
	}

	rel, err := filepath.Rel(cleanRoot, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("resolved path outside project root: %s", reqFile)
	}

	return os.ReadFile(targetPath)
}
