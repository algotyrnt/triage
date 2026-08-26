// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"testing"
)

func TestGenerateSecureAPIKey(t *testing.T) {
	k1 := GenerateSecureAPIKey()
	k2 := GenerateSecureAPIKey()

	if len(k1) != 32 {
		t.Fatalf("expected 32 hex chars, got %d (%s)", len(k1), k1)
	}
	if len(k2) != 32 {
		t.Fatalf("expected 32 hex chars, got %d (%s)", len(k2), k2)
	}
	if k1 == k2 {
		t.Fatalf("expected unique keys, got identical %s", k1)
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc", "abc"},
		{"1234", "1234"},
		{"12345", "...2345"},
		{"7c4f9a02d8e135b6f7091234abcd5678", "...5678"},
	}

	for _, tc := range tests {
		got := MaskAPIKey(tc.input)
		if got != tc.expected {
			t.Errorf("MaskAPIKey(%q) = %q; expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestHashKey(t *testing.T) {
	key := "7c4f9a02d8e135b6f7091234abcd5678"
	h1 := HashKey(key)
	h2 := HashKey(key)

	if len(h1) != 64 {
		t.Fatalf("expected 64 char sha256 hex digest, got %d", len(h1))
	}
	if h1 != h2 {
		t.Fatalf("expected deterministic hash")
	}
}

func TestGenerateRandomToken(t *testing.T) {
	tok, err := GenerateRandomToken(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tok) != 32 {
		t.Fatalf("expected 32 chars for 16 bytes, got %d", len(tok))
	}
}
