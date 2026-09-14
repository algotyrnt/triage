// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"triage/engine/internal/db"
	"triage/engine/internal/llm"
)

func TestValidatePatchTargetFile(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"", true},
		{".", true},
		{"../outside.go", true},
		{"/abs/path.go", true},
		{".github/workflows/ci.yml", true},
		{".git/HEAD", true},
		{".vscode/settings.json", true},
		{".idea/workspace.xml", true},
		{"Dockerfile", true},
		{"subdir/Dockerfile.prod", true},
		{"docker-compose.yml", true},
		{".env.local", true},
		{"config/credential.json", true},
		{"secrets/app_secret.txt", true},
		{"keys/id_rsa", true},
		{"keys/id_ed25519", true},
		{"server.pem", true},
		{"server.key", true},
		{"cert.pfx", true},
		{"cert.p12", true},
		{"internal/api/handler.go", false},
		{"main.go", false},
	}

	for _, tt := range tests {
		err := ValidatePatchTargetFile(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePatchTargetFile(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
		}
	}
}

func TestHandleIncidents_CRUDAndFilters(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Without DB
	noDBServer := NewServer(Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	rec := httptest.NewRecorder()
	noDBServer.HandleIncidents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for no DB, got %d", rec.Code)
	}

	// 2. Add Project and Incidents
	_, repoID, err := env.DB.CreateProject(ctx, "testorg", "payments", "", "owner_user")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	inc1 := &db.Incident{
		ID:           "inc_1",
		RepositoryID: repoID,
		Title:        "Nil pointer in pay.go",
		Status:       "OPEN",
		File:         "pay.go",
		Line:         10,
		PanicMessage: "nil pointer",
		StackTrace:   "goroutine 1\npay.go:10",
		CreatedAt:    time.Now().UTC(),
	}
	inc2 := &db.Incident{
		ID:           "inc_2",
		RepositoryID: "",
		Title:        "Index out of range",
		Status:       "RESOLVED",
		File:         "slice.go",
		Line:         20,
		PanicMessage: "index out of range",
		StackTrace:   "goroutine 1\nslice.go:20",
		CreatedAt:    time.Now().UTC(),
	}
	if err := env.DB.SaveIncident(ctx, inc1); err != nil {
		t.Fatalf("save incident 1 failed: %v", err)
	}
	if err := env.DB.SaveIncident(ctx, inc2); err != nil {
		t.Fatalf("save incident 2 failed: %v", err)
	}

	// 3. List all incidents
	req = httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleIncidents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var res struct {
		Incidents []db.Incident `json:"incidents"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if len(res.Incidents) != 2 {
		t.Fatalf("expected 2 incidents, got %d", len(res.Incidents))
	}

	// 4. Filter by repository_id
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/incidents?repository_id=%s", repoID), nil)
	rec = httptest.NewRecorder()
	env.Server.HandleIncidents(rec, req)
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if len(res.Incidents) != 1 || res.Incidents[0].ID != "inc_1" {
		t.Fatalf("expected inc_1, got %+v", res.Incidents)
	}

	// 5. Filter by repo string
	req = httptest.NewRequest(http.MethodGet, "/api/v1/incidents?repo=testorg/payments", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleIncidents(rec, req)
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if len(res.Incidents) != 1 || res.Incidents[0].ID != "inc_1" {
		t.Fatalf("expected inc_1 by repo query, got %+v", res.Incidents)
	}

	// 6. DB error fetching incidents
	env.DB.Close()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleIncidents(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when DB is closed, got %d", rec.Code)
	}
}

func TestHandleResolveIncident_EdgeCases(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/resolve", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleResolveIncident(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rec.Code)
	}

	// 2. Database unavailable
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/resolve", strings.NewReader(`{"incident_id":"1"}`))
	rec = httptest.NewRecorder()
	noDB.HandleResolveIncident(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for no DB, got %d", rec.Code)
	}

	// 3. Missing incident_id
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/resolve", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	env.Server.HandleResolveIncident(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing ID, got %d", rec.Code)
	}

	// 4. Incident not found - resolving nonexistent incident is a safe no-op returning 200
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/resolve", strings.NewReader(`{"incident_id":"not_exist"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleResolveIncident(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for nonexistent incident resolution, got %d", rec.Code)
	}

	// 5. Success resolve with event publishing
	inc := &db.Incident{
		ID:           "inc_resolve_pub",
		Title:        "Panic",
		Status:       "OPEN",
		File:         "main.go",
		Line:         10,
		PanicMessage: "panic",
	}
	_ = env.DB.SaveIncident(ctx, inc)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/resolve", strings.NewReader(`{"incident_id":"inc_resolve_pub"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleResolveIncident(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for resolve, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, _ := env.DB.GetIncidentByID(ctx, "inc_resolve_pub")
	if updated.Status != "RESOLVED" {
		t.Errorf("expected RESOLVED, got %s", updated.Status)
	}
}

func TestHandleCreateIncidentIssue_EdgeCases(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/create-issue", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleCreateIncidentIssue(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}

	// 2. DB unavailable
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", strings.NewReader(`{"incident_id":"1"}`))
	rec = httptest.NewRecorder()
	noDB.HandleCreateIncidentIssue(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	// 3. Missing ID
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentIssue(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	// 4. Incident not found
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", strings.NewReader(`{"incident_id":"none"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentIssue(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	// 5. Already has issue
	incWithIssue := &db.Incident{
		ID:                "inc_with_issue",
		Title:             "Panic",
		Status:            "OPEN",
		File:              "main.go",
		GitHubIssueNumber: 42,
		GitHubIssueURL:    "https://github.com/test/repo/issues/42",
	}
	_ = env.DB.SaveIncident(ctx, incWithIssue)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", strings.NewReader(`{"incident_id":"inc_with_issue"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentIssue(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing issue, got %d", rec.Code)
	}

	// 6. GitHub App not configured
	incNoIssue := &db.Incident{
		ID:     "inc_no_issue",
		Title:  "Panic",
		Status: "OPEN",
		File:   "main.go",
	}
	_ = env.DB.SaveIncident(ctx, incNoIssue)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-issue", strings.NewReader(`{"incident_id":"inc_no_issue"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentIssue(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when github app not configured, got %d", rec.Code)
	}
}

func TestHandleCreateIncidentPR_EdgeCases(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. Method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/create-pr", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleCreateIncidentPR(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}

	// 2. DB unavailable
	noDB := NewServer(Config{})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", strings.NewReader(`{"incident_id":"1"}`))
	rec = httptest.NewRecorder()
	noDB.HandleCreateIncidentPR(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	// 3. Missing ID
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentPR(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	// 4. Incident not found
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", strings.NewReader(`{"incident_id":"none"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentPR(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	// 5. Already has PR
	incWithPR := &db.Incident{
		ID:             "inc_with_pr",
		Title:          "Panic",
		Status:         "OPEN",
		File:           "main.go",
		GitHubPRNumber: 101,
		GitHubPRURL:    "https://github.com/test/repo/pull/101",
	}
	_ = env.DB.SaveIncident(ctx, incWithPR)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", strings.NewReader(`{"incident_id":"inc_with_pr"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentPR(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing PR, got %d", rec.Code)
	}

	// 6. Invalid target file
	incInvalidFile := &db.Incident{
		ID:     "inc_bad_file",
		Title:  "Panic",
		Status: "OPEN",
		File:   "../outside/secret.key",
	}
	_ = env.DB.SaveIncident(ctx, incInvalidFile)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/incidents/create-pr", strings.NewReader(`{"incident_id":"inc_bad_file"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleCreateIncidentPR(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsafe file target, got %d", rec.Code)
	}
}

func TestHandleLLMAnalyzePanic_And_GeneratePatch(t *testing.T) {
	env := setupAPITestEnv(t)
	ctx := context.Background()

	// 1. AnalyzePanic - Method not allowed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/llm/analyze-panic", nil)
	rec := httptest.NewRecorder()
	env.Server.HandleLLMAnalyzePanic(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET analyze panic, got %d", rec.Code)
	}

	// 2. AnalyzePanic - Invalid payload
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/analyze-panic", strings.NewReader(`invalid json`))
	rec = httptest.NewRecorder()
	env.Server.HandleLLMAnalyzePanic(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid json, got %d", rec.Code)
	}

	// 3. AnalyzePanic - AI not configured
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/analyze-panic", strings.NewReader(`{"panicMessage":"nil ptr"}`))
	rec = httptest.NewRecorder()
	env.Server.HandleLLMAnalyzePanic(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when AI is not configured, got %d", rec.Code)
	}

	// 4. Configure LLM with custom mock client
	mockHTTP := mockClient(func(req *http.Request) (*http.Response, error) {
		bodyBytes, _ := io.ReadAll(req.Body)
		bodyStr := string(bodyBytes)
		reply := ""
		if strings.Contains(bodyStr, "unified git diff patch") {
			reply = "```diff\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n+ // fix\n```"
		} else {
			reply = "```json\n{\"root_cause\":\"nil dereference\",\"severity\":\"CRITICAL\",\"suggested_fix\":\"check nil\"}\n```"
		}
		respJSON := fmt.Sprintf(`{"choices":[{"message":{"content":%q},"finish_reason":"stop"}]}`, reply)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(respJSON)),
			Header:     make(http.Header),
		}, nil
	})

	_ = env.ConfigStore.SaveLLM(ctx, llm.Config{
		Provider:   "openai",
		APIKey:     "sk-mock-key",
		Model:      "gpt-4o",
		HTTPClient: mockHTTP,
	})
	env.Server.llmClient = mockHTTP

	// 5. AnalyzePanic - Success
	analyzePayload := map[string]string{
		"panicMessage":   "runtime error: invalid memory address",
		"rawStackTrace":  "goroutine 1 [running]:\nmain.go:10",
		"triggeringFile": "main.go",
		"astCode":        "func main() {}",
	}
	bodyBytes, _ := json.Marshal(analyzePayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/analyze-panic", bytes.NewReader(bodyBytes))
	rec = httptest.NewRecorder()
	env.Server.HandleLLMAnalyzePanic(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for analyze panic, got %d: %s", rec.Code, rec.Body.String())
	}
	var analyzeRes struct {
		Success        bool   `json:"success"`
		RootCause      string `json:"rootCause"`
		RecommendedFix string `json:"recommendedFix"`
		Severity       string `json:"severity"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&analyzeRes)
	if !analyzeRes.Success || analyzeRes.RootCause != "nil dereference" || analyzeRes.Severity != "CRITICAL" {
		t.Errorf("unexpected analyze response: %+v", analyzeRes)
	}

	// 6. GeneratePatch - Method not allowed
	req = httptest.NewRequest(http.MethodGet, "/api/v1/llm/generate-patch", nil)
	rec = httptest.NewRecorder()
	env.Server.HandleLLMGeneratePatch(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET patch, got %d", rec.Code)
	}

	// 7. GeneratePatch - Incident already has a patch in DB
	incWithPatch := &db.Incident{
		ID:             "inc_has_patch",
		Title:          "Crash",
		Status:         "OPEN",
		File:           "main.go",
		SuggestedPatch: "--- a/existing.go\n+++ b/existing.go\n",
	}
	_ = env.DB.SaveIncident(ctx, incWithPatch)

	patchReqPayload := map[string]string{
		"incidentId":     "inc_has_patch",
		"triggeringFile": "main.go",
		"panicMessage":   "nil ptr",
	}
	patchBytes, _ := json.Marshal(patchReqPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/generate-patch", bytes.NewReader(patchBytes))
	rec = httptest.NewRecorder()
	env.Server.HandleLLMGeneratePatch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for cached patch, got %d", rec.Code)
	}
	var patchRes struct {
		Success bool   `json:"success"`
		Patch   string `json:"patch"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&patchRes)
	if patchRes.Patch != "--- a/existing.go\n+++ b/existing.go\n" {
		t.Errorf("expected cached patch, got %s", patchRes.Patch)
	}

	// 8. GeneratePatch - Generate new patch with mock LLM
	newPatchReq := map[string]string{
		"incidentId":     "inc_new_patch",
		"triggeringFile": "main.go",
		"panicMessage":   "nil ptr",
		"astCode":        "func test() {}",
	}
	incNew := &db.Incident{
		ID:     "inc_new_patch",
		Title:  "Crash",
		Status: "OPEN",
		File:   "main.go",
	}
	_ = env.DB.SaveIncident(ctx, incNew)

	newPatchBytes, _ := json.Marshal(newPatchReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/generate-patch", bytes.NewReader(newPatchBytes))
	rec = httptest.NewRecorder()
	env.Server.HandleLLMGeneratePatch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for new patch generation, got %d: %s", rec.Code, rec.Body.String())
	}
	_ = json.NewDecoder(rec.Body).Decode(&patchRes)
	if !strings.Contains(patchRes.Patch, "--- a/main.go") {
		t.Errorf("expected generated diff, got %s", patchRes.Patch)
	}
}
