// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
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
	versionPkg "triage/engine/internal/version"
)

var (
	// Injected at build time via -ldflags "-X main.version=..."
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	opts := config.DefaultOptions()

	portFlag := flag.String("port", opts.Port, "HTTP server port")
	dataDirFlag := flag.String("data-dir", opts.DataDir, "Data directory for embedded SQLite storage")
	dbPathFlag := flag.String("db", "", "Explicit SQLite database file path (defaults to <data-dir>/triage.db)")
	logLevelFlag := flag.String("log-level", opts.LogLevel, "Log level: debug, info, warn, error")
	flag.Parse()

	if version != "" {
		versionPkg.Version = version
	}
	if commit != "" {
		versionPkg.Commit = commit
	}
	if date != "" {
		versionPkg.Date = date
	}
	currentVersion := versionPkg.Get()

	opts.Port = *portFlag
	opts.DataDir = *dataDirFlag
	opts.LogLevel = *logLevelFlag

	dbPath := opts.DatabasePath()
	if *dbPathFlag != "" {
		dbPath = *dbPathFlag
	}

	logger.InitLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.NewDB(ctx, dbPath)
	if err != nil {
		slog.Error("fatal: failed to initialize embedded database", "path", dbPath, "error", err)
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("connected triage to embedded SQLite database", "path", database.Path, "version", currentVersion)

	astManager := ast.NewManagerWithDB(database)
	defer astManager.Close()
	slog.Info("initialized AST indexer")

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
		Version:     currentVersion,
	})

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:         ":" + opts.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("triage server starting", "port", opts.Port, "ui", "embedded", "db", "sqlite")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stopChan
	slog.Info("shutting down triage server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
	}
	slog.Info("triage server stopped cleanly")
}
