package shared

import (
	"errors"
	"fmt"
)

type SslMode string

const (
	SslModeDisable    SslMode = "disable"
	SslModeRequire    SslMode = "require"
	SslModeVerifyCA   SslMode = "verify-ca"
	SslModeVerifyFull SslMode = "verify-full"
)

// TLS holds the SSL mode and the PEM certificates and key in clear text.
type TLS struct {
	Mode       SslMode
	ClientCert string
	ClientKey  string
	RootCert   string
}

// Validate is the SSL rule of logical and physical sources alike: a known mode, the client
// certificate and key together, and certificates only when SSL is enabled. It reads only whether a
// value is set, so it holds for encrypted values too.
func (t TLS) Validate() error {
	switch t.Mode {
	case SslModeDisable, SslModeRequire, SslModeVerifyCA, SslModeVerifyFull:
	default:
		return fmt.Errorf("invalid SSL mode: %s", t.Mode)
	}

	hasClientCert := t.ClientCert != ""
	hasClientKey := t.ClientKey != ""

	if hasClientCert != hasClientKey {
		return errors.New("client certificate and client key must be provided together")
	}

	if t.Mode == SslModeDisable && (hasClientCert || hasClientKey || t.RootCert != "") {
		return errors.New(
			"SSL certificates require SSL to be enabled (set SSL mode to require, verify-ca or verify-full)",
		)
	}

	return nil
}

func (t TLS) getModeOrDefault() SslMode {
	if t.Mode == "" {
		return SslModeDisable
	}

	return t.Mode
}
