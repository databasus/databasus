package mailer

import (
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const (
	fallbackHeloName  = "localhost"
	maxHeloNameLength = 255
	ipv6LiteralPrefix = "IPv6:"
)

var hostNamePattern = regexp.MustCompile(
	`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)*$`,
)

func ValidateHeloName(name string) error {
	if len(name) > maxHeloNameLength {
		return ErrInvalidHeloName
	}

	if hostNamePattern.MatchString(name) {
		return nil
	}

	literal, isBracketed := strings.CutPrefix(name, "[")
	literal, hasClosingBracket := strings.CutSuffix(literal, "]")
	if !isBracketed || !hasClosingBracket {
		return ErrInvalidHeloName
	}

	if ipv6Literal, isIPv6Literal := strings.CutPrefix(literal, ipv6LiteralPrefix); isIPv6Literal {
		ip := net.ParseIP(ipv6Literal)
		if ip == nil || ip.To4() != nil {
			return ErrInvalidHeloName
		}

		return nil
	}

	ip := net.ParseIP(literal)
	if ip == nil || ip.To4() == nil || strings.Contains(literal, ":") {
		return ErrInvalidHeloName
	}

	return nil
}

// GetDefaultHeloName prefers the public host so that mail servers checking the greeting against
// DNS see the name the instance is reached by, rather than a container ID.
func GetDefaultHeloName(databasusURL string) string {
	if databasusURL != "" {
		if parsedURL, err := url.Parse(databasusURL); err == nil {
			if heloName := formatHeloName(parsedURL.Hostname()); ValidateHeloName(heloName) == nil {
				return heloName
			}
		}
	}

	if hostname, err := os.Hostname(); err == nil {
		if heloName := formatHeloName(hostname); ValidateHeloName(heloName) == nil {
			return heloName
		}
	}

	return fallbackHeloName
}

func formatHeloName(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}

	if ipv4 := ip.To4(); ipv4 != nil {
		return "[" + ipv4.String() + "]"
	}

	return "[" + ipv6LiteralPrefix + ip.String() + "]"
}
