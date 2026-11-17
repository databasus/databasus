## ADDED Requirements

### Requirement: Recovery uploads remain private and bounded over any connection

Recovery uploads SHALL preserve the archive's encryption and the exact UTF-8 bytes of the entered master key without trimming or normalizing whitespace or line endings. The key and decrypted dump SHALL NOT appear in responses, URLs, logs, telemetry, browser persistent storage or cacheable content. Staging SHALL restrict file access to the runtime identity, keep memory use bounded and stage no plaintext dump. Archive uploads SHALL enforce a configurable finite byte limit; the default SHALL be 10 GiB. The UTF-8 text-key field SHALL be limited to 64 KiB and SHALL be write-only. Oversized uploads, insufficient disk space and disconnected uploads SHALL fail without replacing the active database or key and SHALL release partial private files. Private staged key copies SHALL be removed after failed, cancelled or expired preparation or completed publication, while inputs needed for interrupted accepted publication SHALL survive a restart. Recovery SHALL NOT delete the operator's original mounted input files or the installed master key as part of staging cleanup.

The interface SHALL upload the archive and the text key over the connection the page uses, HTTP included, without a separate warning or acknowledgment, and the server SHALL accept such an upload.

#### Scenario: Upload exceeds the configured limit without a declared length

- **WHEN** an archive upload without a usable declared length exceeds the configured byte limit
- **THEN** the server refuses it while streaming and releases its partial staging files without replacing active state

#### Scenario: Owner imports over plain HTTP

- **WHEN** root admin selects a recovery archive and enters the key on a page served over HTTP
- **THEN** the upload starts without a warning or an acknowledgment and the server accepts it

## REMOVED Requirements

### Requirement: Uploaded recovery material remains private and bounded

**Reason**: Its HTTP warning and the server's refusal of an unacknowledged upload over HTTP are removed. The rest of the requirement moves unchanged into "Recovery uploads remain private and bounded over any connection".

**Migration**: None. Uploads no longer send `isInsecureTransferAcknowledged`.
