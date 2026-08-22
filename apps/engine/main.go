// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"triage/engine/internal/api"
	"triage/engine/internal/ast"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
	"triage/engine/internal/logger"

	"github.com/joho/godotenv"
)

func loadEnvLocal() {
	_ = godotenv.Load(".env.local", ".env")
}

func loadGitHubAppConfig(ctx context.Context, database *db.DB) *github.AppConfig {
	if database == nil {
		return nil
	}
	appIDStr, _ := database.GetInstanceConfig(ctx, "github_app_id")
	pemKey, _ := database.GetInstanceConfig(ctx, "github_app_private_key")
	webhookSecret, _ := database.GetInstanceConfig(ctx, "github_app_webhook_secret")
	clientID, _ := database.GetInstanceConfig(ctx, "github_app_client_id")
	clientSecret, _ := database.GetInstanceConfig(ctx, "github_app_client_secret")

	if appIDStr == "" || pemKey == "" {
		return nil
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		slog.Warn("invalid GITHUB_APP_ID in database config", "error", err, "app_id", appIDStr)
		return nil
	}
	cfg, err := github.LoadAppConfig(appID, []byte(pemKey), webhookSecret, clientID, clientSecret)
	if err != nil {
		slog.Warn("failed to load GitHub App config", "error", err)
		return nil
	}
	slog.Info("loaded GitHub App configuration", "app_id", appID)
	return cfg
}

func main() {
	loadEnvLocal()
	logger.InitLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		slog.Error("fatal: missing required environment variable DATABASE_URL")
		os.Exit(1)
	}

	database, err := db.NewDB(ctx, dbURL)
	if err != nil {
		slog.Error("fatal: failed to connect to PostgreSQL database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("connected engine to PostgreSQL database pool")

	astManager, err := ast.NewManager(ctx, dbURL)
	if err != nil {
		slog.Error("fatal: failed to connect AST manager to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer astManager.Close()
	slog.Info("connected engine to PostgreSQL AST indexer")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	engineURL := os.Getenv("TRIAGE_ENGINE_URL")
	if engineURL == "" {
		engineURL = fmt.Sprintf("http://localhost:%s", port)
	}

	dashboardURL := os.Getenv("TRIAGE_DASHBOARD_URL")
	if dashboardURL == "" {
		dashboardURL = "http://localhost:3000"
	}

	sessionSecret := ""
	if database != nil {
		storedSecret, _ := database.GetInstanceConfig(ctx, "session_secret")
		if storedSecret != "" {
			sessionSecret = storedSecret
		} else {
			b := make([]byte, 32)
			_, _ = rand.Read(b)
			sessionSecret = hex.EncodeToString(b)
			_ = database.SaveInstanceConfig(ctx, "session_secret", sessionSecret)
			slog.Info("auto-generated and persisted secure session_secret")
		}
	} else {
		sessionSecret = "dev-secret-change-me-in-production"
	}

	var githubApp *github.AppConfig
	if database != nil {
		githubApp = loadGitHubAppConfig(ctx, database)
	}

	server := api.NewServer(api.Config{
		DB:         database,
		GitHubApp:  githubApp,
		ASTManager: astManager,
		AppURL:     dashboardURL,
		EngineURL:  engineURL,
	})

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("triage engine server starting", "port", port, "engine_url", engineURL, "dashboard_url", dashboardURL)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stopChan
	slog.Info("shutting down triage engine gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
	}
	slog.Info("triage engine stopped cleanly")
}
