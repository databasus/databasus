//go:build testhelpers

package users_testing

import (
	"context"
	"errors"
	"sync"
)

type MockEmailSender struct {
	SentEmails          []EmailCall
	ShouldFail          bool
	IsMailServerMissing bool

	// Concurrent sign-in tests send from several requests at once.
	sentEmailsMutex sync.Mutex
}

type EmailCall struct {
	To      string
	Subject string
	Body    string
}

func NewMockEmailSender() *MockEmailSender {
	return &MockEmailSender{
		SentEmails: []EmailCall{},
		ShouldFail: false,
	}
}

func (m *MockEmailSender) IsConfigured() bool {
	return !m.IsMailServerMissing
}

func (m *MockEmailSender) SendEmail(_ context.Context, to, subject, body string) error {
	m.sentEmailsMutex.Lock()
	defer m.sentEmailsMutex.Unlock()

	m.SentEmails = append(m.SentEmails, EmailCall{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	if m.ShouldFail {
		return errors.New("mock email send failure")
	}
	return nil
}
