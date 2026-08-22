// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"fmt"
	"time"
)

// EmailNotification represents an outgoing email message.
type EmailNotification struct {
	Recipient string
	Subject   string
	Body      string
	SentAt    time.Time
}

// Mailer abstracts outbound communication.
type Mailer struct {
	SMTPServer string
	SenderName string
}

func NewMailer(smtpServer string) *Mailer {
	return &Mailer{
		SMTPServer: smtpServer,
		SenderName: "Triage Store Notifier",
	}
}

func (m *Mailer) SendOrderConfirmation(email, orderID string, amount float64) (*EmailNotification, error) {
	if email == "" {
		return nil, fmt.Errorf("recipient email is required")
	}

	return &EmailNotification{
		Recipient: email,
		Subject:   fmt.Sprintf("Order Confirmation #%s", orderID),
		Body:      fmt.Sprintf("Thank you for your purchase of $%.2f", amount),
		SentAt:    time.Now().UTC(),
	}, nil
}
