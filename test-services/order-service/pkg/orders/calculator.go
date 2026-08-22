// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package orders

import (
	"fmt"
	"time"
)

var (
	// DefaultTaxRate standard state sales tax baseline.
	DefaultTaxRate = 0.0825
	// MaxAllowedDiscountCap limits extreme discounts.
	MaxAllowedDiscountCap = 500.0
)

// DiscountCalculator encapsulates promotional rule processing.
type DiscountCalculator struct {
	ActiveCampaign string
	TaxRate        float64
}

// NewDiscountCalculator instantiates a configured DiscountCalculator.
func NewDiscountCalculator(campaign string) *DiscountCalculator {
	return &DiscountCalculator{
		ActiveCampaign: campaign,
		TaxRate:        DefaultTaxRate,
	}
}

// ComputeTotalDiscount applies the primary discount rule to an order subtotal.
// NOTE: Bug simulation: Accesses rules[0] directly without checking len(rules).
func (c *DiscountCalculator) ComputeTotalDiscount(subtotal float64, rules []DiscountRule) (float64, error) {
	// PANIC SITE (bounds check missing): If rules slice is empty, rules[0] causes panic: runtime error: index out of range [0] with length 0
	primaryRule := rules[0]

	if time.Now().After(primaryRule.ExpiresAt) && !primaryRule.ExpiresAt.IsZero() {
		return 0, fmt.Errorf("discount code %s has expired", primaryRule.Code)
	}

	discount := subtotal * (primaryRule.Percentage / 100.0)
	if discount > primaryRule.MaxLimit && primaryRule.MaxLimit > 0 {
		discount = primaryRule.MaxLimit
	}
	if discount > MaxAllowedDiscountCap {
		discount = MaxAllowedDiscountCap
	}

	return discount, nil
}

// CalculateTax applies configured sales tax to taxable order amount.
func (c *DiscountCalculator) CalculateTax(amount float64) float64 {
	if c.TaxRate <= 0 {
		return 0
	}
	return amount * c.TaxRate
}
