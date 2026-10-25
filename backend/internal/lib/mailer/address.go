package mailer

import (
	"fmt"
	"net/mail"
)

const defaultSenderName = "Databasus"

func ParseSender(value string) (*mail.Address, error) {
	sender, err := mail.ParseAddress(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidSender, err)
	}

	if sender.Name == "" {
		sender.Name = defaultSenderName
	}

	return sender, nil
}

func ResolveSender(from, username, host string) (*mail.Address, error) {
	if from != "" {
		return ParseSender(from)
	}

	if isBareAddress(username) {
		return ParseSender(username)
	}

	return ParseSender("noreply@" + host)
}

func ParseRecipient(value string) (*mail.Address, error) {
	recipient, err := mail.ParseAddress(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRecipient, err)
	}

	if recipient.Name != "" {
		return nil, ErrInvalidRecipient
	}

	return recipient, nil
}

func isBareAddress(value string) bool {
	address, err := mail.ParseAddress(value)

	return err == nil && address.Name == "" && address.Address == value
}
