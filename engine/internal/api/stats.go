// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
)

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "disconnected"
	if s.db != nil {
		dbStatus = "connected"
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"version":  s.version,
		"database": dbStatus,
	})
}

func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"status":          "healthy",
		"version":         s.version,
		"total_incidents": 0,
		"funcs_indexed":   0,
	}

	if s.db != nil {
		if dbStats, err := s.db.GetStats(r.Context()); err == nil {
			stats = dbStats
			stats["version"] = s.version
		}
	}

	writeJSON(w, http.StatusOK, stats)
}
