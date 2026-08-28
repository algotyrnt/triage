// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HandleTeamMembers lists or removes team members.
func (s *Server) HandleTeamMembers(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		users, err := s.db.ListUsers(r.Context())
		if err != nil {
			http.Error(w, "Failed to list members", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"members": users,
		})

	case http.MethodDelete:
		// Owner-only operation
		claims := s.getUserClaims(r)
		if claims == nil || claims.Role != "Owner" {
			http.Error(w, "Forbidden: Only Owners can remove team members", http.StatusForbidden)
			return
		}

		targetID := r.URL.Query().Get("id")
		if targetID == "" {
			http.Error(w, "Missing member id", http.StatusBadRequest)
			return
		}

		if claims.UserID == targetID {
			http.Error(w, "Cannot remove yourself", http.StatusBadRequest)
			return
		}

		if err := s.db.DeleteUser(r.Context(), targetID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"id":     targetID,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTeamMemberRole updates a team member's role.
func (s *Server) HandleTeamMemberRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.db == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	claims := s.getUserClaims(r)
	if claims == nil || (claims.Role != "Owner" && claims.Role != "Admin") {
		http.Error(w, "Forbidden: Only Owners and Admins can update roles", http.StatusForbidden)
		return
	}

	var req struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Role == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Non-owners cannot promote to Owner or change an Owner's role
	if claims.Role != "Owner" && req.Role == "Owner" {
		http.Error(w, "Forbidden: Only Owners can assign the Owner role", http.StatusForbidden)
		return
	}

	if err := s.db.UpdateUserRole(r.Context(), req.ID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"id":     req.ID,
		"role":   req.Role,
	})
}

// HandleTeamInvites manages invitations for GitHub usernames.
func (s *Server) HandleTeamInvites(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not connected", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		invites, err := s.db.ListInvitations(r.Context())
		if err != nil {
			http.Error(w, "Failed to list invitations", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"invitations": invites,
		})

	case http.MethodPost:
		claims := s.getUserClaims(r)
		if claims == nil || (claims.Role != "Owner" && claims.Role != "Admin") {
			http.Error(w, "Forbidden: Only Owners and Admins can send invites", http.StatusForbidden)
			return
		}

		var req struct {
			GitHubUsername string `json:"github_username"`
			Role           string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.GitHubUsername) == "" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		cleanUsername := strings.TrimPrefix(strings.TrimSpace(req.GitHubUsername), "@")
		role := req.Role
		if role == "" {
			role = "Developer"
		}

		// Non-owners cannot invite as Owner
		if role == "Owner" {
			role = "Admin"
		}

		invitedBy := claims.UserID

		inv, err := s.db.CreateInvitation(r.Context(), cleanUsername, role, invitedBy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"status":     "created",
			"invitation": inv,
		})

	case http.MethodDelete:
		claims := s.getUserClaims(r)
		if claims == nil || (claims.Role != "Owner" && claims.Role != "Admin") {
			http.Error(w, "Forbidden: Only Owners and Admins can cancel invites", http.StatusForbidden)
			return
		}

		target := r.URL.Query().Get("id")
		if target == "" {
			http.Error(w, "Missing invitation id", http.StatusBadRequest)
			return
		}

		if err := s.db.DeleteInvitation(r.Context(), target); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"id":     target,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
