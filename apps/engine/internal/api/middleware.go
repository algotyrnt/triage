// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

// isLocalhostOrigin checks if an origin URI corresponds to a local loopback address.
func isLocalhostOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "0.0.0.0" || hostname == "::1"
}

// resolveAllowedOrigin dynamically verifies if an incoming browser Origin header is permitted.
func (s *Server) resolveAllowedOrigin(ctx context.Context, r *http.Request) string {
	if r == nil {
		return ""
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return ""
	}

	var configuredURL string
	if s.configStore != nil {
		configuredURL = s.configStore.GetInstanceURL(ctx)
	}

	// 1. Initial Setup mode: when instance_url is not yet stored in PostgreSQL, permit setup origin
	if configuredURL == "" {
		return origin
	}

	// 2. Exact match with configured instance URL
	if strings.EqualFold(strings.TrimRight(origin, "/"), strings.TrimRight(configuredURL, "/")) {
		return origin
	}

	// 3. Localhost loopback origins (permitted for local development & debugging)
	if isLocalhostOrigin(origin) {
		return origin
	}

	// 4. Untrusted cross-origin request
	return ""
}

// withMiddleware wraps a handler with origin-restricted CORS, panic recovery, and logging.
func (s *Server) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Panic recovery
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("recovered from panic in HTTP handler",
					"panic", fmt.Sprintf("%v", rec),
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
			}
		}()

		// Dynamic CORS headers
		if allowedOrigin := s.resolveAllowedOrigin(r.Context(), r); allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Triage-API-Key")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		start := time.Now()
		next(w, r)
		duration := time.Since(start)

		if duration > 500*time.Millisecond {
			slog.Warn("slow HTTP request detected",
				"method", r.Method,
				"path", r.URL.Path,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
			)
		} else {
			slog.Debug("HTTP request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}
}

// IsValidAPIKey checks if the provided API key is valid against the database.
func (s *Server) IsValidAPIKey(ctx context.Context, key string) bool {
	if key == "" || s.db == nil {
		return false
	}

	return s.db.VerifyAPIKey(ctx, key)
}

// ExtractBearerOrAPIKey retrieves credentials from headers or query parameters.
func ExtractBearerOrAPIKey(r *http.Request) string {
	apiKey := r.Header.Get("X-Triage-API-Key")
	if apiKey != "" {
		return apiKey
	}
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return apiKey
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}
