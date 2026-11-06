## 1. Establish the existing behavior

- [x] 1.1 Before production edits, run the internal backuping tests and physical backup API tests using the first targeted command below. Confirm that the existing shutdown scenarios cover completed FULL and INCREMENTAL results, interrupted execution, failed metadata publication, and user cancellation. Record the baseline outcome; this refactor does not introduce a new defect reproduction or new scenarios.

## 2. Share orchestration

- [x] 2.1 Replace `runFullBackup` and `runIncrementalBackup` with one `runBackup` method and a named run-options value. Build common settings and the progress callback once; preserve full-only raw-size measurement, incremental parent preparation, retry-budget classification, shutdown classification, detached result persistence, logging attributes, and notification ordering. Keep the existing executors temporarily so this slice is independently runnable. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/internal/physical/backuping ./internal/modules/backups/physicalapitests/backuping` from the prepared backend shell below.

## 3. Share execution

- [x] 3.1 Introduce one common execution lifecycle and a small internal strategy for FULL and INCREMENTAL preparation, stream error classification, timeline failure rechecks, and successful-stream validation. Preserve preflight order, manifest cleanup, credential cleanup, replication-slot ownership, per-attempt object keys and publication validation, codec fallback, timestamp placement, and receipt accumulation. Remove the redundant executor types, constructors, internal specs, and service/backuper fields after updating their callers. Keep kind-specific helpers in files named for the concept they implement. Verify with the same targeted Go command as task 2.1, including the existing timeline and stream tests.
- [x] 3.2 Replace the duplicated full and incremental metadata constructors with one common construction path and small kind-specific field handling. Preserve serialized fields, full and incremental labels, root references, and omission of a parent-incremental reference that equals the root; retain `uploadMetadata` and its detached timeout. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/physicalapitests/backuping ./internal/modules/backups/physicalapitests/restoring` from the prepared backend shell below.

## 4. Verify the complete backup and recovery pipeline

- [x] 4.1 Run every existing physical API package and both PostgreSQL 17 and 18 physical end-to-end packages using the final targeted commands below, without filtering or changing scenarios. Require actual cluster recovery for full-only backups, full-plus-incremental chains, and target-time recovery, together with promotion, slot, tunnel, metadata, publication, cancellation, and shutdown coverage supplied by those packages. Preserve `decideImmediateReplacementFull` and its scheduler caller. Record command outcomes and fix any exposed defects according to the testing skill.
- [x] 4.2 Run `mise run backend:format`, `mise run backend:lint`, and the routine suite in Change verification. Search the repository for removed internal identifiers and update affected live references without rewriting archived changes. Confirm that no public contract, schema, stored format, main spec, or agent instruction changed. Validate the artifacts with `openspec validate unify-physical-backup-execution --strict`, record verification evidence, and finish for mandatory implementation review with the working tree ready to commit.

## Targeted verification commands

Use the pinned mise toolchain. The Fedora shim and disabled Ryuk setting follow `docs/testing.md`. Each command runs as an ordinary foreground command; these are implementation checks, not commands already run during planning.

Prepare the backend shell:

```sh
cd backend
source ../scripts/fedora.sh
apply_libedit_shim
```

Baseline and orchestration/executor checks:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/internal/physical/backuping ./internal/modules/backups/physicalapitests/backuping
```

Metadata checks:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/physicalapitests/backuping ./internal/modules/backups/physicalapitests/restoring
```

Final physical API and real cluster recovery checks:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/physicalapitests/...
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups/e2e/postgresqlphysical/pg17 ./internal/modules/backups/e2e/postgresqlphysical/pg18
```

## Verification evidence

- Before production edits, the targeted internal backuping package passed in 0.338 seconds and the physical backuping API package passed in 18.386 seconds.
- After orchestration was shared, the same packages passed in 0.339 and 20.835 seconds. An earlier run was interrupted after changing the start-log message prevented the existing shutdown observer from synchronizing; the original FULL and INCREMENTAL start messages were restored, and only that interrupted run's four remaining containers were removed by their unique testcontainers session label.
- After execution was shared, the same packages passed in 0.341 and 16.381 seconds.
- After metadata construction was shared, the internal package passed in 0.339 seconds, backuping API tests in 32.434 seconds, and restoring API tests in 3.385 seconds. No test scenario or assertion changed; the incremental stream-error test file was renamed to match the shared classifier unit.
- All five physical API packages passed: backuping (54.968 seconds), catalog (32.295 seconds), restoring (17.292 seconds), streamstatistics (36.845 seconds), and walstreaming (218.159 seconds, including its real-time idle-source observation).
- Both unfiltered physical end-to-end packages passed with real cluster recovery: PostgreSQL 17 in 111.868 seconds and PostgreSQL 18 in 47.081 seconds.
- `mise run backend:format` and the final `mise run backend:lint` passed; the latter reported zero Go issues and passed API generation, contract, architecture, and module checks.
- `gopls check` reported no diagnostics for the shared orchestration, execution, strategy, kind-specific preparation, and metadata files.
- The final production Go line count in backuping decreased from 3,753 to 3,504. The renamed stream-error test is byte-for-byte identical to its original file. No public API, schema, main spec, stored format, or agent instruction changed.
- `mise run test` passed with exit status 0: all selected backend packages and all 2,616 frontend tests across 189 files passed. This was the routine command, not `test-full`; real physical cluster recovery was verified separately by the two end-to-end packages above.
- The final targeted internal backuping and backuping API checks passed in 0.339 and 18.398 seconds after the result constructors were placed beside their sole caller. Backend formatting and lint passed again with zero issues.
- `openspec validate unify-physical-backup-execution --strict` and `git diff --check` passed.

## Change verification

```sh
mise run test
```
