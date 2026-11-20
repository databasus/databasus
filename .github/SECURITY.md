# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Databasus, please report it privately. **Do not create a public GitHub issue for security vulnerabilities.** Security reports take priority over other work.

### How to Report

1. **GitHub Security Advisories** (preferred): use [private vulnerability reporting](https://github.com/databasus/databasus/security/advisories/new)
2. **Email**: send details to [info@databasus.com](mailto:info@databasus.com)
3. **Telegram**: contact [@rostislav_dugin](https://t.me/rostislav_dugin)

### What to Include

Copy this template into your report and fill in every section that applies.

```markdown
## Databasus version (screenshot)

It is displayed in the bottom left corner of the Databasus UI. Please attach a screenshot, not just the version text.

## Deployment

<!-- e.g. Docker on Ubuntu 22.04 x64, Helm chart on Kubernetes, macOS 14 ARM -->

## Database type and version (optional, for DB-related issues)

<!-- e.g. PostgreSQL 17 in Docker, MySQL 8.4 installed on server, MongoDB 7 in Atlas -->

## Describe the vulnerability (please write manually, do not ask AI to summarize)

**What an attacker can do:**

**What access the attacker needs:**
<!-- e.g. none, network access to the UI, a workspace viewer account, access to the backup storage -->

## Steps to reproduce

1.
2.
3.

## Impact and severity

<!-- What data or systems are affected; your severity estimate, e.g. a CVSS vector -->

## Suggested fix (optional)

## Additional context / logs

<!-- Proof of concept, screenshots, relevant log output. Remove secrets and real credentials. -->
```

## Response Timeline

- **Acknowledgment**: within 48-72 hours
- **Initial assessment**: within 1 week
- **Fix**: depends on severity; critical issues are addressed first

We follow a coordinated disclosure policy. Please give us reasonable time to release a fix before any public disclosure. Once a fix is released, we publish a GitHub Security Advisory, with a CVE identifier when one is assigned.

## Published Advisories

Fixed vulnerabilities, with their advisories and CVEs, are listed on the [Security page of the repository](https://github.com/databasus/databasus/security).

## Supported Versions

| Version | Supported |
| ------- | --------- |
| Latest  | Yes       |

Security fixes are released only in the latest version of Databasus, so please keep your installation up to date. The database engines and versions Databasus supports are listed in the [README](../README.md#-supported-databases).

## Security Features

Databasus is designed with security in mind. The [README](../README.md#-enterprise-grade-security-docs) lists the security features, and the [security documentation](https://databasus.com/security) describes them and the release checks in detail.

## License

Databasus is licensed under [Apache 2.0](../LICENSE).
