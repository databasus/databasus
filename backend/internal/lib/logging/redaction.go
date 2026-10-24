package logging

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

const redactedValue = "***"

var (
	// "webhook" belongs here rather than with the URL keys: an incoming-webhook URL is itself the
	// credential, and its secret sits in the path, where stripping userinfo would not reach it.
	secretKeyParts = []string{
		"password", "passwd", "secret", "token", "api_key", "apikey",
		"authorization", "cookie", "credential", "private_key", "webhook",
	}
	urlKeyParts   = []string{"url", "uri", "dsn", "endpoint", "conn"}
	emailKeyParts = []string{"email", "mail"}

	urlCredentialsPattern = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*)://[^\s:/@]+:[^\s@]*@`)
	keyValueSecretPattern = regexp.MustCompile(
		`(?i)\b(password|passwd|secret|token|api[_-]?key|authorization)\s*=\s*\S+`,
	)
	// Header-shaped secrets use a colon, not "=", so the key=value pattern never sees them. The auth
	// scheme is kept because knowing which one was sent is diagnostic; the credential is not.
	headerSecretPattern = regexp.MustCompile(
		`(?i)\b(password|passwd|secret|token|api[_-]?key|authorization)(\s*:\s*)` +
			`((?:bearer|basic|digest|token)\s+)?(\S+)`,
	)
	emailPattern = regexp.MustCompile(`[\w.+-]+@[\w-]+(?:\.[\w-]+)+`)
)

func redactAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()

	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()

		redactedGroup := make([]slog.Attr, 0, len(group))
		for _, member := range group {
			redactedGroup = append(redactedGroup, redactAttr(member))
		}

		attr.Value = slog.GroupValue(redactedGroup...)

		return attr
	}

	key := strings.ToLower(attr.Key)

	// Numbers keep their type even under a secret-looking key: "token_count" is a diagnostic, not a
	// secret, and blanking it loses information without protecting anything.
	if hasAnyPart(key, secretKeyParts) && !isNumericLike(attr.Value) {
		attr.Value = slog.StringValue(redactedValue)

		return attr
	}

	// "error" is the most common attr key in this codebase and errors are KindAny, so without this
	// an error text carrying a DSN - url.Error formats the whole URL, password included - would
	// reach the file and the remote backend untouched.
	if wrapped, isError := attr.Value.Any().(error); isError {
		attr.Value = slog.StringValue(redactMessage(wrapped.Error()))

		return attr
	}

	isText := attr.Value.Kind() == slog.KindString

	switch {
	// These keys are semantically textual, so rendering a non-string value to a scrubbed string is
	// safe. Numeric look-alikes such as "connection_count" or "endpoint_timeout" are not: they
	// match the same substring lists, and retyping them would corrupt the record.
	case hasAnyPart(key, urlKeyParts) && (isText || isRenderableText(attr.Value)):
		attr.Value = slog.StringValue(stripURLCredentials(attr.Value.String()))
	case hasAnyPart(key, emailKeyParts) && (isText || isRenderableText(attr.Value)):
		attr.Value = slog.StringValue(maskEmails(attr.Value.String()))
	case isText:
		attr.Value = slog.StringValue(redactMessage(attr.Value.String()))
	// A struct or map handed to slog.Any matches none of the arms above, so without this a DTO or a
	// header map carrying a password would reach the sinks verbatim. The rendered form is kept only
	// when it actually held a secret: rendering costs the typed value, and this runs on every record.
	case attr.Value.Kind() == slog.KindAny:
		rendered := fmt.Sprintf("%+v", attr.Value.Any())

		if scrubbed := redactMessage(rendered); scrubbed != rendered {
			attr.Value = slog.StringValue(scrubbed)
		}
	}

	return attr
}

// The entry point for callers that persist a provider error outside a log record,
// where the sink's redaction never runs but the same rules still have to hold.
func Redact(message string) string {
	return redactMessage(message)
}

// Credentials and addresses reach the message through fmt.Sprintf at call sites - audit messages
// in particular embed raw user emails, and those messages are now exported off-box.
// The strings.Contains guards keep three regex scans off the hot path: this runs on every record,
// and the overwhelming majority of messages contain none of these markers.
func redactMessage(message string) string {
	if strings.Contains(message, "://") {
		message = urlCredentialsPattern.ReplaceAllString(message, "$1://"+redactedValue+":"+redactedValue+"@")
	}

	if strings.Contains(message, "=") {
		message = keyValueSecretPattern.ReplaceAllString(message, "$1="+redactedValue)
	}

	if strings.Contains(message, ":") {
		message = redactHeaderSecrets(message)
	}

	return maskEmails(message)
}

// A purely numeric value is a count, not a credential - the same carve-out redactAttr makes for
// "token_count" - so "replication token: 5 slots" survives intact. Sentence punctuation is split
// off first, because \S+ swallows it and would make "token: 5," look non-numeric.
func redactHeaderSecrets(message string) string {
	return headerSecretPattern.ReplaceAllStringFunc(message, func(match string) string {
		groups := headerSecretPattern.FindStringSubmatch(match)
		secretKey, separator, authScheme, value := groups[1], groups[2], groups[3], groups[4]

		value, trailingPunctuation := splitTrailingPunctuation(value)

		if isAllDigits(value) {
			return match
		}

		return secretKey + separator + authScheme + redactedValue + trailingPunctuation
	})
}

func splitTrailingPunctuation(value string) (string, string) {
	trimmed := strings.TrimRight(value, ".,;:!?)]}")

	return trimmed, value[len(trimmed):]
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}

	return strings.IndexFunc(value, func(character rune) bool {
		return character < '0' || character > '9'
	}) < 0
}

func hasAnyPart(key string, parts []string) bool {
	for _, part := range parts {
		if strings.Contains(key, part) {
			return true
		}
	}

	return false
}

func isNumericLike(value slog.Value) bool {
	switch value.Kind() {
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool,
		slog.KindDuration, slog.KindTime:
		return true
	case slog.KindAny, slog.KindString, slog.KindGroup, slog.KindLogValuer:
		return false
	default:
		return false
	}
}

func isRenderableText(value slog.Value) bool {
	if value.Kind() != slog.KindAny {
		return false
	}

	_, isStringer := value.Any().(fmt.Stringer)

	return isStringer
}

// stripURLCredentials keeps the host readable instead of blanking the whole value - an endpoint
// nobody can identify is useless when diagnosing where logs are going.
func stripURLCredentials(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.User == nil {
		// Not a bare URL, so fall through to the full message scrub rather than the key=value
		// pattern alone - that pattern does not match userinfo, and a value like
		// "connecting to postgres://admin:pw@db/app" would otherwise pass through intact.
		return redactMessage(value)
	}

	// A literal "***" would be percent-encoded by url.String(); this marker survives round-tripping
	// while still showing that credentials were present.
	parsed.User = url.User("redacted")

	return parsed.String()
}

func maskEmails(value string) string {
	if !strings.Contains(value, "@") {
		return value
	}

	return emailPattern.ReplaceAllStringFunc(value, func(address string) string {
		localPart, domain, isFound := strings.Cut(address, "@")
		if !isFound || localPart == "" {
			return address
		}

		return localPart[:1] + redactedValue + "@" + domain
	})
}
