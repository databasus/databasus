package mailer

import "errors"

var (
	ErrUnknownSecurity         = errors.New("unknown SMTP security mode, expected tls, starttls or none")
	ErrStartTLSNotOffered      = errors.New("SMTP server does not offer STARTTLS")
	ErrAuthMechanismNotOffered = errors.New("SMTP server offers neither PLAIN nor LOGIN authentication")
	ErrUnencryptedAuth         = errors.New("refusing to authenticate over an unencrypted SMTP connection")
	ErrUnknownLoginPrompt      = errors.New("unexpected LOGIN authentication prompt from SMTP server")
	ErrSessionTimeout          = errors.New("SMTP session timed out")
	ErrInvalidSender           = errors.New("sender must be an email address, optionally preceded by a display name")
	ErrInvalidRecipient        = errors.New("recipient must be a single email address")
	ErrInvalidHeloName         = errors.New(
		"greeting name must be a host name or an address literal such as [192.0.2.1]",
	)
)
