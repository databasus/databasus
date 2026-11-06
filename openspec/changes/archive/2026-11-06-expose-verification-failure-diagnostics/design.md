## Context

See [proposal.md](proposal.md) for motivation and scope. This design is necessary because the change crosses the agent report, persistent verification data and the sidebar, with explicit output-size and disclosure constraints.

The current data path loses the diagnostic before the server receives it:

- `backend/internal/agent/restore/restorer.go:69` retains an 8 KiB stderr tail in the restore result while evaluating ignorable errors against the complete stderr.
- `backend/internal/agent/runner/runner.go:367` logs that tail, but `backend/internal/agent/runner/runner.go:545` sends only the short message, exit code and duration in a failed report.
- `backend/internal/modules/verification/internal/verifications/report_service.go:83` passes failed reports to the retry service. `backend/internal/modules/verification/internal/verifications/retry_service.go:31` persists the short message and sends it in terminal notifications.
- `backend/internal/modules/verification/internal/verifications/verification_repository.go:284` currently updates failed outcomes without the ownership predicate used by successful reports. The new outcome-and-diagnostic write must be guarded against cancellation and reassignment races.
- `backend/internal/modules/verification/internal/verifications/service.go:149` already authorizes detail reads through the database's workspace.
- `frontend/src/pages/databases/ui/verification/VerificationDetailDrawer.tsx:151` renders the failure message above the existing sections.
- `backend/internal/lib/logging/redaction.go:108` exposes `logging.Redact` for errors persisted outside log sinks. It masks credential patterns, but does not remove PostgreSQL row excerpts or arbitrary SQL bodies.

The implementation follows `backend/AGENTS.md` for contract-first handlers, service transactions, repository-owned queries and agent dependency boundaries; `frontend/AGENTS.md` for AntD controls, clipboard access and generated API types; and root `AGENTS.md` for scope and verification. `docs/architecture.md` sections "Modules", "Engines and providers", "HTTP API", "Verification agent" and "Logs" determine ownership. `docs/frontend-design.md` and `docs/frontend-localization.md` determine presentation and literal diagnostic rendering.

## Goals / Non-Goals

**Goals:** deliver the underlying error through the existing final report, persist a bounded diagnostic with the final failed attempt, and make it available after agent cleanup without expanding list responses or notifications.

**Non-goals:** use the proposal's out-of-scope list. In particular, the output is a diagnostic excerpt for one failed verification, not a transcript of every command or a complete agent log.

## Decisions

### 1. Add an optional diagnostic object to the report and detail contracts

Add `failureDiagnostic` to `VerificationReport` in `backend/api/agent.yaml` and only to `VerificationDetails` in `backend/api/openapi.yaml`.

Add "Failure diagnostic" to the Product table in `docs/glossary.md`, with API name `failureDiagnostic`: the bounded, sanitized diagnostic of a verification's final failed attempt. Distinguish it from the short failure summary (`failMessage`) used in lists and notifications. Keep the definition short and leave the detailed lifecycle and disclosure rules in the specification and architecture document.

The optional, non-null diagnostic object has two required, non-null properties:

| Field | Shape | Meaning |
| --- | --- | --- |
| `output` | nonempty string, `maxLength: 65536` | Sanitized diagnostic output; additionally limited to 65,536 UTF-8 bytes |
| `isTruncated` | boolean | Some sanitized output was omitted to meet the byte limit |

Document the byte constraint alongside the character constraint. Validate the byte length in the verification service and return `invalid_request` on `failureDiagnostic.output` before any mutation. Reject an object on a `COMPLETED` report. Omission means no diagnostic; do not use explicit null or an empty object. Keep `failMessage` as the short summary and mask it under the same disclosure policy.

Regenerate both servers, the agent and test clients and frontend types with `mise run generate-api`. There is no new endpoint or agent pull request. Optional fields fit reports that have no captured output and historical records; they do not require a compatibility shim or a new agent protocol version. Existing discovery/download paths and required fields are unchanged.

**Alternatives:** appending stderr to `failMessage` is smaller but would also enlarge list cards and external notifications. A separate upload or log-fetch endpoint adds an unnecessary lifecycle and depends on an agent or container that can already be gone.

### 2. Build the excerpt from complete error output before transport

The restore path prepares `failureDiagnostic` from the complete stderr it already receives, rather than from `StderrTail`. It keeps the current full-output evaluation of ignorable errors and unavailable extensions. Diagnostic sanitization and truncation never feed back into verdict classification or disk-exhaustion handling.

For failures in download, temporary-environment setup, restore hooks or the subsequent check, use the underlying error already available at that failure boundary. When restore execution itself fails, retain any captured stderr supplied in its result and the execution error; when no output was captured, do not invent it. Preserve the existing distinction between tool rejection with an exit code and infrastructure failures without one. Do not collect additional container logs or enable verbose tool output.

Failure boundaries explicitly supply the underlying error or captured output. Report sanitization only processes a supplied diagnostic; it never derives one from `failMessage`. A tool rejection without stderr keeps its generic summary and has no diagnostic. If sanitization removes all supplied output, subsequent sanitization preserves that absence.

After sanitization, keep output of at most 65,536 UTF-8 bytes unchanged. For longer output, reserve space for an ASCII middle-omission marker, divide the remaining budget evenly between the beginning and end, and move both cut points to UTF-8 boundaries. Set `isTruncated` when content was omitted. If sanitization removes all output, omit the object. The JSON envelope may exceed 64 KiB because of escaping; the limit is on decoded diagnostic output, not the report body.

**Alternatives:** the existing 8 KiB tail can omit the first causal error. Unlimited stderr would enlarge reports and database rows without a bound. First-and-last excerpts retain both ends of the sanitized output with a fixed storage cost.

### 3. Share the diagnostic policy without introducing a logging subsystem

Keep PostgreSQL-aware diagnostic construction in a dedicated `failure_diagnostic.go` unit under `backend/internal/engines/postgresql/logical/pgrestore`, with pure behavior exercised by focused tests. Both the agent and the verification module may use it without introducing a dependency from the agent into a module or from a library into an engine. Reuse `logging.Redact` for the existing credential patterns rather than duplicating its regexes or changing ordinary restores.

Apply the following policy before the agent sends a report, and apply the same pattern/section policy before the server stores it:

1. Normalize invalid UTF-8, line endings and terminal control sequences into inert readable text. Preserve line breaks and tabs.
2. Suppress output from the first recognized PostgreSQL `DETAIL:`, `CONTEXT:` or `Command was:` section through the end of the stream, replacing it with a fixed marker. This unstructured text has no trusted section-end boundary: SQL strings and row values can contain lines beginning with `ERROR:`, `FATAL:`, `pg_restore:` or the omission marker itself. Those lines cannot safely resume output. Keep primary error and object-identification records before the first sensitive section; later records are omitted with the ambiguous body.
3. Replace exact nonempty credentials known to the agent, then call the existing redaction function. The server reapplies pattern redaction and section filtering because agent input is untrusted. It cannot independently recognize an arbitrary credential known only to the agent.
4. Apply the byte bound after sanitization. If the server's redaction expands accepted output, reapply the same bound and combine truncation flags with logical OR.

Use this policy for detailed diagnostics and the short agent failure summary; do not add diagnostic bodies to notifications, audit events or request logs. Represent sanitization markers as technical diagnostic text, not translated prose. The same policy must be idempotent so processing an already sanitized report does not remove its remaining primary errors.

**Alternatives:** relying on logger masking does not protect an HTTP report or a database column. A universal SQL/data scrubber cannot guarantee recognition of arbitrary values inside custom primary error messages. This design removes known credential forms and recognized sensitive sections, and retains the existing workspace access boundary for remaining diagnostic content.

### 4. Store diagnostics on the existing verification row

Add `failure_diagnostic_output TEXT NULL` and `is_failure_diagnostic_truncated BOOLEAN NOT NULL DEFAULT FALSE` to `verification_runs`. Add structural constraints so null output requires a false truncation flag, and non-null output is nonempty and has an `octet_length` of at most 65,536. The migration is additive and existing rows remain without diagnostics. No new table, stored file or separate cleanup worker is needed.

Extend the verification entity and report mapping. Handle an accepted report and its terminal outcome in the existing transaction. Guard report-driven writes by verification ID, current owning agent and running status, using a row lock and/or conditional updates that make the ownership check atomic with persistence. Ensure a cancellation or reassignment that wins the race prevents both the stale outcome and stale diagnostic from being written. Keep the existing `410 verification_gone` response for stale reports.

For an infrastructure error that is requeued, discard the attempt's diagnostic and clear both new columns in the requeue update. Completion, cancellation and server-generated failure paths clear diagnostics. A terminal failed agent report saves the new columns together with the failure summary; preserve its existing exit-code classification and retry limit. No earlier attempt's diagnostic survives into a later outcome. Deleting the verification deletes the diagnostic automatically because it is on the same row.

Map the object only in the detail response. Use an explicit projection excluding the new columns for verification list queries, as well as omitting them from the list schema and mapper. Reuse the existing detail authorization; no extra privilege or admin-only restriction is introduced.

**Alternatives:** a separate diagnostic table and per-attempt records require extra joins and retention semantics that the agreed scope does not need. Writing diagnostics before or after the outcome transaction could associate one attempt's output with another outcome.

### 5. Use a collapsed section in the existing drawer

Add an AntD `Collapse` immediately below the failure alert, visible only for `FAILED` verifications. Its translated label is "Error details". The initial state is collapsed whenever the selected verification changes. Fetch diagnostics through the drawer's existing detail request and retain its request-revision protection.

Render the diagnostic as React text in a monospace region, retaining line breaks, with technical horizontal scrolling confined to that region and the drawer retaining vertical scroll ownership. Show the translated truncation notice when `isTruncated` is true. When the object is absent, show a translated explanation that no detailed diagnostic was recorded and omit the copy action.

Copy through `BrowserClipboard.copyToClipboard`, await it, and show translated success or failure without a blocking dialog. Scope pending feedback to the selected drawer session so a closed or replaced verification cannot receive stale feedback. The current fallback in `frontend/src/shared/lib/BrowserClipboard.ts:6` does not check the return value of `execCommand('copy')`; make it reject a false return value and always remove its temporary element, with focused regression tests. Do not redesign other clipboard callers.

Update every interface dictionary, preserving live language switching without altering diagnostic text or expanded state. Keep existing cancellation and success presentation unchanged. Screen tests cover behavior; the existing production browser UI suite covers keyboard interaction, both themes, long output and narrow/enlarged layouts.

**Alternatives:** a new tab separates the failure from its context and adds navigation to a drawer that currently reads vertically. An always-expanded raw log would overwhelm the existing status and results sections.

### 6. Verify delivery through the actual report boundary

Add a regression that fails on the current behavior: a failed verification's actual tool error must appear in its later detail response, not only in an agent log. Cover the agent against the existing end-to-end backend harness with a deliberately invalid restore fixture, and use verification module API tests for report validation, ownership, atomic persistence, retry/reset behavior, list omission and workspace authorization. Pure tests cover sanitization, truncation, UTF-8 and idempotence, including secret or section boundaries near truncation points. No fake replaces our backend, database or verification service.

Use screen tests for disclosure, copy success/failure, absence, language changes and stale selection behavior. Extend `frontend/browser/scenarios/verificationScreens.spec.ts` for geometry and keyboard behavior using its existing finite HTTP fixtures. Finish implementation with the module formatters/linters, frontend build, `mise run test`, the production browser UI command and `mise run agent:e2e`.

**Alternatives:** only testing the new field's mapping could pass while the agent still drops stderr. Reproducing the user's full Supabase environment is unnecessary for verifying error delivery and would conflate this change with fixing that restore.

## Risks / Trade-offs

- A bounded excerpt can omit a useful middle error. Preserve the beginning and end and explicitly identify truncation; do not label the excerpt a complete log.
- PostgreSQL custom error messages can contain application data outside recognized sections. Filter recognized row/SQL bodies and credentials, retain workspace authorization, and avoid promising universal anonymization.
- Redaction must occur before truncation to avoid credential fragments. Reapply it on ingestion, test idempotence and keep verdict evaluation on original tool output.
- An agent can disappear before reporting. Preserve the server's failure summary and explain missing diagnostics; do not imply that this feature guarantees a log for every failure.
- Only the final attempt is retained. Clear diagnostics on every requeue and terminal path without a diagnostic, and test reassignment/cancellation races at the HTTP boundary.
- Diagnostics add up to 64 KiB per failed run. Detail-only loading and existing verification deletion bound the feature's lifecycle without a separate retention policy.

## Migration Plan

1. Add the nullable output column, false-default truncation column and structural constraints. Regenerate contracts and update server, agent and frontend together in the normal release.
2. Do not backfill diagnostics. Existing reports without the optional object remain valid; existing records and absent agent reports use the explicit unavailable state.
3. Add the "Failure diagnostic" (`failureDiagnostic`) definition to `docs/glossary.md`, distinguishing it from the short failure summary (`failMessage`). Update `docs/architecture.md` under "Verification agent" with the bounded report-to-row-to-detail flow and a link to the existing log-redaction description. Do not change agent instructions or ordinary restore documentation.
4. Before rolling back the additive migration, stop code that reads or writes the new columns and deploy code that does not name them. The down migration removes the constraints and columns and discards recorded diagnostics; backup files and verification outcomes remain intact.
