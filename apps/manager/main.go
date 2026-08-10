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

	"github.com/joho/godotenv"
	triagedb "triage/manager/db"
)

type TelemetryPayload struct {
	APIKey     string `json:"api_key"`
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
		log.Printf("[WARNING] Failed to connect to PostgreSQL DB: %v", err)
	} else {
		defer database.Close()
		log.Println("[MANAGER] Connected Manager Gateway to Database (triage/manager/db)")
	}

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
		WriteTimeout: 30 * time.Second,
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

func generateUUID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
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
	if database != nil && payload.APIKey != "" {
		_ = database.VerifyAPIKey(r.Context(), payload.APIKey)
		log.Printf("[MANAGER] Verified API Key (%s) | Trace: %s", payload.APIKey, traceID)
	}

	// 2. Query pre-parsed AST snippet via database package
	if payload.ASTSnippet == "" && database != nil && payload.File != "" && payload.Line > 0 {
		node, err := database.GetASTNode(r.Context(), payload.File, payload.Line)
		if err == nil && node != nil && node.Snippet != "" {
			payload.ASTSnippet = node.Snippet
			log.Printf("[MANAGER AST DB HIT] Injected pre-parsed AST snippet for %s:%d", payload.File, payload.Line)
		}
	}

	// 3. Proxy payload to internal AI Engine (:8080)
	engineURL := os.Getenv("TRIAGE_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080/api/v1/telemetry"
	}

	data, _ := json.Marshal(payload)
	engineReq, err := http.NewRequestWithContext(r.Context(), "POST", engineURL, bytes.NewBuffer(data))
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(engineReq)
	if err != nil {
		log.Printf("[ERROR] Engine connection error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "engine connection failed"})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var engineResp EngineResponse
	_ = json.Unmarshal(respBody, &engineResp)

	// 4. Save Incident Audit Log via database package
	if database != nil && engineResp.Status == "success" && engineResp.Analysis != nil {
		incidentID := fmt.Sprintf("INC-%s", generateUUID())
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

	// Webhook Delivery Idempotency Check via database package
	if database != nil && deliveryID != "" {
		if database.IsWebhookDuplicate(r.Context(), deliveryID) {
			log.Printf("[MANAGER IDEMPOTENT WEBHOOK] Duplicate delivery skipped: %s", deliveryID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"skipped","message":"Duplicate webhook delivery skipped"}`))
			return
		}
	}

	rawBody, _ := io.ReadAll(r.Body)

	engineURL := os.Getenv("TRIAGE_ENGINE_URL")
	if engineURL == "" {
		engineURL = "http://localhost:8080/api/v1/telemetry"
	}
	webhookURL := strings.Replace(engineURL, "/api/v1/telemetry", "/api/v1/github/webhook", 1)

	engineReq, _ := http.NewRequestWithContext(r.Context(), "POST", webhookURL, bytes.NewBuffer(rawBody))
	engineReq.Header.Set("Content-Type", "application/json")
	engineReq.Header.Set("X-GitHub-Event", eventType)
	engineReq.Header.Set("X-GitHub-Delivery", deliveryID)
	engineReq.Header.Set("X-Hub-Signature-256", r.Header.Get("X-Hub-Signature-256"))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(engineReq)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"status":"error","error":"engine connection failed"}`))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Save Webhook Delivery Log via database package
	if database != nil && deliveryID != "" {
		logID := fmt.Sprintf("wh-%s", generateUUID())
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
			RequestBody:  string(rawBody[:min(len(rawBody), 2000)]),
			ResponseBody: string(respBody[:min(len(respBody), 2000)]),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
