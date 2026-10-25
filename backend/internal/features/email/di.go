// Package email hands the instance sender that internal/app built to the code that has not moved into
// internal/modules yet. A test binary built with the testhelpers tag builds it on first use instead.
package email

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/lib/mailer"
)

var installedSender atomic.Pointer[mailer.InstanceSender]

var getInstanceSenderWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(instanceSender *mailer.InstanceSender) {
	installedSender.Store(instanceSender)
}

func GetInstanceSender() *mailer.InstanceSender {
	if instanceSender := installedSender.Load(); instanceSender != nil {
		return instanceSender
	}

	return getInstanceSenderWithoutApp()
}
