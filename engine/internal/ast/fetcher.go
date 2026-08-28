// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

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
	for strings.HasPrefix(cleanFile, "./") {
		cleanFile = strings.TrimPrefix(cleanFile, "./")
	}
	cleanRoot := strings.Trim(strings.TrimSpace(filepath.ToSlash(rootDir)), "/")
	for strings.HasPrefix(cleanRoot, "./") {
		cleanRoot = strings.TrimPrefix(cleanRoot, "./")
	}

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

// OnDemandFetcher fetches source files on-demand from GitHub via installation tokens.
type OnDemandFetcher struct {
	GitHubApp *gh.AppConfig
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
	return &OnDemandFetcher{}
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
	for strings.HasPrefix(cleanOrig, "./") {
		cleanOrig = strings.TrimPrefix(cleanOrig, "./")
	}

	candidatePaths := []string{normPath}
	if cleanOrig != normPath {
		candidatePaths = append(candidatePaths, cleanOrig)
	}

	var lastErr error
	for _, candidate := range candidatePaths {
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
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to fetch file from GitHub: %s", filePath)
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
