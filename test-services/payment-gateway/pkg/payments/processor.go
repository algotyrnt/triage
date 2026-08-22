// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"fmt"
	"time"

	"payment-gateway/pkg/fees"
	"payment-gateway/pkg/vault"
)

var (
	// DefaultCurrency represents default gateway transaction currency.
	DefaultCurrency = "USD"
	// StandardFeeSchedule standard 2.9% + 30c fee schedule.
	StandardFeeSchedule = &fees.FeeSchedule{
		BasisPoints:   290,
		FixedFeeCents: 30,
	}
)

// PaymentProcessor coordinates card tokenization, risk scoring, and ledger creation.
type PaymentProcessor struct {
	VaultClient    vault.CardVault
	MerchantConfig *MerchantConfig
	Environment    string
}

// NewPaymentProcessor creates a PaymentProcessor with vault client and config.
func NewPaymentProcessor(v vault.CardVault, cfg *MerchantConfig) *PaymentProcessor {
	return &PaymentProcessor{
		VaultClient:    v,
		MerchantConfig: cfg,
		Environment:    "production",
	}
}

// ProcessCharge processes a card payment request.
// NOTE: Bug simulation: If VaultClient is nil, calling TokenizeCard triggers nil interface method panic.
func (p *PaymentProcessor) ProcessCharge(req *PaymentRequest) (*Transaction, error) {
	if req == nil {
		return nil, fmt.Errorf("payment request is required")
	}

	// PANIC SITE: Calling method on nil interface if p.VaultClient == nil
	token, err := p.VaultClient.TokenizeCard(req.Method.CardPAN, req.Method.CardCVV)
	if err != nil {
		return nil, fmt.Errorf("vault tokenization error: %w", err)
	}

	tx := &Transaction{
		ID:         fmt.Sprintf("tx_%d", time.Now().UnixNano()),
		MerchantID: req.MerchantID,
		Status:     "CAPTURED",
		Amount:     req.AmountCents,
		Currency:   req.Currency,
		Token:      token,
		CreatedAt:  time.Now().UTC(),
	}

	return tx, nil
}

// VerifyMerchantWebhook verifies webhook payload HMAC signature against merchant secret.
// NOTE: Bug simulation: Accesses p.MerchantConfig.WebhookSecret without nil check on MerchantConfig.
func (p *PaymentProcessor) VerifyMerchantWebhook(payload []byte, signature string) bool {
	// PANIC SITE: Nil pointer dereference if p.MerchantConfig is nil
	secret := p.MerchantConfig.WebhookSecret
	return len(secret) > 0 && len(signature) > 0
}
