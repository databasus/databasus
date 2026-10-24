//go:build testhelpers

package workspaces_testing

import (
	"context"
	"errors"
)

type MockEmailSender struct {
	SendEmailCalls []EmailCall
	ShouldFail     bool
}

type EmailCall struct {
	To      string
	Subject string
	Body    string
}

func NewMockEmailSender() *MockEmailSender {
	return &MockEmailSender{
		SendEmailCalls: []EmailCall{},
		ShouldFail:     false,
	}
}

func (m *MockEmailSender) SendEmail(_ context.Context, to, subject, body string) error {
	m.SendEmailCalls = append(m.SendEmailCalls, EmailCall{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	if m.ShouldFail {
		return errors.New("mock email send failure")
	}
	return nil
}
