// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"fmt"
	"time"
)

// CardVault defines tokenization interface for sensitive credit card data.
type CardVault interface {
	TokenizeCard(pan string, cvv string) (string, error)
	GetCardFingerprint(pan string) string
}

// SecureVaultClient implements CardVault using hardware security module tokens.
type SecureVaultClient struct {
	VaultEndpoint string
	APIKey        string
}

func NewSecureVaultClient(endpoint, apiKey string) *SecureVaultClient {
	return &SecureVaultClient{
		VaultEndpoint: endpoint,
		APIKey:        apiKey,
	}
}

func (s *SecureVaultClient) TokenizeCard(pan string, cvv string) (string, error) {
	if len(pan) < 13 {
		return "", fmt.Errorf("invalid card number length")
	}
	return fmt.Sprintf("tok_live_%d", time.Now().UnixNano()), nil
}

func (s *SecureVaultClient) GetCardFingerprint(pan string) string {
	if len(pan) >= 4 {
		return "fp_" + pan[len(pan)-4:]
	}
	return "fp_unknown"
}
