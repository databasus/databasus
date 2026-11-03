## Context

See `proposal.md` for motivation and scope. This design is required because compression policy spans the application router, shared HTTP middleware and several download operations, and because the logical download length needs a decision about existing size metadata.

- `backend/internal/app/server.go:122` applies `httpserver.Compress` to every route. `backend/internal/lib/httpserver/compress.go:9` excludes only URL extensions; the three download URLs have no excluded extension.
- `backend/go.mod:12` pins `github.com/gin-contrib/gzip` to v1.2.3. Its `WithExcludedPathsRegexs` option preserves the built-in request checks. Its custom predicate replaces those checks, including negotiation through `Accept-Encoding`.
- `backend/internal/modules/backups/internal/httpapi/download_handler.go:48` supplies `ContentLength: 0`. The download result at `backend/internal/modules/backups/internal/download/service.go:65` carries only the reader and filename.
- `backend/internal/modules/backups/internal/pipeline/backup.go:254` counts bytes after backup compression and before encryption. At line 318, that count becomes a floating-point megabyte value by division by `1024 * 1024`. `backend/internal/modules/backups/internal/logical/backup_run_service.go:113` publishes it as `BackupSizeMb`; `backend/internal/modules/backups/internal/logical/backup_repository.go:196` stores it without decimal rounding. The column is `DOUBLE PRECISION` in `backend/migrations/20250605090323_init.sql:195`.
- `backend/internal/modules/backups/internal/pipeline/backup_file.go:12` decrypts a download without removing its stored compression format. The recorded pre-encryption size therefore describes the downloaded file.
- `backend/internal/modules/backups/internal/httpapi/physical_restore_handler.go:70` streams a generated tar archive without announcing its complete size. `backend/internal/app/systemhttpapi/binary_service.go:36` already obtains the agent binary's exact size through `Stat`.
- `backend/internal/agent/api/api.go:89` uses an ordinary Go HTTP client. Its default transport can request gzip and transparently decompress it, so an ordinary client response can hide the current compression.

The change follows root `AGENTS.md` and `backend/AGENTS.md`. In particular, `backend/AGENTS.md` keeps wiring in `internal/app`, domain logic inside its module, and domain knowledge out of `internal/lib`. The module boundaries and HTTP adapters described in `docs/architecture.md` under "Modules", "Shared code" and "HTTP API" remain unchanged. Its "Backup data flow" rule requires bounded streaming memory.

## Goals / Non-Goals

**Goals:** Apply a small route-specific compression policy, restore a known byte length, and verify the bytes and headers visible to HTTP clients.

**Non-Goals:** Do not introduce response buffering, storage metadata lookups, new persisted size fields or a generic content-type-aware response writer. The full out-of-scope list is in `proposal.md`.

## Decisions

### Configure exact download exclusions in the application router

Give `httpserver.Compress` a slice of excluded path regexes and apply it through `gzip.WithExcludedPathsRegexs`. Keep the existing extension exclusions. Supply the following expressions from `internal/app`, where the route-specific policy belongs:

```text
^/api/v1/logical-backups/[^/]+/file$
^/api/v1/physical-restore-sets$
^/api/v1/system/verification-agent$
```

The library matches `URL.Path`, so query parameters do not affect the exclusions. Anchors prevent neighboring operations from being excluded. The middleware continues to rely on its existing negotiation and connection-upgrade checks. A download operation's error response also bypasses gzip because the decision is made before the handler runs.

Rejected alternatives:

- Hardcoding these paths in `internal/lib/httpserver` would put domain-specific routing policy in shared code.
- Prefix exclusions could unintentionally stop compression for neighboring operations.
- A custom compression predicate would require duplicating the library's existing request checks.
- Deciding from response content type would require another response-writer abstraction because this middleware chooses compression before the handler writes headers.
- Disabling gzip globally would also remove useful compression from JSON and text static assets.

### Carry the existing pre-encryption length through the logical download service

Add `SizeBytes int64` to the download service's `Download` result. Compute it in the service from the already loaded backup as `int64(backup.BackupSizeMb * 1024 * 1024)`, then map it directly to the generated response's `ContentLength` in the handler. The handler remains a response adapter and performs no storage inspection.

The megabyte conversion divides and multiplies by a power of two. For integer byte counts through `2^53`, binary floating-point storage preserves the exact value; no decimal rounding is involved. The counter measures the bytes before encryption, so encrypted storage overhead must not be added. This uses the existing size convention, also used by physical artifact sizing, without a migration.

Rejected alternatives:

- Leaving the length absent would retain the browser's missing total size despite already having the length.
- Using the encrypted storage object's size would announce too many bytes for encrypted backups.
- Reading the complete file before responding would delay the download and violate bounded streaming memory or require staging another file.
- Adding a byte-size column would require a migration and historical backfill without improving precision within the existing representable range.

### Preserve streaming and the agent binary's existing size

Keep the physical restore-set writer and the agent binary handler unchanged. The restore set still builds tar entries, padding and its integrity manifest while streaming, with no required total length. The binary response keeps its existing `Stat`-derived length once gzip stops stripping it. Keep the agent HTTP client unchanged so the server policy applies to browsers, scripts and installed agents alike.

Rejected alternatives:

- Summing physical backup sizes would omit tar overhead and generated content.
- Prebuilding the restore set would add latency and disk or memory use.
- Disabling compression only in the agent client would leave browser downloads and other clients affected.

### Observe gzip explicitly in regression tests

Use the existing contract-checking API client with a request editor that explicitly sets `Accept-Encoding: gzip` for binary downloads. Go then leaves the body and `Content-Encoding` intact instead of transparently decompressing them. Check status, absence of encoding, the actual backup or tar contents, and exact lengths where available.

Use agent binary fixtures larger than 2 KiB for both architectures so automatic small-response length inference cannot hide a missing explicit length.

Extend `TestDownloadLogicalBackup`, add `TestDownloadPhysicalRestoreSet` beside the physical restoring API tests, and extend `TestSystemAgentDownload`. Keep preparation through the existing real application and source fixtures. For exact logical length, cover encrypted and unencrypted backups larger than the HTTP server's small-response buffer and not a whole number of megabytes. Explicitly request identity encoding in a separate length check so it fails on the unfixed code for the missing header, independently of the gzip defect. After restoring the length, extend the explicit-gzip logical scenarios to check both absent encoding and exact length on the same response, for encrypted and unencrypted backups. Exercise each logical engine's existing backup/download API coverage for the size convention.

Update the existing damaged-file fixture test to expect `io.ErrUnexpectedEOF` when the stored file has been truncated. The response announces the complete backup's recorded size, so the HTTP client detects the missing bytes while still receiving the truncated contents. The backup's stored status remains unchanged.

Add `TestCompress` in `lib/httpserver` through a small HTTP router, covering JSON gzip with successful decompression, identity or absent acceptance, text static assets, every existing excluded extension, exact exclusions and neighboring paths. Do not put application download paths into the shared middleware tests. Confirm actual application wiring with the download API tests and a neighboring JSON operation: a default Go transport may be used there if the test asserts `HTTPResponse.Uncompressed` as evidence that gzip was received and decoded.

Rejected alternatives:

- Checking only the absence of `Content-Encoding` on a default Go client can pass while the server still compresses.
- Testing only an in-memory response recorder would miss real router wiring and transport behavior.
- Tiny download fixtures can acquire an automatic `Content-Length` from the HTTP server, hiding the missing handler length.

## Risks / Trade-offs

- [Incorrect length causes incomplete downloads or rejected writes] -> Prove exact delivered length for encrypted and unencrypted logical backups, non-whole-megabyte sizes, and the four engines' existing download coverage.
- [The existing floating-point size convention cannot represent every byte count above `2^53`] -> Keep the current size model and document its existing precision limit; extending that model is outside this small fix.
- [Future route changes could miss an exclusion] -> Assert the behavior through the real download operations, with anchored route expressions defined in application wiring.
- [Agent binaries and some restore-set content can benefit from gzip] -> Accept potentially larger transfers in exchange for avoiding server compression during these downloads; retain compression for JSON and eligible static assets. Do not promise a measured throughput gain.
- [A reverse proxy may apply compression independently] -> These guarantees cover the Databasus server; proxy configuration is outside this change.

## Migration Plan

Update the three download descriptions in `backend/api/openapi.yaml` before implementing their response guarantees, and regenerate the API. Document the logical download length in the response description; OpenAPI treats `Content-Length` as a standard transport header, so a duplicate custom generated header field is unnecessary.

Deploy with the normal application release. No database migration, agent update or frontend change is required. Reverting the application change restores the prior compression behavior without any data conversion. Keep the public agent discovery and download paths unchanged.
