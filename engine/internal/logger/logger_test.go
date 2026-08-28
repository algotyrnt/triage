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
