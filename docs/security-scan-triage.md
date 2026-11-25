# Security scan triage

Security scans remain blocking. Each exception identifies one finding and explains the behavior
that is intentionally supported. An exception does not disable a rule for a directory or an entire
test suite.

## Configurable notifier destinations

CodeQL alerts [150](https://github.com/databasus/databasus/security/code-scanning/150),
[151](https://github.com/databasus/databasus/security/code-scanning/151),
[152](https://github.com/databasus/databasus/security/code-scanning/152) and
[153](https://github.com/databasus/databasus/security/code-scanning/153) are all
`go/request-forgery`. They cover intentional requests to Discord, Mattermost, Microsoft Teams and
Telegram destinations that workspace members configure. They are dismissed as accepted behavior.
Private network destinations remain supported for self-hosted Mattermost and Telegram Bot API
servers.

Access to a notifier is limited as follows:

- Creating, changing or testing a notifier requires workspace membership with permission to manage
  notifiers.
- Unauthenticated callers are refused.
- Callers from another workspace cannot find the notifier.
- Viewers cannot test saved credentials.

Omitted credentials are recovered only when the destination still matches the saved resource
address. Where a proxy is relevant, it must match too. Changing that address requires the caller to
supply credentials again.

The notifier API tests verify these access and credential boundaries. Run them with:

```sh
go -C backend test -count=1 ./internal/modules/notifiers
```

This decision accepts that authenticated workspace members choose outbound destinations the server
can reach. It does not assert that those destinations are public or trusted. Operators control
server egress, which is the outbound traffic the server may send. They do so according to their
deployment's network policy.

## Notification email content

Notification headings and bodies are plain text. The email provider escapes body text before
embedding it in HTML, and it preserves line breaks. Untrusted notification content stays literal
text.

Authentication email templates may supply trusted HTML. This does not make notification text
trusted.

CodeQL alert [149](https://github.com/databasus/databasus/security/code-scanning/149),
`go/email-injection`, also reports data that flows through parsed mail addresses and escaped body
text. The query's [configuration](https://github.com/github/codeql/blob/main/go/ql/src/Security/CWE-640/EmailInjection.qll)
defines no sanitization barrier, so it does not treat serialization or escaping as making data
safe. Its remaining paths pass through address serialization and HTML escaping followed by MIME
encoding. For the serialized addresses and the escaped notification text, they are false positives.
Alert 149 is dismissed as a false positive with this rationale and the regression coverage below.

Mailer and real SMTP integration tests cover address and header injection, and they check that
notification content stays literal. Run them with:

```sh
go -C backend test -count=1 ./internal/lib/mailer ./internal/modules/notifiers/internal/providers/email
```

No CodeQL rule or source path is excluded from analysis.
