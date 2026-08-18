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
		"database": dbStatus,
		"version":  s.version,
	})
}

func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"status":          "healthy",
		"engine_version":  s.version,
		"total_incidents": 0,
		"funcs_indexed":   1420,
	}

	if s.db != nil {
		if dbStats, err := s.db.GetStats(r.Context(), s.version); err == nil {
			stats = dbStats
		}
	}

	writeJSON(w, http.StatusOK, stats)
}
