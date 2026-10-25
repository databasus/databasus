package identity

import "net/mail"

// IsEmailAddress accepts a bare address such as owner@example.com and refuses a display name, a
// group or a login such as admin. Sign-up, the profile, the test email and the second-factor gate
// all ask it, so they never disagree about what an address is, and the mail transport parses
// addresses the same way.
func IsEmailAddress(value string) bool {
	parsedAddress, err := mail.ParseAddress(value)

	return err == nil && parsedAddress.Address == value
}
