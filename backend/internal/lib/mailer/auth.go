package mailer

import (
	"fmt"
	"net/smtp"
	"slices"
	"strings"
)

const (
	mechanismPlain = "PLAIN"
	mechanismLogin = "LOGIN"
)

var (
	loginUsernamePrompts = []string{"username:", "user name"}
	loginPasswordPrompts = []string{"password:", "password"}
)

type credentials struct {
	username               string
	password               string
	isUnencryptedAuthOptIn bool
}

// plainAuth replaces smtp.PlainAuth, which decides by host name whether an unencrypted
// connection is acceptable. Here the operator's security mode decides.
type plainAuth struct {
	credentials
}

type loginAuth struct {
	credentials
}

func chooseAuth(offeredMechanisms string, userCredentials credentials) (smtp.Auth, string, error) {
	mechanisms := strings.Fields(strings.ToUpper(offeredMechanisms))

	if slices.Contains(mechanisms, mechanismPlain) {
		return &plainAuth{userCredentials}, mechanismPlain, nil
	}

	if slices.Contains(mechanisms, mechanismLogin) {
		return &loginAuth{userCredentials}, mechanismLogin, nil
	}

	return nil, "", ErrAuthMechanismNotOffered
}

func (c credentials) checkConnectionEncryption(server *smtp.ServerInfo) error {
	if !server.TLS && !c.isUnencryptedAuthOptIn {
		return ErrUnencryptedAuth
	}

	return nil
}

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if err := a.checkConnectionEncryption(server); err != nil {
		return "", nil, err
	}

	return mechanismPlain, []byte("\x00" + a.username + "\x00" + a.password), nil
}

func (a *plainAuth) Next(_ []byte, isMoreExpected bool) ([]byte, error) {
	if isMoreExpected {
		return nil, fmt.Errorf("unexpected server challenge during %s authentication", mechanismPlain)
	}

	return nil, nil
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if err := a.checkConnectionEncryption(server); err != nil {
		return "", nil, err
	}

	return mechanismLogin, nil, nil
}

func (a *loginAuth) Next(serverPrompt []byte, isMoreExpected bool) ([]byte, error) {
	if !isMoreExpected {
		return nil, nil
	}

	normalizedPrompt := strings.ToLower(strings.TrimRight(string(serverPrompt), "\x00"))

	switch {
	case slices.Contains(loginUsernamePrompts, normalizedPrompt):
		return []byte(a.username), nil
	case slices.Contains(loginPasswordPrompts, normalizedPrompt):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownLoginPrompt, serverPrompt)
	}
}
