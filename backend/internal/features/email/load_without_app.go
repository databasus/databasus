//go:build !testhelpers

package email

import "databasus-backend/internal/lib/mailer"

func loadWithoutApp() mailer.Sender {
	panic("email.GetInstanceSender is called before app.New installed the instance sender")
}
