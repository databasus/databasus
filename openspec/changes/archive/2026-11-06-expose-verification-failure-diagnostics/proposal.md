## Why

A failed backup verification can show only `pg_restore failed ... (code 1)` even though the verification agent received the database tool's actual error. Users must find the agent's host logs to understand the failure; the verification sidebar should provide that diagnostic output directly.

## What Changes

- Send a separate, bounded failure diagnostic with the verification agent's existing final report, keeping the short failure message for lists and notifications.
- Preserve the first and last parts of long diagnostic output, up to 64 KiB of UTF-8 text, and identify when output was truncated.
- Sanitize diagnostics before transmission and again before persistence, masking credentials and suppressing database row excerpts and SQL command bodies in recognized PostgreSQL diagnostic sections.
- Store diagnostics with the verification and expose them only through its existing detail operation, under the same workspace authorization.
- Add a collapsed "Error details" section below the failure summary in the verification sidebar, with literal monospace output, truncation feedback and a copy action. Explain when a failed verification has no recorded diagnostic.
- Keep diagnostics tied to the final attempt: requeueing clears the previous attempt's diagnostic, and terminal failures without an agent report have no invented tool output.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: report, retain and display useful failure diagnostics without changing verification verdicts, retry policy or access boundaries.

## Impact

- Backend and verification agent: `backend/internal/agent/restore`, `backend/internal/agent/runner`, the verification module, generated API bindings and an additive migration for verification diagnostics.
- Contracts: additive optional diagnostics in `backend/api/agent.yaml` final reports and `backend/api/openapi.yaml` verification details. Existing paths and required request fields remain unchanged; no compatibility adapters are introduced.
- Frontend: the existing verification drawer, its screen/browser tests and all interface dictionaries. The shared clipboard fallback also needs to reject a refused copy so the new action cannot report false success. No additional UI framework or logging service is needed.
- Documentation: define "Failure diagnostic" (`failureDiagnostic`) in `docs/glossary.md` as the bounded, sanitized diagnostic of a verification's final failed attempt, distinct from its short failure summary (`failMessage`); extend the verification-agent data flow in `docs/architecture.md` and preserve the existing tool-output and frontend presentation conventions.
- Applicable rules: root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Out of Scope

- Backup creation, ordinary restores and their existing error dialogs.
- Fixing the Supabase/PostGIS failure itself, changing bundled extensions or adding provider-specific restore behavior.
- Live log streaming, collecting the agent's entire host log, fetching logs from an agent when the sidebar opens, or downloadable log files.
- Diagnostics for successful verifications, per-attempt history, additional container/image metadata and changes to verification scheduling or success criteria.
- Recovering diagnostics for historical runs or for an agent that disappears before delivering a report.
