package totp

import (
	"net/url"
	"strconv"
)

// BuildAddress is the otpauth://totp/ address an app reads from a QR code. It names the issuer and
// the account, and spells out the algorithm, digits and period, so no app falls back to a default
// of its own.
func BuildAddress(issuer, accountName, secret string) string {
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", strconv.Itoa(codeDigitCount))
	query.Set("period", strconv.Itoa(stepSeconds))

	address := url.URL{
		Scheme:   "otpauth",
		Host:     "totp",
		Path:     "/" + issuer + ":" + accountName,
		RawQuery: query.Encode(),
	}

	return address.String()
}
