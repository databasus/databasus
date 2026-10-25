package mailer

import (
	"context"
	"log/slog"
)

// Settings are the operator's SMTP server for the instance's own mail. An empty Host means the
// operator configured no server.
type Settings struct {
	Host                 string
	Port                 int
	Security             Security
	Username             string
	Password             string
	From                 string
	HeloName             string
	IsInsecureSkipVerify bool
}

type Deps struct {
	Settings Settings
	Logger   *slog.Logger
}

// InstanceSender sends the instance's own mail, such as sign-in codes and invitations, through the
// operator's server. Notification channels carry their own server and call Send instead.
type InstanceSender struct {
	settings Settings
	logger   *slog.Logger
}

func NewInstanceSender(deps Deps) *InstanceSender {
	return &InstanceSender{settings: deps.Settings, logger: deps.Logger}
}

func (s *InstanceSender) IsConfigured() bool {
	return s.settings.Host != ""
}

// SendEmail skips the message with a warning when no server is configured, so a flow that mails a
// code still completes on an instance without SMTP.
func (s *InstanceSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	if !s.IsConfigured() {
		s.logger.WarnContext(ctx, "skipping email send, SMTP is not configured",
			slog.String("to_email", to), slog.String("subject", subject))

		return nil
	}

	sender, err := ResolveSender(s.settings.From, s.settings.Username, s.settings.Host)
	if err != nil {
		return err
	}

	return Send(ctx, Connection{
		Host:                 s.settings.Host,
		Port:                 s.settings.Port,
		Security:             s.settings.Security,
		Username:             s.settings.Username,
		Password:             s.settings.Password,
		HeloName:             s.settings.HeloName,
		IsInsecureSkipVerify: s.settings.IsInsecureSkipVerify,
	}, Message{
		Sender:    sender,
		Recipient: to,
		Subject:   subject,
		HTMLBody:  htmlBody,
	})
}
