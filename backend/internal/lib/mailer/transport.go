package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

const sessionTimeout = 30 * time.Second

type Connection struct {
	Host                 string
	Port                 int
	Security             Security
	Username             string
	Password             string
	HeloName             string
	IsInsecureSkipVerify bool
}

func Send(ctx context.Context, connection Connection, message Message) error {
	if !connection.Security.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnknownSecurity, connection.Security)
	}

	recipient, err := ParseRecipient(message.Recipient)
	if err != nil {
		return err
	}

	content, err := buildMessage(message.Sender, recipient, message.Subject, message.HTMLBody, time.Now().UTC())
	if err != nil {
		return err
	}

	sessionCtx, cancel := context.WithTimeout(ctx, sessionTimeout)
	defer cancel()

	err = runSession(sessionCtx, connection, message.Sender.Address, recipient.Address, content)
	if err != nil && (sessionCtx.Err() != nil || errors.Is(err, os.ErrDeadlineExceeded)) {
		return fmt.Errorf("%w: %w", ErrSessionTimeout, err)
	}

	return err
}

func runSession(
	ctx context.Context,
	connection Connection,
	envelopeSender, envelopeRecipient string,
	content []byte,
) error {
	conn, err := dial(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		if err := conn.SetDeadline(deadline); err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to set SMTP session deadline: %w", err)
		}
	}

	stopInterruptOnCancel := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
	defer stopInterruptOnCancel()

	client, err := smtp.NewClient(conn, connection.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to read SMTP server greeting: %w", err)
	}

	if err := openSession(client, connection); err != nil {
		_ = client.Close()
		return err
	}

	if err := deliver(client, envelopeSender, envelopeRecipient, content); err != nil {
		_ = client.Close()
		return err
	}

	_ = client.Quit()

	return nil
}

func dial(ctx context.Context, connection Connection) (net.Conn, error) {
	address := net.JoinHostPort(connection.Host, strconv.Itoa(connection.Port))

	if connection.Security == SecurityTLS {
		tlsDialer := &tls.Dialer{Config: buildTLSConfig(connection)}
		return tlsDialer.DialContext(ctx, "tcp", address)
	}

	var dialer net.Dialer

	return dialer.DialContext(ctx, "tcp", address)
}

func openSession(client *smtp.Client, connection Connection) error {
	heloName := connection.HeloName
	if heloName == "" {
		heloName = fallbackHeloName
	}

	if err := client.Hello(heloName); err != nil {
		return fmt.Errorf("SMTP greeting failed: %w", err)
	}

	if connection.Security == SecurityStartTLS {
		if isOffered, _ := client.Extension("STARTTLS"); !isOffered {
			return ErrStartTLSNotOffered
		}

		if err := client.StartTLS(buildTLSConfig(connection)); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	if connection.Username == "" || connection.Password == "" {
		return nil
	}

	_, offeredMechanisms := client.Extension("AUTH")

	auth, mechanism, err := chooseAuth(offeredMechanisms, credentials{
		connection.Username,
		connection.Password,
		connection.Security == SecurityNone,
	})
	if err != nil {
		return err
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication with %s failed: %w", mechanism, err)
	}

	return nil
}

func deliver(client *smtp.Client, envelopeSender, envelopeRecipient string, content []byte) error {
	if err := client.Mail(envelopeSender); err != nil {
		return fmt.Errorf("SMTP server rejected the sender: %w", err)
	}

	if err := client.Rcpt(envelopeRecipient); err != nil {
		return fmt.Errorf("SMTP server rejected the recipient: %w", err)
	}

	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP server refused the message: %w", err)
	}

	if _, err := dataWriter.Write(content); err != nil {
		_ = dataWriter.Close()
		return fmt.Errorf("failed to send the message: %w", err)
	}

	if err := dataWriter.Close(); err != nil {
		return fmt.Errorf("SMTP server rejected the message: %w", err)
	}

	return nil
}

func buildTLSConfig(connection Connection) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         connection.Host,
		InsecureSkipVerify: connection.IsInsecureSkipVerify,
	}
}
