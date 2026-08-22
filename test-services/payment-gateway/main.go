// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	triage "github.com/algotyrnt/triage/sdk/go"
	"payment-gateway/pkg/fees"
	"payment-gateway/pkg/payments"
)

func requireEnv(key string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		log.Fatalf("Fatal: missing required environment variable %q", key)
	}
	return val
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func main() {
	apiKey := requireEnv("TRIAGE_API_KEY")
	engineURL := getEnv("TRIAGE_ENGINE_URL", "http://localhost:8080/api/v1/telemetry")
	port := getEnv("PORT", "8083")

	// Uninitialized processor (VaultClient is nil, MerchantConfig is nil)
	uninitializedProcessor := payments.NewPaymentProcessor(nil, nil)

	mux := http.NewServeMux()

	// Landing status endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service": "Payment Gateway Service", "port": "%s", "endpoints": ["/payments/nil-vault-client", "/payments/nil-config", "/payments/zero-division"]}`+"\n", port)
	})

	// 1. Crash Route 1: Nil Interface Method Invocation (p.VaultClient.TokenizeCard)
	mux.HandleFunc("/payments/nil-vault-client", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating charge with nil vault client...")
		req := &payments.PaymentRequest{
			MerchantID:  "merch_alpha_99",
			AmountCents: 5000,
			Currency:    "USD",
			Method: payments.PaymentMethod{
				Type:    "card",
				CardPAN: "4242424242424242",
				CardCVV: "123",
			},
		}

		tx, err := uninitializedProcessor.ProcessCharge(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(tx)
	})

	// 2. Crash Route 2: Nil Struct Dereference on uninitialized config (p.MerchantConfig.WebhookSecret)
	mux.HandleFunc("/payments/nil-config", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating webhook signature check with uninitialized config...")
		_ = uninitializedProcessor.VerifyMerchantWebhook([]byte(`{"event":"charge.succeeded"}`), "sig_test_123")
	})

	// 3. Crash Route 3: Integer Division by Zero (installments = 0)
	mux.HandleFunc("/payments/zero-division", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating installment fee calculation with 0 installments...")
		_, _ = fees.CalculateFeePerInstallment(payments.StandardFeeSchedule, 10000, 0)
	})

	wrappedHandler := triage.Middleware(apiKey, engineURL)(mux)

	log.Printf("Starting payment-gateway on :%s ...", port)
	if err := http.ListenAndServe(":"+port, wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
