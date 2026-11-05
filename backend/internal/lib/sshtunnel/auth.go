package sshtunnel

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func buildAuthMethods(settings Settings) ([]ssh.AuthMethod, error) {
	switch settings.AuthType {
	case AuthTypePassword:
		return []ssh.AuthMethod{ssh.Password(settings.Password)}, nil
	case AuthTypePrivateKey:
		signer, err := buildPrivateKeySigner(settings)
		if err != nil {
			return nil, err
		}

		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnknownAuthType, settings.AuthType)
}

func buildPrivateKeySigner(settings Settings) (ssh.Signer, error) {
	if settings.PrivateKeyPassphrase == "" {
		signer, err := ssh.ParsePrivateKey([]byte(settings.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse the SSH tunnel private key: %w", err)
		}

		return signer, nil
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase(
		[]byte(settings.PrivateKey),
		[]byte(settings.PrivateKeyPassphrase),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the SSH tunnel private key: %w", err)
	}

	return signer, nil
}
