package mailer

type Security string

const (
	SecurityTLS      Security = "tls"
	SecurityStartTLS Security = "starttls"
	SecurityNone     Security = "none"
)

func (s Security) IsValid() bool {
	switch s {
	case SecurityTLS, SecurityStartTLS, SecurityNone:
		return true
	default:
		return false
	}
}

const implicitTLSPort = 465

func GetDefaultSecurityForPort(port int) Security {
	if port == implicitTLSPort {
		return SecurityTLS
	}

	return SecurityStartTLS
}
