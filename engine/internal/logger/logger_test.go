// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"log/slog"
	"os"
	"testing"
)

func TestInitLogger_DefaultText(t *testing.T) {
	_ = os.Unsetenv("LOG_LEVEL")
	_ = os.Unsetenv("LOG_FORMAT")
	_ = os.Unsetenv("ENVIRONMENT")

	l := InitLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestInitLogger_JSONAndDebugLevel(t *testing.T) {
	_ = os.Setenv("LOG_LEVEL", "DEBUG")
	_ = os.Setenv("LOG_FORMAT", "json")
	defer func() {
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("LOG_FORMAT")
	}()

	l := InitLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	slog.Debug("test debug structured logging", "key", "value")
}

func TestInitLogger_WarnAndError(t *testing.T) {
	_ = os.Setenv("LOG_LEVEL", "WARN")
	_ = os.Setenv("ENVIRONMENT", "production")
	defer func() {
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("ENVIRONMENT")
	}()

	l := InitLogger()
	if l == nil {
		t.Fatal("expected non-nil logger for WARN")
	}

	_ = os.Setenv("LOG_LEVEL", "WARNING")
	_ = os.Setenv("ENVIRONMENT", "prod")
	l2 := InitLogger()
	if l2 == nil {
		t.Fatal("expected non-nil logger for WARNING")
	}

	_ = os.Setenv("LOG_LEVEL", "ERROR")
	_ = os.Unsetenv("ENVIRONMENT")
	l3 := InitLogger()
	if l3 == nil {
		t.Fatal("expected non-nil logger for ERROR")
	}
}
