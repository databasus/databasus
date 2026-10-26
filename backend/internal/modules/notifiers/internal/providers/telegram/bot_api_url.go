package telegram

import (
	"fmt"
	"net/url"
	"strings"
)

// validateBotAPIURL keeps a secret out of the URL, which is stored and returned in clear. A query
// or fragment would also break the path the bot token is appended to. It answers the reason as
// the detail of the refused setting.
func validateBotAPIURL(rawBotAPIURL string) (string, bool) {
	parsedBotAPIURL, err := url.Parse(rawBotAPIURL)
	if err != nil ||
		(parsedBotAPIURL.Scheme != "http" && parsedBotAPIURL.Scheme != "https") ||
		parsedBotAPIURL.Host == "" {
		return "must be a valid http/https URL with a host", false
	}

	if parsedBotAPIURL.User != nil {
		return "must not contain a username or password", false
	}

	// An empty query or fragment ("/telegram?") parses to empty fields, so the delimiters are checked
	// in the raw string. Outside user info, rejected above, they can only be those delimiters.
	if strings.ContainsAny(rawBotAPIURL, "?#") {
		return "must not contain a query or fragment", false
	}

	return "", true
}

// buildSendMessageURL concatenates rather than calling url.JoinPath, which keeps "./" and "../" in
// the prefix exactly as validated.
func buildSendMessageURL(botAPIURL, botToken string) string {
	return fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(botAPIURL, "/"), botToken)
}
