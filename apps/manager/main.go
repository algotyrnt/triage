// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	triagedb "triage/manager/db"

	"github.com/joho/godotenv"
)

type TelemetryPayload struct {
	APIKey     string `json:"api_key,omitempty"`
	Owner      string `json:"owner,omitempty"`
	Repo       string `json:"repo,omitempty"`
	Commit     string `json:"commit,omitempty"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	StackTrace string `json:"stack_trace"`
	ASTSnippet string `json:"ast_snippet,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

type EngineResponse struct {
	Status       string      `json:"status"`
	TraceID      string      `json:"trace_id,omitempty"`
	AST          string      `json:"ast,omitempty"`
	Analysis     *AIAnalysis `json:"analysis,omitempty"`
	ErrorMessage string      `json:"error,omitempty"`
}

type AIAnalysis struct {
	RootCause    string `json:"root_cause"`
	SuggestedFix string `json:"suggested_fix"`
}

var database *triagedb.DB

func loadEnvLocal() {
	_ = godotenv.Load(".env.local", ".env")
}

func main() {
	loadEnvLocal()

	dbURL := os.Getenv("DATABASE_URL")
	var err error
	database, err = triagedb.NewDB(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to PostgreSQL DB: %v", err)
	}
	defer database.Close()
	log.Println("[MANAGER] Connected Manager Gateway to Database (triage/manager/db)")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/telemetry", handleTelemetryIngest)
	mux.HandleFunc("/api/github/webhook", handleWebhookProxy)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Triage Go Manager Function listening on :%s ...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Manager server stopped unexpectedly: %v", err)
		}
	}()

	<-stopCtx.Done()
	log.Println("Shutting down Triage Go Manager gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","component":"triage-manager-go","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func truncateUTF8(s string, maxChars int) string {
	if utf8.RuneCountInString(s) <= maxChars {
		return s
	}
	var count int
	for idx := range s {
		if count == maxChars {
			return s[:idx]
		}
		count++
	}
	return s
}

func handleTelemetryIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var payload TelemetryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "invalid JSON body"})
		return
	}

	traceID := payload.TraceID
	if traceID == "" {
		traceID = r.Header.Get("X-Triage-Trace-ID")
	}

	// 1. Verify API Key via database package
	if payload.APIKey == "" || database == nil || !database.VerifyAPIKey(r.Context(), payload.APIKey) {
		log.Printf("[MANAGER WARNING] Unauthorized telemetry ingest attempt | Trace: %s", traceID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "unauthorized: missing or invalid API key"})
		return
	}
	log.Printf("[MANAGER] Verified API Key | Trace: %s", traceID)

	// 2. Query pre-parsed AST snippet via database package
	if payload.ASTSnippet == "" && database != nil && payload.File != "" && payload.Line > 0 {
		node, err := database.GetASTNode(r.Context(), payload.Owner, payload.Repo, payload.File, payload.Line)
		if err == nil && node != nil && node.Snippet != "" {
			payload.ASTSnippet = node.Snippet
			log.Printf("[MANAGER AST DB HIT] Injected pre-parsed AST snippet for %s/%s %s:%d", payload.Owner, payload.Repo, payload.File, payload.Line)
		}
	}

	// Clear API key before forwarding to Engine
	payload.APIKey = ""

	// 3. Proxy payload to internal AI Engine (:8080)
	engineBaseURL := os.Getenv("TRIAGE_ENGINE_BASE_URL")
	if engineBaseURL == "" {
		engineBaseURL = os.Getenv("TRIAGE_ENGINE_URL")
		if engineBaseURL == "" {
			engineBaseURL = "http://localhost:8080"
		}
	}
	engineBaseURL = strings.TrimRight(engineBaseURL, "/")
	if !strings.HasSuffix(engineBaseURL, "/api/v1/telemetry") {
		engineBaseURL += "/api/v1/telemetry"
	}

	data, err := json.Marshal(payload)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "failed to marshal engine payload"})
		return
	}

	engineReq, err := http.NewRequestWithContext(r.Context(), "POST", engineBaseURL, bytes.NewBuffer(data))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "failed to build engine request"})
		return
	}
	engineReq.Header.Set("Content-Type", "application/json")
	if traceID != "" {
		engineReq.Header.Set("X-Triage-Trace-ID", traceID)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(engineReq)
	if err != nil {
		log.Printf("[ERROR] Engine connection error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "engine connection failed"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(respBody) > 1<<20 {
		log.Printf("[ERROR] Engine response read failed or size limit exceeded: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "engine response invalid or oversized"})
		return
	}

	var engineResp EngineResponse
	_ = json.Unmarshal(respBody, &engineResp)

	// 4. Save Incident Audit Log via database package
	if database != nil && engineResp.Status == "success" && engineResp.Analysis != nil {
		randID, idErr := generateID()
		if idErr != nil {
			log.Printf("[ERROR] Failed to generate incident ID: %v", idErr)
		} else {
			incidentID := fmt.Sprintf("INC-%s", randID)
			panicMsg := "Runtime panic"
			if payload.StackTrace != "" {
				lines := strings.Split(payload.StackTrace, "\n")
				if len(lines) > 0 {
					panicMsg = lines[0]
				}
			}

			_ = database.SaveIncident(r.Context(), &triagedb.Incident{
				ID:           incidentID,
				Title:        fmt.Sprintf("Panic in %s:%d", payload.File, payload.Line),
				Status:       "CRITICAL",
				File:         payload.File,
				Line:         payload.Line,
				PanicMessage: panicMsg,
				StackTrace:   payload.StackTrace,
				ASTSnippet:   engineResp.AST,
				RootCause:    engineResp.Analysis.RootCause,
				SuggestedFix: engineResp.Analysis.SuggestedFix,
			})
			log.Printf("[MANAGER DB SAVE] Created Incident %s in PostgreSQL", incidentID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func handleWebhookProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deliveryID := r.Header.Get("X-GitHub-Delivery")
	eventType := r.Header.Get("X-GitHub-Event")

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "invalid request body or size limit exceeded"})
		return
	}

	var logID string
	// Pre-dispatch Webhook Delivery Idempotency Check & Log Insertion
	if database != nil && deliveryID != "" {
		isDup, dupErr := database.IsWebhookDuplicate(r.Context(), deliveryID)
		if dupErr != nil || isDup {
			log.Printf("[MANAGER IDEMPOTENT WEBHOOK] Duplicate delivery or check error skipped: %s (err: %v)", deliveryID, dupErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"skipped","message":"Duplicate webhook delivery skipped"}`))
			return
		}

		randID, idErr := generateID()
		if idErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "failed to generate log ID"})
			return
		}
		logID = fmt.Sprintf("wh-%s", randID)

		initErr := database.SaveWebhookLog(r.Context(), &triagedb.WebhookLog{
			ID:           logID,
			DeliveryID:   deliveryID,
			EventType:    eventType,
			Status:       "PENDING",
			StatusCode:   0,
			RequestBody:  truncateUTF8(string(rawBody), 2000),
			ResponseBody: "",
		})
		if initErr != nil {
			log.Printf("[MANAGER WEBHOOK DUP] Duplicate delivery constraint failure: %v", initErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"skipped","message":"Duplicate webhook delivery skipped"}`))
			return
		}
	}

	engineBaseURL := os.Getenv("TRIAGE_ENGINE_BASE_URL")
	if engineBaseURL == "" {
		engineBaseURL = "http://localhost:8080"
	}
	webhookURL := strings.TrimRight(engineBaseURL, "/") + "/api/v1/github/webhook"

	engineReq, reqErr := http.NewRequestWithContext(r.Context(), "POST", webhookURL, bytes.NewBuffer(rawBody))
	if reqErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "failed to build engine request"})
		return
	}
	engineReq.Header.Set("Content-Type", "application/json")
	engineReq.Header.Set("X-GitHub-Event", eventType)
	engineReq.Header.Set("X-GitHub-Delivery", deliveryID)
	engineReq.Header.Set("X-Hub-Signature-256", r.Header.Get("X-Hub-Signature-256"))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(engineReq)
	if err != nil {
		if database != nil && logID != "" {
			_ = database.SaveWebhookLog(r.Context(), &triagedb.WebhookLog{
				ID:           logID,
				DeliveryID:   deliveryID,
				EventType:    eventType,
				Status:       "ERROR",
				StatusCode:   http.StatusBadGateway,
				RequestBody:  truncateUTF8(string(rawBody), 2000),
				ResponseBody: "engine connection failed",
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"status":"error","error":"engine connection failed"}`))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))

	// Update Webhook Delivery Log via database package
	if database != nil && logID != "" {
		statusStr := "SUCCESS"
		if resp.StatusCode != 200 {
			statusStr = "ERROR"
		}
		_ = database.SaveWebhookLog(r.Context(), &triagedb.WebhookLog{
			ID:           logID,
			DeliveryID:   deliveryID,
			EventType:    eventType,
			Status:       statusStr,
			StatusCode:   resp.StatusCode,
			RequestBody:  truncateUTF8(string(rawBody), 2000),
			ResponseBody: truncateUTF8(string(respBody), 2000),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}
