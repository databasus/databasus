// Package email hands the instance mailer that internal/app built to the code that has not moved into
// internal/modules yet. A test binary built with the testhelpers tag builds it on first use instead.
package email

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/lib/mailer"
)

type installedMailer struct {
	sender mailer.Sender
}

var installed atomic.Pointer[installedMailer]

var getInstanceSenderWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(sender mailer.Sender) {
	installed.Store(&installedMailer{sender: sender})
}

func GetInstanceSender() mailer.Sender {
	if installedSender := installed.Load(); installedSender != nil {
		return installedSender.sender
	}

	return getInstanceSenderWithoutApp()
}
