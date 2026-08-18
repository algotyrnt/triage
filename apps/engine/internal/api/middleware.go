// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"triage/engine/internal/session"
)

// withMiddleware wraps a handler with standard CORS, panic recovery, and logging.
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

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Triage-API-Key")

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

// IsValidAPIKey checks if the provided API key is valid against database or environment.
func (s *Server) IsValidAPIKey(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}

	if s.db != nil && s.db.VerifyAPIKey(ctx, key) {
		return true
	}

	expectedKey := os.Getenv("TRIAGE_API_KEY")
	if expectedKey != "" && subtle.ConstantTimeCompare([]byte(key), []byte(expectedKey)) == 1 {
		return true
	}

	return false
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

// ValidateSessionToken parses the JWT session token if provided.
func (s *Server) ValidateSessionToken(tokenString string) (*session.Claims, error) {
	if s.sessionSecret == "" {
		s.sessionSecret = os.Getenv("SESSION_SECRET")
	}
	return session.ValidateSessionJWT(tokenString, s.sessionSecret)
}
