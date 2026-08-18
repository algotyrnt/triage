// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
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
	if !s.IsValidAPIKey(r.Context(), apiKey) {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error":  "unauthorized: missing or invalid API key",
		})
		return
	}

	if s.astManager == nil {
		http.Error(w, "Database AST Manager uninitialized", http.StatusInternalServerError)
		return
	}

	workspacePath := req.WorkspacePath
	if workspacePath != "" {
		resolvedPath, valErr := s.ValidateAndResolveFilePath(workspacePath)
		if valErr != nil {
			slog.Warn("workspacePath validation failed", "error", valErr, "workspace_path", workspacePath)
			workspacePath = os.Getenv("AST_WORKSPACE_ROOT")
		} else {
			workspacePath = resolvedPath
		}
	}
	if workspacePath == "" {
		workspacePath = os.Getenv("AST_WORKSPACE_ROOT")
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
