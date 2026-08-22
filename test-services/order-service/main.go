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
	"order-service/pkg/notifier"
	"order-service/pkg/orders"
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
	port := getEnv("PORT", "8082")

	orderService := orders.NewOrderService("TechGadgets Store")
	_ = notifier.NewMailer("smtp.example.com")

	mux := http.NewServeMux()

	// Landing status endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service": "Order Processing Service", "port": "%s", "endpoints": ["/orders/checkout-nil-address", "/orders/discount-empty-slice", "/orders/uninitialized-metadata"]}`+"\n", port)
	})

	// 1. Crash Route 1: Nil Pointer Dereference in nested struct (order.Customer.ShippingAddress.ZipCode)
	mux.HandleFunc("/orders/checkout-nil-address", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating order checkout with nil shipping address...")
		order := &orders.Order{
			ID: "ORD-9901",
			Customer: &orders.Customer{
				ID:              "CUST-100",
				Name:            "Alex Mercer",
				Email:           "alex@example.com",
				ShippingAddress: nil, // Intentionally nil
			},
			Items: []orders.OrderItem{
				{SKU: "IPHONE-16", Title: "Smartphone", Quantity: 1, UnitPrice: 999.99},
			},
		}

		processed, err := orderService.ProcessCheckout(order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(processed)
	})

	// 2. Crash Route 2: Slice Bounds Out of Range in calculator.ComputeTotalDiscount
	mux.HandleFunc("/orders/discount-empty-slice", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating discount computation on empty slice...")
		emptyRules := []orders.DiscountRule{} // Empty slice passed
		calc := orders.NewDiscountCalculator("SUMMER_PROMO")
		_, _ = calc.ComputeTotalDiscount(250.0, emptyRules)
	})

	// 3. Crash Route 3: Write to uninitialized nil map in order.Metadata
	mux.HandleFunc("/orders/uninitialized-metadata", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Simulating assignment to nil map...")
		order := &orders.Order{
			ID:       "ORD-8822",
			Metadata: nil, // Intentionally nil map
		}
		orderService.UpdateOrderMetadata(order, "processed_by", "worker-1")
	})

	wrappedHandler := triage.Middleware(apiKey, engineURL)(mux)

	log.Printf("Starting order-service on :%s ...", port)
	if err := http.ListenAndServe(":"+port, wrappedHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
