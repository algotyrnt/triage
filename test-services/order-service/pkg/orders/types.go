// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package orders

import "time"

// Address represents a customer's physical shipping or billing location.
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// Customer contains identity and contact information for an order.
type Customer struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Email           string   `json:"email"`
	ShippingAddress *Address `json:"shipping_address"`
	BillingAddress  *Address `json:"billing_address"`
}

// OrderItem represents an individual line item in a customer order.
type OrderItem struct {
	SKU       string  `json:"sku"`
	Title     string  `json:"title"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// DiscountRule defines a promotional discount percentage or flat reduction.
type DiscountRule struct {
	Code       string  `json:"code"`
	Percentage float64 `json:"percentage"`
	MaxLimit   float64 `json:"max_limit"`
	ExpiresAt  time.Time
}

// Order is the root domain entity containing customer, line items, and fulfillment state.
type Order struct {
	ID            string            `json:"id"`
	Customer      *Customer         `json:"customer"`
	Items         []OrderItem       `json:"items"`
	DiscountRules []DiscountRule    `json:"discount_rules"`
	Subtotal      float64           `json:"subtotal"`
	TaxAmount     float64           `json:"tax_amount"`
	TotalAmount   float64           `json:"total_amount"`
	Status        string            `json:"status"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
}
