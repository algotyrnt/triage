// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"triage/engine/internal/api"
	"triage/engine/internal/ast"
	"triage/engine/internal/config"
	"triage/engine/internal/db"
	"triage/engine/internal/logger"
)

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		slog.Error("fatal: invalid environment configuration", "error", err)
		os.Exit(1)
	}

	logger.InitLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.NewDB(ctx, env.DatabaseURL)
	if err != nil {
		slog.Error("fatal: failed to connect to PostgreSQL database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("connected engine to PostgreSQL database pool")

	astManager, err := ast.NewManager(ctx, env.DatabaseURL)
	if err != nil {
		slog.Error("fatal: failed to connect AST manager to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer astManager.Close()
	slog.Info("connected engine to PostgreSQL AST indexer")

	configStore := config.NewStore(database)
	if _, err := configStore.EnsureSessionSecret(ctx); err != nil {
		slog.Warn("failed to ensure session secret", "error", err)
	}

	githubApp, _ := configStore.GetGitHubApp(ctx)

	server := api.NewServer(api.Config{
		DB:          database,
		ConfigStore: configStore,
		GitHubApp:   githubApp,
		ASTManager:  astManager,
	})

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:         ":" + env.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("triage engine server starting", "port", env.Port)
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
