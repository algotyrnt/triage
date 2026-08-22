// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package payments

import "time"

// PaymentMethod details provided by the payer.
type PaymentMethod struct {
	Type    string `json:"type"` // "card", "bank_transfer", "crypto"
	CardPAN string `json:"card_pan,omitempty"`
	CardCVV string `json:"card_cvv,omitempty"`
	TokenID string `json:"token_id,omitempty"`
}

// MerchantConfig stores per-merchant gateway preferences and credentials.
type MerchantConfig struct {
	MerchantID     string `json:"merchant_id"`
	WebhookSecret  string `json:"webhook_secret"`
	SettlementBank string `json:"settlement_bank"`
	AutoCapture    bool   `json:"auto_capture"`
}

// PaymentRequest contains payment payload sent from clients.
type PaymentRequest struct {
	MerchantID  string        `json:"merchant_id"`
	AmountCents int           `json:"amount_cents"`
	Currency    string        `json:"currency"`
	Method      PaymentMethod `json:"method"`
}

// Transaction represents the resulting payment record.
type Transaction struct {
	ID         string    `json:"id"`
	MerchantID string    `json:"merchant_id"`
	Status     string    `json:"status"`
	Amount     int       `json:"amount"`
	Currency   string    `json:"currency"`
	Token      string    `json:"token"`
	CreatedAt  time.Time `json:"created_at"`
}
