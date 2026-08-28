// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// In production / embedded builds, the Vite assets in dist/ are embedded.
//
//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves embedded static frontend assets
// with optimal caching headers and Single Page Application (SPA) fallback routing.
func Handler() http.Handler {
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Dashboard UI not built. Run 'bun run build' in dashboard.", http.StatusServiceUnavailable)
		})
	}

	indexHTML, indexErr := fs.ReadFile(subFS, "index.html")
	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			reqPath = "index.html"
		}

		// Check if file exists in the embedded subFS
		f, err := subFS.Open(reqPath)
		if err == nil {
			_ = f.Close()
			if ext := filepath.Ext(reqPath); ext != "" {
				if mimeType := mime.TypeByExtension(ext); mimeType != "" {
					w.Header().Set("Content-Type", mimeType)
				}
			}
			// Immutable cache for content-hashed assets
			if strings.HasPrefix(reqPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// If the path looks like a static asset (has extension) and was not found, return 404
		if strings.Contains(filepath.Base(reqPath), ".") {
			http.NotFound(w, r)
			return
		}

		// SPA route fallback: serve index.html with no-cache header
		if indexErr != nil {
			http.Error(w, "index.html not found in embedded UI bundle", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(indexHTML)
	})
}
