// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateSecureAPIKey generates a cryptographically secure 128-bit random ingestion API key (32 hex characters).
func GenerateSecureAPIKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// MaskAPIKey returns the masked representation of an API key (e.g. "...5678").
func MaskAPIKey(key string) string {
	if len(key) <= 4 {
		return key
	}
	return fmt.Sprintf("...%s", key[len(key)-4:])
}

// HashKey computes the SHA-256 hex digest of a plaintext API key.
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GenerateRandomToken generates a cryptographically secure hex token with the specified byte length.
func GenerateRandomToken(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 16
	}
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
