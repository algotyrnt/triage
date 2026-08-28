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

	// 1. Initial Setup mode: when instance_url is not yet configured, only permit localhost origins
	if configuredURL == "" {
		if isLocalhostOrigin(origin) {
			return origin
		}
		return ""
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

		if duration > 500*time.Millisecond && r.URL.Path != "/api/v1/events/stream" {
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

// ExtractBearerOrAPIKey retrieves credentials from headers or cookies.
func ExtractBearerOrAPIKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	apiKey := r.Header.Get("X-Triage-API-Key")
	if apiKey != "" {
		return apiKey
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if cookie, err := r.Cookie("triage_session"); err == nil && cookie != nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

type contextKey string

const userClaimsContextKey contextKey = "user_claims"

// getUserClaims extracts and verifies UserClaims from context, Authorization header, or session cookie.
func (s *Server) getUserClaims(r *http.Request) *UserClaims {
	if r == nil {
		return nil
	}
	if val := r.Context().Value(userClaimsContextKey); val != nil {
		if claims, ok := val.(*UserClaims); ok {
			return claims
		}
	}

	tokenStr := ""
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	} else if cookie, err := r.Cookie("triage_session"); err == nil && cookie != nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}
	// Note: r.URL.Query().Get("token") is explicitly ignored (SEC-003).

	if tokenStr != "" {
		secret, err := s.getSessionSecret(r.Context())
		if err == nil && secret != "" {
			if claims, err := ParseAndVerifyUserJWT(tokenStr, secret); err == nil {
				return claims
			}
		}
	}
	return nil
}

// public wraps a public endpoint with CORS, panic recovery, and logging.
func (s *Server) public(next http.HandlerFunc) http.HandlerFunc {
	return s.withMiddleware(next)
}

// withAuth enforces that the request has a valid JWT session or cookie.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return s.withMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := s.getUserClaims(r)
		if claims == nil {
			http.Error(w, `{"error":"Unauthorized: Authentication required"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userClaimsContextKey, claims)
		next(w, r.WithContext(ctx))
	})
}

// withAuthRole enforces that the request has a valid JWT session and meets required RBAC roles.
func (s *Server) withAuthRole(next http.HandlerFunc, requiredRoles ...string) http.HandlerFunc {
	return s.withMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := s.getUserClaims(r)
		if claims == nil {
			http.Error(w, `{"error":"Unauthorized: Authentication required"}`, http.StatusUnauthorized)
			return
		}

		if len(requiredRoles) > 0 {
			allowed := false
			for _, role := range requiredRoles {
				if strings.EqualFold(claims.Role, role) {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, `{"error":"Forbidden: Insufficient role permissions"}`, http.StatusForbidden)
				return
			}
		}

		ctx := context.WithValue(r.Context(), userClaimsContextKey, claims)
		next(w, r.WithContext(ctx))
	})
}
