package apptest

import (
	"context"
	"sync"
)

// MailMessage is a message the test application sent.
type MailMessage struct {
	To       string
	Subject  string
	HTMLBody string
}

// MailRecorder stands in for the operator's mail server: it keeps every message the application
// sends, so a test reads a sign-in or reset code from it.
type MailRecorder struct {
	mutex           sync.Mutex
	messages        []MailMessage
	refusedMessages []MailMessage
	isConfigured    bool
	isRefusingSend  bool
}

func newMailRecorder(isConfigured bool) *MailRecorder {
	return &MailRecorder{
		mutex:           sync.Mutex{},
		messages:        nil,
		refusedMessages: nil,
		isConfigured:    isConfigured,
		isRefusingSend:  false,
	}
}

func (r *MailRecorder) IsConfigured() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.isConfigured
}

// SendEmail records the message, or refuses it after RefuseSends. Without a mail server it skips
// the message as the real sender does.
func (r *MailRecorder) SendEmail(_ context.Context, to, subject, htmlBody string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isConfigured {
		return nil
	}

	if r.isRefusingSend {
		r.refusedMessages = append(r.refusedMessages, MailMessage{To: to, Subject: subject, HTMLBody: htmlBody})

		return errMailServerRefused
	}

	r.messages = append(r.messages, MailMessage{To: to, Subject: subject, HTMLBody: htmlBody})

	return nil
}

// RefuseSends makes every later send fail, as a mail server that rejects the message does.
func (r *MailRecorder) RefuseSends() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.isRefusingSend = true
}

// GetMessagesTo returns the messages sent to the address, the oldest first.
func (r *MailRecorder) GetMessagesTo(address string) []MailMessage {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return filterMessagesTo(r.messages, address)
}

// GetRefusedMessagesTo returns the messages to the address that RefuseSends turned away, so a test
// tells a refused send from a send that never happened.
func (r *MailRecorder) GetRefusedMessagesTo(address string) []MailMessage {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return filterMessagesTo(r.refusedMessages, address)
}

func filterMessagesTo(messages []MailMessage, address string) []MailMessage {
	var addressedMessages []MailMessage

	for _, message := range messages {
		if message.To == address {
			addressedMessages = append(addressedMessages, message)
		}
	}

	return addressedMessages
}

func (r *MailRecorder) GetMessageCount() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.messages)
}
