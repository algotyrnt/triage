// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"triage/engine/internal/ast"
)

type IndexRequest struct {
	APIKey        string `json:"api_key,omitempty"`
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	Commit        string `json:"commit"`
	WorkspacePath string `json:"workspace_path"`
	RootDir       string `json:"root_dir,omitempty"`
	RootPath      string `json:"root_path,omitempty"`
}

func (s *Server) HandleASTIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = r.Header.Get("X-Triage-API-Key")
	}
	if apiKey != "" && !s.IsValidAPIKey(r.Context(), apiKey) {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error":  "unauthorized: invalid API key",
		})
		return
	}

	if s.astManager == nil {
		http.Error(w, "Database AST Manager uninitialized", http.StatusInternalServerError)
		return
	}

	workspacePath := req.WorkspacePath
	if workspacePath == "" {
		workspacePath = "."
	}

	rootDir := req.RootDir
	if rootDir == "" {
		rootDir = req.RootPath
	}

	count, err := s.astManager.IndexRepositoryAST(r.Context(), req.Owner, req.Repo, req.Commit, workspacePath, rootDir)
	if err != nil {
		slog.Error("AST indexing failed", "error", err, "owner", req.Owner, "repo", req.Repo)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	slog.Info("AST indexing completed", "indexed_count", count, "owner", req.Owner, "repo", req.Repo)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"indexed_count": count,
	})
}

// HandleASTTree returns the Go AST file structure, package info, and indexed functions for a repository.
func (s *Server) HandleASTTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	rootDir := r.URL.Query().Get("root_dir")

	// Allow passing full slug in repo or owner (e.g. repo=algotyrnt/triage)
	if strings.Contains(repo, "/") {
		parts := strings.SplitN(repo, "/", 2)
		owner = parts[0]
		repo = parts[1]
	}

	var files []ast.ASTFileItem
	var err error

	if s.astManager != nil && owner != "" && repo != "" {
		files, err = s.astManager.ListASTFiles(r.Context(), owner, repo, rootDir)
		if err != nil {
			slog.Warn("failed to list AST files from database", "error", err, "owner", owner, "repo", repo)
		}
	}
	if files == nil {
		files = []ast.ASTFileItem{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"owner":    owner,
		"repo":     repo,
		"root_dir": rootDir,
		"total":    len(files),
		"files":    files,
	})
}
