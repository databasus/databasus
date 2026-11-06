## Why

Physical full and incremental backups repeat the same orchestration and executor lifecycle, so changes to cancellation, publication, timing, or cleanup must be maintained twice. Sharing that lifecycle reduces the chance that the two backup kinds diverge while preserving the backup and restore behavior already covered by the existing suites.

## What Changes

- Replace the separate full and incremental orchestration methods with one backup run that builds common settings, persists the result, and sends notifications.
- Replace the two executor lifecycles with one execution path. Keep the preparation, validation, error classification, and metadata differences in small functions for each backup kind.
- Share metadata construction and upload while preserving the existing full and incremental metadata formats and chain references.
- Preserve immediate replacement full backups after failover or a timeline switch; `decideImmediateReplacementFull` has a live scheduler caller.
- Preserve the existing completed-result shutdown fix, incremental retry budget, publication fences, write receipts, and cleanup ordering.
- Verify the refactor with existing physical API tests and actual PostgreSQL 17 and 18 cluster recovery tests. No new behavior or test scenarios are planned.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is an internal refactor with no requirement changes; `.openspec.yaml` declares `skip_specs: true`.

## Impact

- Backend implementation under `backend/internal/modules/backups/internal/physical/backuping/`, including service construction, run settings, executors, and metadata construction.
- Governed by root `AGENTS.md` and `backend/AGENTS.md`, the existing architecture boundaries, and the testing skill.
- Existing physical API and PostgreSQL physical backup end-to-end suites provide regression coverage.
- No changes to HTTP operations, module contracts, database schema, dependencies, stored backup formats, or recovery commands.

### Out of scope

- Changing scheduling decisions, backup cadence, retry policy, or immediate replacement rules.
- Changing cancellation, shutdown deadlines, file publication eligibility, retention, or notification policy.
- Reworking WAL streaming, restore execution, logical backups, or frontend code.
- Introducing a general executor framework, plugin registry, compatibility wrappers, or a fixed line-reduction target.
- Adding product requirements or changing agent instructions.
