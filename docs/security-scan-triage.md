# Security scan triage

Security scans remain blocking. Exceptions identify one finding and explain the behavior that is
intentionally supported; they do not disable a rule for a directory or an entire test suite.

## Configurable notifier destinations

CodeQL alerts [150](https://github.com/databasus/databasus/security/code-scanning/150),
[151](https://github.com/databasus/databasus/security/code-scanning/151),
[152](https://github.com/databasus/databasus/security/code-scanning/152) and
[153](https://github.com/databasus/databasus/security/code-scanning/153), all `go/request-forgery`,
cover intentional requests to the Discord, Mattermost, Microsoft Teams and Telegram destinations
configured by workspace members. They are dismissed as accepted behavior. Private network
destinations remain supported for self-hosted Mattermost and Telegram Bot API servers.

Creating, changing or testing a notifier requires workspace membership with permission to manage
notifiers. Unauthenticated callers are refused; callers from another workspace cannot find the
notifier; viewers cannot test saved credentials. Omitted credentials are recovered only when the
destination, and the proxy where relevant, still matches the saved resource address. Changing that
address requires the caller to supply credentials again.

The notifier API tests verify these access and credential boundaries. Run them with:

```sh
go -C backend test -count=1 ./internal/modules/notifiers
```

This decision accepts authenticated workspace members choosing outbound destinations reachable by
the server. It does not assert that those destinations are public or trusted. Operators control
server egress according to their deployment's network policy.

## Notification email content

Notification headings and bodies are plain text. The email provider escapes body text before
embedding it in HTML and preserves line breaks. Untrusted notification content remains literal
text. Authentication email templates may supply trusted HTML; this does not make notification
text trusted.

CodeQL alert [149](https://github.com/databasus/databasus/security/code-scanning/149),
`go/email-injection`, also reports data flowing through parsed mail addresses and escaped body text.
The query's [configuration](https://github.com/github/codeql/blob/main/go/ql/src/Security/CWE-640/EmailInjection.qll)
defines no sanitization barrier. Its remaining paths pass through address serialization and HTML
escaping followed by MIME encoding. They are false positives for the serialized addresses and
escaped notification text. Alert 149 is dismissed
as a false positive with this rationale and the regression coverage below.

Mailer and real SMTP integration tests cover address/header injection and literal notification
content. Run them with:

```sh
go -C backend test -count=1 ./internal/lib/mailer ./internal/modules/notifiers/internal/providers/email
```

No CodeQL rule or source path is excluded from analysis.
