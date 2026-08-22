// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package orders

import (
	"fmt"
	"time"
)

// OrderService manages the full checkout and lifecycle processing of orders.
type OrderService struct {
	Calculator *DiscountCalculator
	StoreName  string
}

// NewOrderService constructs an OrderService with default calculators.
func NewOrderService(storeName string) *OrderService {
	return &OrderService{
		Calculator: NewDiscountCalculator("SPRING_SALE"),
		StoreName:  storeName,
	}
}

// ProcessCheckout processes order validation, calculates discounts and taxes, and locks shipping.
// NOTE: Bug simulation: Accesses nested customer shipping address without validating if customer or address is non-nil.
func (s *OrderService) ProcessCheckout(order *Order) (*Order, error) {
	if order == nil {
		return nil, fmt.Errorf("order cannot be nil")
	}

	// Calculate subtotal from items
	var subtotal float64
	for _, item := range order.Items {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}
	order.Subtotal = subtotal

	// PANIC SITE: Nested nil pointer dereference if order.Customer or order.Customer.ShippingAddress is nil
	shippingZip := order.Customer.ShippingAddress.ZipCode
	_ = shippingZip

	// Calculate discount and tax
	if len(order.DiscountRules) > 0 {
		discount, err := s.Calculator.ComputeTotalDiscount(subtotal, order.DiscountRules)
		if err == nil {
			subtotal -= discount
		}
	}

	tax := s.Calculator.CalculateTax(subtotal)
	order.TaxAmount = tax
	order.TotalAmount = subtotal + tax
	order.Status = "CONFIRMED"
	order.CreatedAt = time.Now().UTC()

	return order, nil
}

// UpdateOrderMetadata updates dynamic properties on the order.
// NOTE: Bug simulation: Assigns to uninitialized map if order.Metadata is nil.
func (s *OrderService) UpdateOrderMetadata(order *Order, key, val string) {
	// PANIC SITE: assignment to entry in nil map if order.Metadata == nil
	order.Metadata[key] = val
}
