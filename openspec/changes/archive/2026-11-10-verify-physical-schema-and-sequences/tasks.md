Run the Go commands from the repository root using the pinned tools. On Fedora, first prepare the current shell as `docs/testing.md` describes:

```sh
source scripts/fedora.sh
apply_libedit_shim
```

## 1. PostgreSQL 17 recovery from FULL and WAL

- [x] 1.1 Add connection-based schema and sequence fixture units under `backend/internal/app/apptest/physicaltest/` and the FULL-plus-WAL case of `TestRestorePhysicalSchemaAndSequences` in `pg17`. Cover baseline and post-FULL objects, added and renamed columns, a table-rewriting type conversion with converted rows, defaults, identity mode, indexes, constraints, rename/drop operations and rolled-back DDL. Recover through the downloaded bundle and served script after contiguous WAL reaches the final commit; assert one completed FULL and no incrementals. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences$' ./internal/modules/backups/e2e/postgresqlphysical/pg17`.
- [x] 1.2 Complete sequence coverage in that runner: serial, identity and explicit defaults, pre-FULL and post-FULL sequences, `CACHE 1` and a larger cache, `nextval`, forward `setval` with both `is_called` values, committed parameter changes and rolled-back calls and sequence DDL. Assert semantic definitions, ownership dependencies, controlled forward floors, restored-ID safety and repeated allocations across connections without requiring an exact first recovered value. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences$' ./internal/modules/backups/e2e/postgresqlphysical/pg17`.

## 2. Incremental recovery and PITR

- [x] 2.1 Add the PostgreSQL 17 FULL-plus-two-incrementals-plus-WAL case using the same SQL fixture and assertions. Put distinct changes between backups and after the final incremental, verify completed backup counts, and prevent later backups from absorbing the WAL-only stage. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences$' ./internal/modules/backups/e2e/postgresqlphysical/pg17`.
- [x] 2.2 Add `TestRestorePhysicalSchemaAndSequencesToTargetTime` in `pg17` with independent before-commit and after-commit cases. Use a second source connection to choose the early target after DDL execution but before its transaction commits; put both targets after the final incremental, preserve source-clock margins and archive through a later schema transaction. Check the appropriate definitions, rows, rename/drop state, transactional sequence parameters and usable generated IDs at each target, without treating sequence calls as transactional row writes. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequencesToTargetTime$' ./internal/modules/backups/e2e/postgresqlphysical/pg17`.

## 3. PostgreSQL 18 and observation sensitivity

- [x] 3.1 Add both top-level tests to `pg18`, calling the same runners with the PostgreSQL 18 image and exercising all four recovery cases without copied SQL or assertions. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences' ./internal/modules/backups/e2e/postgresqlphysical/pg18`.
- [x] 3.2 Add bounded sensitivity checks after the positive recovery assertions: remove a representative added column and alter a sequence cache on disposable recovered fixture objects, then require the same catalog observations to distinguish each mutation from its saved definition. Keep cleanup unconditional. Verify the final shared fixture and all eight recovery cases with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences' ./internal/modules/backups/e2e/postgresqlphysical/pg17 ./internal/modules/backups/e2e/postgresqlphysical/pg18`. If a case exposes a production defect, record its failing run before a limited repair, rerun the regression and the affected package tests, and reconcile the change artifacts.

## 4. Documentation and backend checks

- [x] 4.1 Document the fixture's coverage, connection-based reuse, sequence caveats and focused run command in `docs/testing.md`. Cross-check the implementation against the delta scenarios, format only this change's backend files and run `mise run backend:lint`; verify the artifacts with `openspec validate verify-physical-schema-and-sequences --strict`.

## Change verification

```sh
mise run test
```
