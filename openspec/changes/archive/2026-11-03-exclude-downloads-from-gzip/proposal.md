## Why

The server applies HTTP gzip to backup downloads even though their payloads are already compressed, adding CPU work for the duration of each transfer. Logical backup downloads also omit their known byte length, preventing browsers from showing total download size; both problems should be addressed before release.

## What Changes

- Exclude logical backup files, physical restore sets and verification-agent binaries from HTTP gzip, including when the client accepts gzip.
- Keep the existing compression behavior for JSON responses and static assets, including existing extension exclusions.
- Return the exact `Content-Length` of a successful logical backup download, including a backup encrypted in storage, using the recorded size of its compressed format before encryption.
- Preserve the exact byte length already supplied by the agent-binary handler. Keep physical restore sets streaming without requiring a length in advance.
- Document these response guarantees in the user API contract and add regression coverage that observes the encoded HTTP response.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `api-http-server`: Exclude the three binary download operations from HTTP gzip while preserving compression negotiation and existing static-asset behavior.
- `logical-backups`: Announce the exact byte length of a downloaded logical backup after decryption, preserving its compressed file format.

## Impact

- Governed by the root `AGENTS.md` and `backend/AGENTS.md`.
- Backend wiring and middleware: `backend/internal/app/server.go` and `backend/internal/lib/httpserver/compress.go`.
- Logical download service and HTTP adapter: `backend/internal/modules/backups/internal/download/` and `backend/internal/modules/backups/internal/httpapi/download_handler.go`.
- Contract: response descriptions in `backend/api/openapi.yaml`, followed by API generation and its consistency checks.
- Tests: logical download API tests, physical restore-set API tests, system agent-download tests and HTTP compression middleware tests.
- No new dependencies, database migrations, request-shape changes or verification-agent client changes. Existing installed agents continue to use the same discovery and download paths.
- Agent binaries may compress well; excluding them deliberately trades additional transfer bytes for avoiding server-side compression during downloads. No throughput or CPU improvement has been measured.

## Out of Scope

- Calculating a complete physical restore-set length before streaming.
- Changing backup compression, encryption, storage formats or persisted size units.
- Changing tokens, access checks, download concurrency, download filenames or archive contents.
- Adding range requests, resumable downloads, proxy configuration, frontend progress controls or compression settings.
- A general content-type-aware compression framework or performance benchmark suite.
