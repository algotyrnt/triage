// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gh "triage/engine/internal/github"
)

// FileFetcher defines the interface for fetching source file content.
type FileFetcher interface {
	FetchFile(ctx context.Context, owner, repo, commit, filePath string, rootDir ...string) ([]byte, error)
}

// NormalizeMonorepoPath combines rootDir and filePath if rootDir is specified
// and filePath does not already contain rootDir as a prefix. It also strips any
// developer host absolute paths leading up to rootDir.
func NormalizeMonorepoPath(filePath, rootDir string) string {
	cleanFile := strings.TrimPrefix(filepath.ToSlash(filePath), "/")
	cleanRoot := strings.Trim(strings.TrimSpace(filepath.ToSlash(rootDir)), "/")

	if cleanRoot != "" && cleanRoot != "." {
		// If cleanFile contains cleanRoot/ somewhere in an absolute path, extract from cleanRoot
		if idx := strings.Index(cleanFile, cleanRoot+"/"); idx != -1 {
			return cleanFile[idx:]
		}
		if cleanFile == cleanRoot {
			return cleanFile
		}
		if strings.HasPrefix(cleanFile, cleanRoot+"/") {
			return cleanFile
		}
		return cleanRoot + "/" + cleanFile
	}

	return cleanFile
}

// OnDemandFetcher fetches source files on-demand from GitHub (via installation tokens)
// or from the local workspace filesystem.
type OnDemandFetcher struct {
	HTTPClient *http.Client
	GitHubApp  *gh.AppConfig
	// GetInstallationID looks up the installation_id for a given owner/repo.
	// Injected by the caller (typically from db.GetInstallationForRepo).
	GetInstallationID func(ctx context.Context, owner, repo string) (int64, error)
}

// FetchResult contains the fetched file content and optional blob SHA for caching.
type FetchResult struct {
	Content []byte
	BlobSHA string // Git blob SHA for content-addressable caching
}

func NewOnDemandFetcher() *OnDemandFetcher {
	return &OnDemandFetcher{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchFile retrieves file content for AST parsing. Implements FileFetcher interface.
func (f *OnDemandFetcher) FetchFile(ctx context.Context, owner, repo, commit, filePath string, rootDir ...string) ([]byte, error) {
	result, err := f.FetchFileWithMeta(ctx, owner, repo, commit, filePath, rootDir...)
	if err != nil {
		return nil, err
	}
	return result.Content, nil
}

// FetchFileWithMeta retrieves file content along with the blob SHA for caching.
func (f *OnDemandFetcher) FetchFileWithMeta(ctx context.Context, owner, repo, commit, filePath string, rootDir ...string) (*FetchResult, error) {
	rd := ""
	if len(rootDir) > 0 {
		rd = rootDir[0]
	}

	normPath := NormalizeMonorepoPath(filePath, rd)
	cleanOrig := strings.TrimPrefix(filepath.ToSlash(filePath), "/")

	candidatePaths := []string{normPath}
	if cleanOrig != normPath {
		candidatePaths = append(candidatePaths, cleanOrig)
	}

	var lastErr error
	for _, candidate := range candidatePaths {
		// 1. If GitHub App is configured, try installation token-based fetch (Contents API)
		if f.GitHubApp != nil && f.GetInstallationID != nil && owner != "" && repo != "" {
			result, err := f.fetchFromGitHubApp(ctx, owner, repo, commit, candidate)
			if err == nil && len(result.Content) > 0 {
				return result, nil
			}
			if err != nil {
				lastErr = err
				slog.Debug("GitHub App fetch attempt failed", "owner", owner, "repo", repo, "commit", commit, "candidate", candidate, "error", err)
			}
		}

		// 2. Fallback: raw.githubusercontent.com with GITHUB_TOKEN (legacy)
		if owner != "" && repo != "" {
			rawCommit := commit
			if rawCommit == "" {
				rawCommit = "HEAD"
			}
			content, err := f.fetchFromGitHubRaw(ctx, owner, repo, rawCommit, candidate)
			if err == nil && len(content) > 0 {
				return &FetchResult{Content: content}, nil
			}
			if err != nil {
				lastErr = err
			}
		}

		// 3. Final fallback: local workspace filesystem
		content, err := f.fetchFromLocalWorkspace(candidate)
		if err == nil && len(content) > 0 {
			return &FetchResult{Content: content}, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to fetch file: %s", filePath)
}

// fetchFromGitHubApp uses the GitHub App installation token to fetch file content
// via the Contents API. Returns both content and blob SHA for cache keying.
func (f *OnDemandFetcher) fetchFromGitHubApp(ctx context.Context, owner, repo, commit, cleanPath string) (*FetchResult, error) {
	installID, err := f.GetInstallationID(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("no installation found for %s/%s: %w", owner, repo, err)
	}

	content, blobSHA, err := f.GitHubApp.FetchFileContent(ctx, installID, owner, repo, commit, cleanPath)
	if err != nil {
		return nil, err
	}

	return &FetchResult{
		Content: content,
		BlobSHA: blobSHA,
	}, nil
}

// fetchFromGitHubRaw uses raw.githubusercontent.com with a personal GITHUB_TOKEN.
// This is the legacy approach — no blob SHA is available.
func (f *OnDemandFetcher) fetchFromGitHubRaw(ctx context.Context, owner, repo, commit, cleanPath string) ([]byte, error) {
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
