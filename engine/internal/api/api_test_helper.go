// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"path/filepath"
	"testing"

	"triage/engine/internal/config"
	"triage/engine/internal/db"
	"triage/engine/internal/github"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func setupTestGitHubApp(t *testing.T) *github.AppConfig {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	der := x509.MarshalPKCS1PrivateKey(privKey)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})

	cfg, err := github.LoadAppConfig(12345, pemBytes, "whsec_test", "client_123", "sec_123")
	if err != nil {
		t.Fatalf("failed to load app config: %v", err)
	}
	return cfg
}

type testAPIEnv struct {
	Server      *Server
	DB          *db.DB
	ConfigStore *config.Store
	Secret      string
	OwnerUser   *db.User
	OwnerToken  string
	DevUser     *db.User
	DevToken    string
	ViewerUser  *db.User
	ViewerToken string
}

func setupAPITestEnv(t *testing.T) *testAPIEnv {
	t.Helper()
	ctx := context.Background()

	dbPath := filepath.Join(t.TempDir(), "test_api.db")
	testDB, err := db.NewDB(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to init sqlite db: %v", err)
	}
	t.Cleanup(func() {
		testDB.Close()
	})

	cfgStore := config.NewStore(testDB)
	secret, err := cfgStore.EnsureSessionSecret(ctx)
	if err != nil {
		t.Fatalf("failed to ensure session secret: %v", err)
	}

	// 1. Owner User
	owner, err := testDB.UpsertUserWithRole(ctx, "1001", "owner_user", "owner@example.com", "https://avatar.example.com/owner.png", "Owner")
	if err != nil {
		t.Fatalf("failed to save owner user: %v", err)
	}
	ownerToken, err := GenerateUserJWT(owner, secret)
	if err != nil {
		t.Fatalf("failed to generate owner JWT: %v", err)
	}

	// 2. Dev User
	dev, err := testDB.UpsertUserWithRole(ctx, "1002", "dev_user", "dev@example.com", "https://avatar.example.com/dev.png", "Developer")
	if err != nil {
		t.Fatalf("failed to save dev user: %v", err)
	}
	devToken, err := GenerateUserJWT(dev, secret)
	if err != nil {
		t.Fatalf("failed to generate dev JWT: %v", err)
	}

	// 3. Viewer User
	viewer, err := testDB.UpsertUserWithRole(ctx, "1003", "viewer_user", "viewer@example.com", "https://avatar.example.com/viewer.png", "Viewer")
	if err != nil {
		t.Fatalf("failed to save viewer user: %v", err)
	}
	viewerToken, err := GenerateUserJWT(viewer, secret)
	if err != nil {
		t.Fatalf("failed to generate viewer JWT: %v", err)
	}

	srv := NewServer(Config{
		DB:          testDB,
		ConfigStore: cfgStore,
	})

	return &testAPIEnv{
		Server:      srv,
		DB:          testDB,
		ConfigStore: cfgStore,
		Secret:      secret,
		OwnerUser:   owner,
		OwnerToken:  ownerToken,
		DevUser:     dev,
		DevToken:    devToken,
		ViewerUser:  viewer,
		ViewerToken: viewerToken,
	}
}
