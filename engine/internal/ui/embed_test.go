// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_AllBranches(t *testing.T) {
	handler := Handler()

	// 1. Test root "/"
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, reqRoot)

	if recRoot.Code != http.StatusOK {
		t.Fatalf("expected status 200 for root, got %d", recRoot.Code)
	}

	// 2. Test static file without asset prefix: "/favicon.svg"
	reqFav := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	recFav := httptest.NewRecorder()
	handler.ServeHTTP(recFav, reqFav)

	if recFav.Code != http.StatusOK {
		t.Fatalf("expected status 200 for favicon.svg, got %d", recFav.Code)
	}
	if recFav.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache for root asset, got %s", recFav.Header().Get("Cache-Control"))
	}

	// 3. Test content-hashed asset in /assets/
	reqAsset := httptest.NewRequest(http.MethodGet, "/assets/index-BjxQqYvZ.css", nil)
	recAsset := httptest.NewRecorder()
	handler.ServeHTTP(recAsset, reqAsset)

	if recAsset.Code != http.StatusOK {
		t.Fatalf("expected status 200 for asset css, got %d", recAsset.Code)
	}
	if !strings.Contains(recAsset.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("expected immutable Cache-Control for assets/, got %s", recAsset.Header().Get("Cache-Control"))
	}

	// 4. Test missing file with extension -> 404
	reqMissing := httptest.NewRequest(http.MethodGet, "/assets/notfound.js", nil)
	recMissing := httptest.NewRecorder()
	handler.ServeHTTP(recMissing, reqMissing)

	if recMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing static asset with extension, got %d", recMissing.Code)
	}

	// 5. Test SPA client route fallback
	reqSPA := httptest.NewRequest(http.MethodGet, "/incidents/INC-12345", nil)
	recSPA := httptest.NewRecorder()
	handler.ServeHTTP(recSPA, reqSPA)

	if recSPA.Code != http.StatusOK {
		t.Fatalf("expected status 200 for SPA route, got %d", recSPA.Code)
	}
	if !strings.Contains(recSPA.Header().Get("Content-Type"), "text/html") {
		t.Errorf("expected Content-Type text/html for SPA route fallback, got %s", recSPA.Header().Get("Content-Type"))
	}
	if recSPA.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache for SPA fallback, got %s", recSPA.Header().Get("Cache-Control"))
	}
}
