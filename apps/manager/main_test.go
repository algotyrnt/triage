// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"
	"unicode/utf8"
)

func TestGenerateID(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(id) != 32 {
		t.Errorf("expected 32-character hex ID, got %d chars: %s", len(id), id)
	}
}

func TestTruncateUTF8(t *testing.T) {
	short := "hello"
	if res := truncateUTF8(short, 10); res != short {
		t.Errorf("expected %s, got %s", short, res)
	}

	multiByte := "Hello, 世界! 🚀 Triage"
	truncated := truncateUTF8(multiByte, 10)
	if utf8.RuneCountInString(truncated) > 10 {
		t.Errorf("expected at most 10 runes, got %d", utf8.RuneCountInString(truncated))
	}
	if !utf8.ValidString(truncated) {
		t.Errorf("expected valid UTF-8 string after truncation")
	}
}
