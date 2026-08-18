// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"triage/engine/internal/db"
)

type DetectedModule struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	IsRoot bool   `json:"is_root"`
}

func (s *Server) HandleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Repo          string `json:"repo"`
			Owner         string `json:"owner"`
			RootDir       string `json:"root_dir"`
			ServicePath   string `json:"service_path"`
			OwnerUsername string `json:"owner_username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}
		if req.Repo == "" {
			http.Error(w, "Field 'repo' is required", http.StatusBadRequest)
			return
		}
		owner := req.Owner
		repoName := req.Repo
		if strings.Contains(req.Repo, "/") {
			parts := strings.Split(req.Repo, "/")
			owner = parts[0]
			repoName = parts[1]
		}
		if owner == "" {
			owner = "algotyrnt"
		}
		rootDir := req.RootDir
		if rootDir == "" {
			rootDir = req.ServicePath
		}
		rootDir = strings.Trim(strings.TrimSpace(rootDir), "/")

		apiKey := fmt.Sprintf("tr_live_%s_%d", repoName, time.Now().UnixNano())
		keyMasked := fmt.Sprintf("tr_live_...%s", apiKey[len(apiKey)-4:])
		if s.db != nil {
			k, _, err := s.db.CreateProject(r.Context(), owner, repoName, rootDir, req.OwnerUsername)
			if err == nil && k != "" {
				apiKey = k
				if len(k) >= 4 {
					keyMasked = fmt.Sprintf("tr_live_...%s", k[len(k)-4:])
				}
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"repo":       fmt.Sprintf("%s/%s", owner, repoName),
			"root_dir":   rootDir,
			"api_key":    apiKey,
			"key_masked": keyMasked,
		})
		return
	}

	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"projects": []db.Repository{}})
		return
	}

	projects, err := s.db.GetProjects(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch projects: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"projects": projects})
}

func (s *Server) HandleProjectKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		owner := r.URL.Query().Get("owner")
		repo := r.URL.Query().Get("repo")
		rootDir := r.URL.Query().Get("root_dir")

		if strings.Contains(repo, "/") {
			parts := strings.Split(repo, "/")
			owner = parts[0]
			repo = parts[1]
		}

		if s.db == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{"keys": []db.APIKeyRecord{}})
			return
		}

		keys, err := s.db.GetAPIKeys(r.Context(), owner, repo, rootDir)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch API keys: %v", err), http.StatusInternalServerError)
			return
		}
		if keys == nil {
			keys = []db.APIKeyRecord{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"keys": keys})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Repo    string `json:"repo"`
			Owner   string `json:"owner"`
			RootDir string `json:"root_dir"`
			Name    string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
			return
		}
		owner := req.Owner
		repoName := req.Repo
		if strings.Contains(req.Repo, "/") {
			parts := strings.Split(req.Repo, "/")
			owner = parts[0]
			repoName = parts[1]
		}
		if owner == "" {
			owner = "algotyrnt"
		}
		if repoName == "" {
			repoName = "triage"
		}

		if s.db == nil {
			rawKey := fmt.Sprintf("tr_live_%s_%d", repoName, time.Now().UnixNano())
			masked := fmt.Sprintf("tr_live_...%s", rawKey[len(rawKey)-4:])
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"key": map[string]interface{}{
					"id":         fmt.Sprintf("key_%d", time.Now().UnixNano()),
					"name":       req.Name,
					"raw_key":    rawKey,
					"key_masked": masked,
					"created_at": time.Now().UTC().Format(time.RFC3339),
					"status":     "ACTIVE",
				},
			})
			return
		}

		keyRecord, err := s.db.CreateAPIKey(r.Context(), owner, repoName, req.RootDir, req.Name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create API key: %v", err), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"key":     keyRecord,
		})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) HandleRevokeProjectKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		KeyID string `json:"key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.KeyID == "" {
		req.KeyID = r.URL.Query().Get("key_id")
	}

	if req.KeyID == "" {
		http.Error(w, "Field 'key_id' is required", http.StatusBadRequest)
		return
	}

	if s.db != nil {
		if err := s.db.RevokeAPIKey(r.Context(), req.KeyID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to revoke API key: %v", err), http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) HandleDetectModules(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner = parts[0]
			repo = parts[1]
		}
	}

	modules := []DetectedModule{
		{Path: "", Name: "Repository Root (/)", IsRoot: true},
	}
	seen := map[string]bool{"": true}

	// 1. Try detecting via GitHub App if configured
	if s.githubApp != nil && s.db != nil && owner != "" && repo != "" {
		installID, err := s.db.GetInstallationForRepo(r.Context(), owner, repo)
		if err != nil || installID == 0 {
			if inst, iErr := s.db.GetInstallation(r.Context()); iErr == nil && inst != nil {
				installID = inst.InstallationID
			}
		}

		if installID > 0 {
			token, tokenErr := s.githubApp.GetInstallationToken(r.Context(), installID)
			if tokenErr == nil {
				// Query GitHub repo tree recursively
				branchesToTry := []string{"HEAD", "main", "master"}
				for _, b := range branchesToTry {
					treesURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, b)
					treeReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, treesURL, nil)
					treeReq.Header.Set("Authorization", "Bearer "+token)
					treeReq.Header.Set("Accept", "application/vnd.github+json")

					client := &http.Client{Timeout: 10 * time.Second}
					resp, err := client.Do(treeReq)
					if err == nil && resp != nil {
						if resp.StatusCode == http.StatusOK {
							var treeData struct {
								Tree []struct {
									Path string `json:"path"`
									Type string `json:"type"`
								} `json:"tree"`
							}
							if jsonErr := json.NewDecoder(resp.Body).Decode(&treeData); jsonErr == nil {
								for _, item := range treeData.Tree {
									if item.Type == "blob" && strings.HasSuffix(item.Path, "go.mod") {
										dir := filepath.ToSlash(filepath.Dir(item.Path))
										if dir == "." {
											dir = ""
										}
										if !seen[dir] {
											seen[dir] = true
											displayName := fmt.Sprintf("%s/ (Go Module)", dir)
											if dir == "" {
												displayName = "Repository Root (/)"
											}
											modules = append(modules, DetectedModule{
												Path:   dir,
												Name:   displayName,
												IsRoot: dir == "",
											})
										}
									}
								}
							}
							_ = resp.Body.Close()
							break
						}
						_ = resp.Body.Close()
					}
				}
			}
		}
	}

	// 2. Scan local workspace if present
	wsRoot := os.Getenv("TRIAGE_WORKSPACE_ROOT")
	if wsRoot == "" {
		wsRoot = os.Getenv("AST_WORKSPACE_ROOT")
	}
	if wsRoot == "" {
		wsRoot, _ = os.Getwd()
	}

	if wsRoot != "" {
		_ = filepath.Walk(wsRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if info.Name() == "go.mod" {
				rel, relErr := filepath.Rel(wsRoot, path)
				if relErr == nil {
					dir := filepath.ToSlash(filepath.Dir(rel))
					if dir == "." {
						dir = ""
					}
					if !seen[dir] {
						seen[dir] = true
						displayName := fmt.Sprintf("%s/ (Go Module)", dir)
						if dir == "" {
							displayName = "Repository Root (/)"
						}
						modules = append(modules, DetectedModule{
							Path:   dir,
							Name:   displayName,
							IsRoot: dir == "",
						})
					}
				}
			}
			return nil
		})
	}

	// Sort modules: Root first, then alphabetically
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].IsRoot != modules[j].IsRoot {
			return modules[i].IsRoot
		}
		return modules[i].Path < modules[j].Path
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"modules": modules,
	})
}
