// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package fees

import "fmt"

// FeeSchedule defines baseline interchange rates.
type FeeSchedule struct {
	BasisPoints   int // e.g. 290 for 2.9%
	FixedFeeCents int // e.g. 30 for $0.30
}

// CalculateFeePerInstallment divides fixed fee evenly across installment count.
// NOTE: Bug simulation: Division by zero if installments is 0.
func CalculateFeePerInstallment(schedule *FeeSchedule, totalCents int, installments int) (int, error) {
	if schedule == nil {
		return 0, fmt.Errorf("fee schedule cannot be nil")
	}

	percentageFee := (totalCents * schedule.BasisPoints) / 10000
	totalFee := percentageFee + schedule.FixedFeeCents

	// PANIC SITE: Integer divide by zero if installments == 0
	feePerInstallment := totalFee / installments
	return feePerInstallment, nil
}
