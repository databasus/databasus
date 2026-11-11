# Verification record

Verification date: 2026-11-11. Host: Fedora, Linux x86_64 (amd64).
The executed fixture reported PostgreSQL 17.11 and 18.6, Patroni 4.1.5,
etcd 3.6.16 and HAProxy 3.2.26-3e0f11a. No native arm64 HA execution is claimed.
The added arm64 `pg_waldump` clients were started under QEMU with the Debian
bookworm runtime ABI; this does not establish native arm64 scenario success.
Dependency selection and upstream publication evidence are in the
[fixture manifest](../../../../backend/internal/lib/testcontainers/testdata/patroni/README.md).

## Executed checks

Go commands ran with the pinned mise environment after `source scripts/fedora.sh`
and `apply_libedit_shim`. Container tests used
`TESTCONTAINERS_RYUK_DISABLED=true`.

| Check | Outcome |
| --- | --- |
| `mise run doctor` outside the sandbox | Passed; Codex client and trusted review hooks detected |
| `mise run verify-deps` | Passed |
| `go -C backend test -count=1 ./internal/lib/testimages` | Passed |
| `mise run backend:lint` | Passed, including formatting, contract/generated-code and architecture checks |
| `shellcheck scripts/test.sh backend/internal/lib/testcontainers/testdata/patroni/start.sh backend/internal/modules/backups/internal/physical/restorestream/recovery_script.sh` | Passed |
| `python3 -m unittest discover -s scripts -p test_sh_test.py` | Passed; 44 runner checks, including Patroni slow-package selection and 40/15-minute limits |
| `mise run python:lint` | Passed |
| `go -C backend test -count=1 ./internal/lib/schedule` | Passed; UTC boundary checks with stored offsets UTC-3, UTC and UTC+3, and existing explicit-zone coverage |
| `openspec validate verify-postgresql-patroni-wal --strict` | Passed |
| `mise run test` | Passed; backend suites and 194 frontend files / 2866 tests |
| `pg_waldump --version` for both bundled majors and architectures with the runtime ABI | Passed; amd64 native, arm64 under QEMU; PostgreSQL client versions remain 17.10 and 18.4 |

The shared assertion regressions passed on both majors after the source-time
repair: PostgreSQL 17 in 36.330 seconds and PostgreSQL 18 in 36.744 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 40m \
  -run 'TestPhysicalBackupAfterPromotion|TestPhysicalFullHistoryAfterPromotion|TestRestorePhysicalSchemaAndSequences|TestWalSourceReconnect' \
  ./internal/modules/backups/e2e/postgresqlphysical/pg17 \
  ./internal/modules/backups/e2e/postgresqlphysical/pg18
```

The owning engine, recovery API, stream statistics, slot recovery and health
regressions all passed. The physical-backup API suite also passed after adding
the worker-completion diagnostic:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 40m \
  ./internal/modules/backups/physicalapitests/backuping
```

The engine and recovery regression commands were:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 40m \
  ./internal/engines/postgresql/physical \
  ./internal/modules/backups/physicalapitests/restoring \
  ./internal/modules/backups/physicalapitests/streamstatistics \
  ./internal/modules/backups/internal/physical/walstreaming \
  ./internal/modules/backups/internal/physical/streamhealth
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 \
  -run 'Test_Recheck|Test_Classify.*StreamFailure' \
  ./internal/modules/backups/internal/physical/backuping
```

## Reproducer and repair

The initial real Patroni recovery failed before the script repair on both
PostgreSQL majors: copied configuration referenced the source's absolute HBA
and ident paths, which did not exist in the independent target. The script now
uses copied local authentication files when available, clears inherited source
replication and competing recovery targets, removes inherited signals and
disables source archiving before selecting the requested recovery operation.

The independent-recovery tests passed for target time, latest, explicit backup
without extra archived WAL, and external configuration on both majors. They
assert the stale settings and both signal files exist in the extracted input,
then boot the script's output while the former primary remains stopped.
External configuration is prepared only in the target after the script's
existing diagnostic; source HBA hashes and limited-user restrictions are checked.

Additional unfixed reproductions failed on both majors with whitespace-only
assignments (`recovery_target_xid '1'`, `hba_file '/source/pg_hba.conf'`) and
with a nested local include graph ending in `target.settings`. The direct case
retained its competing target; latest recovery refused to start with the stale
HBA path. Recovery preparation now removes both assignment forms and traverses
`include`, `include_if_exists` and `include_dir` relative to each containing
file, including paths with spaces and filenames without `.conf`. All eight
independent-recovery cases then passed in 331.268 seconds, with booted recovery
on both majors.

The absolute-include reproducer then failed on both majors: PostgreSQL could
not open `/var/lib/postgresql/patroni/restore options/entry.settings` in the
independent target. Recovery preparation now reads the source `data_directory`
before removing it and rebases source-local absolute includes to relative
paths inside restored PGDATA. The regression combines absolute include and
include-directory paths, optional missing files, nested relative includes and
filenames without `.conf`. All eight independent-recovery cases passed in
353.747 seconds with booted PostgreSQL.

A further booted regression moved `data_directory` into a local settings file
included from `postgresql.conf`, with an absolute include before that declaration.
The unfixed PostgreSQL 17 case failed in 67.449 seconds because startup still
referenced `/var/lib/postgresql/patroni/restore options/entry.settings`.
Source-directory discovery now follows the local include graph before
sanitization and repeats with the discovered directory to resolve earlier
absolute includes. All eight independent-recovery cases passed on PostgreSQL
17 and 18 in 302.738 seconds, including external configuration, with no failures
or skips. No fixture-labeled containers, networks or volumes remained:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run '^TestPatroniPhysicalBackups/recovers_independently' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

The focused source-time engine and recovery/statistics API regressions also
passed after the formatting changes in 2.900, 0.450 and 0.438 seconds,
respectively. Backend lint, recovery-script ShellCheck and strict OpenSpec
validation passed.

The subdirectory-authentication reproducer failed on both PostgreSQL majors in
95.598 seconds: PostgreSQL tried to load the absent source path
`/var/lib/postgresql/patroni/auth/hba.rules`. The backup contained `auth/hba.rules`
and `auth/ident.rules` without default root-level authentication files. The
source kept its active configuration externally, so staging these copied files
did not alter its access rules. Preparation now maps authentication paths to
native runtime PGDATA or the selected image's PGDATA with `--combine-image`.
All eight independent-recovery cases passed in 317.361 seconds, including
booted authentication checks through direct `pg_ctl -D` and image entrypoints
on both majors. HBA/ident hashes matched the source, and default root-level
authentication files remained absent.

The final diagnostic checks then passed all four explicit-backup and
external-configuration cases in 169.705 seconds. Copied subdirectory
authentication files caused no missing-configuration warning; genuinely external
configuration retained its diagnostic and operator-controlled preparation:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run '^TestPatroniPhysicalBackups/recovers_independently_with_(backup|external-configuration)' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

No recorded containers from the authentication regression runs remained,
including image-entrypoint recovery containers. No fixture-labeled networks or
volumes remained. Final backend lint, recovery-script ShellCheck and strict
OpenSpec validation passed.

A further authentication-precedence regression adds distinct root-level
defaults alongside the referenced subdirectory files. Its root HBA rejects all
connections, and its root ident contains an unrelated mapping. Both defaults
are verified to differ from the referenced files before backup and to exist in
the extracted input and prepared output. Both PostgreSQL majors boot using the
referenced files, verified by the hashes of `current_setting('hba_file')` and
`current_setting('ident_file')`, through direct `pg_ctl -D` and image entrypoints.
The two new cases and the two existing explicit-backup cases without defaults
passed in 173.748 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run 'TestPatroniPhysicalBackups/(uses_referenced_authentication_when_root_defaults_coexist|recovers_independently_with_backup)' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

A mutation that discarded the referenced HBA/ident rewrites whenever root
defaults existed failed both new cases in 82.852 seconds: the prepared main
configuration no longer contained the required referenced HBA path. The original
script was restored byte-for-byte after this sensitivity check.
Both new cases then passed again in 84.225 seconds with the restored script:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run 'TestPatroniPhysicalBackups/uses_referenced_authentication_when_root_defaults_coexist' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

Final backend lint reported zero issues; recovery-script ShellCheck and strict
OpenSpec validation passed. No containers recorded by these authentication
precedence runs or fixture-labeled networks and volumes remained.

A targeted repeat of the interrupted INCREMENTAL case on PostgreSQL 18
observed positive archive bytes at the gateway but timed out waiting for a
matching active server progress row. The held operation subsequently reached
its byte-stall timeout. The test proxy now adds `MAX_RATE 2048` to the pending
held backup so progress observation does not rely solely on socket backpressure.
It records the backend PID from `BackendKeyData`; the
scenario requires that exact PID to report positive streamed bytes and active
state before switching the primary. The original assertions and timeout remain.
All eight interrupted FULL and INCREMENTAL transfer cases passed with the rate
cap in 516.136 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run '^TestPatroniPhysicalBackups/(replaces_an_interrupted|preserves_the_failure)' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

Three independent repetitions of the PostgreSQL 18 INCREMENTAL routing-outage
case then passed with exact backend-PID matching in 499.259 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=3 -timeout 40m \
  -run '^TestPatroniPhysicalBackups/preserves_the_failure_during_a_routing_outage_for_INCREMENTAL_with_postgres:18$' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

The strengthened retry scenario exposed an existing schedule defect: timestamps
read from PostgreSQL could carry the host offset, which cron used instead of UTC.
The recorded UTC due time then passed without dispatch. The pure cron regression
failed for UTC-3 and UTC+3 stored timestamps on the unfixed evaluator. Normalizing
the previous run to UTC before computing the next occurrence passed that
regression and all existing schedule tests, including explicit timezone prefixes.
The four dedicated routing-outage cases then passed in 400.614 seconds on
PostgreSQL 17 and 18, including both FULL and INCREMENTAL failures, exact due-time
dispatch and booted recovery.

## WAL source-time evidence

The delayed-publication API reproducer returned `201` instead of the required
`409`: old-timeline WAL had a source transaction at minute 5, a claim time at
minute 15, and an unreachable promotion-gap target at minute 7. The WAL catalog
and sidecar now store nullable `last_transaction_at` decoded by the matching
`pg_waldump`. Resolver and coverage use this source evidence from the selected
timeline's contiguous published run. Receipt time remains an archiving and
retention timestamp. Existing rows without evidence do not extend confirmed
explicit PITR time; latest replay and completed-backup recovery remain available.

Inclusive time recovery needs a transaction strictly later than the target, so
the confirming upper bound is one microsecond before its timestamp. API tests
cover the exact boundary, late old-timeline publication before and after the
replacement FULL, preserved earlier targets, and missing evidence. Statistics
tests cover delayed publication and a following segment without evidence.

The download regression obtains a valid token, removes its confirming WAL and
retains late-received WAL whose transaction evidence cannot confirm that target.
Downloading returns `409 target_time_unreachable` as a problem response without
archive headers. A controlled substitution of receipt time for transaction time
made this test fail with `200` instead of `409`. The statistics regressions now
normalize fixture times to microseconds and compare both range endpoints and
`latestRestorableAt` exactly. Removing the one-microsecond adjustment made them
fail by exactly one microsecond. After restoring the implementation, the full
recovery and statistics API suites passed in 2.909 and 17.119 seconds.

The engine tests read real WAL from PostgreSQL 17 and 18. They also generate a
20 MiB logical WAL record spanning two segments, reject timestamp evidence from
its incomplete prefix and read the complete transaction from the next segment.
The uploader retains WAL bytes when decoding cannot establish a further bound,
logs the diagnostic and uses only already decoded complete records. The full
engine and recovery/statistics API suites passed in 35.835, 5.417 and 19.090
seconds, respectively.

The two-switchover cases temporarily hold a real old-timeline catalog publication
and publish it with a receipt time after the recorded promotion-gap target.
Both majors refused the gap before and after replacement FULLs, recovered the
old/new eligible targets and repeated those recoveries after cleanup. This
focused run passed in 106.572 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run '^TestPatroniPhysicalBackups/recovers_after_two_switchovers' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

The read-only Codex consultation recommended source-time evidence instead of
receipt-time or observed-promotion-time cutoffs. The repair keeps PostgreSQL's
WAL decoder, catalog-only restore selection and the existing per-timeline
restore policy. No new HTTP operation or response field was introduced.

`pg_waldump` was extracted from the exact matching PGDG server packages at
17.10 and 18.4, without replacing existing client binaries. Package URLs,
archive hashes and executable hashes for amd64/arm64 are recorded in
`tools/dependencies/clients.json`. All four require at most glibc 2.34 and the
existing zstd/lz4 runtime libraries. Their version commands passed inside the
pinned Debian bookworm base, using explicit QEMU execution for arm64 without
registering host-wide emulation. The complete HA matrix remains native amd64.

## Prepared-transaction payload evidence

The real-WAL reproducer failed on both PostgreSQL majors in 2.888 seconds.
A multiline prepared-transaction identifier supplied a fabricated COMMIT time
in 2099, both as a plain description and as a complete-looking record header.
The reader now keeps a trusted prefix before PREPARE, records later candidates
without trusting their timestamps, and verifies the last candidate through a
separate single-record decode at its exact LSN. Only that decode's first line
can extend evidence; failed verification preserves the prefix and reports a
diagnostic. No segment needs more than one additional decoder process.

The prepared-transaction cases cover both fabricated forms and genuine prepared
COMMIT/ABORT completion after the multiline identifier. The genuine completion
must fall between measured server timestamps, so accepting only an older prefix
cannot satisfy those cases. The full engine suite, recovery API suite and
statistics API suite passed in 34.153, 5.720 and 19.908 seconds, respectively:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 40m \
  ./internal/engines/postgresql/physical \
  ./internal/modules/backups/physicalapitests/restoring \
  ./internal/modules/backups/physicalapitests/streamstatistics
```

The initial and two-switchover Patroni cases also passed on both majors in
151.429 seconds, including booted recovery and delayed-publication refusal:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  -run '^TestPatroniPhysicalBackups/(restores_initial|recovers_after_two_switchovers)' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

A targeted read-only Codex consultation found no false-accept path through
multiline GIDs and recommended no further correctness change. The accepted
tradeoff is conservative evidence after an invalid final candidate: newer
unverified records do not extend the trusted prefix. Final backend lint and
strict OpenSpec validation passed. No recorded Patroni regression containers,
fixture networks or fixture volumes remained.

## Fixture lifecycle evidence

The lifecycle suite passed startup and durable settings on both majors, replay
barriers, A-to-B-to-A routing, election after whole-node SIGKILL, transfer release
and cancellation, and isolated intentional failures.

The parent process inspected Docker after each child exited. Success,
cancellation and held-transfer failure each removed four containers and three
volumes; partial startup removed two containers and one volume. Each removed
its network and temporary directory. The parent also checked the expected
nonzero child status and diagnostic, and confirmed an unrelated network remained.

Parent-side fallback is registered before child launch and is restricted to the
unique fixture label and the parent-owned child directory. A further lifecycle
case killed the child's process group after fixture readiness, invoked the
fallback and verified all recorded containers, volumes, network and temporary
files were absent. Normal child-cleanup assertions run before fallback cleanup.
All eleven lifecycle cases passed in 94.327 seconds with Ryuk disabled.

## Full matrix

The complete package command includes 16 backup scenarios for each major and
the lifecycle suite:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 40m \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

The final complete run passed in 1258.763 seconds after the source-directory,
authentication-precedence, transaction-evidence, completed-backup target and
maximum-trusted-time repairs. All 32 backup cases and all eleven lifecycle cases
passed, with no failed or skipped cases. Earlier complete runs passed in
1297.690 seconds after the completed-backup target repair and 1343.393 seconds
before it.

After the final command exited, no fixture-labeled containers, networks or
volumes remained. A read-only cleanup check covered 196 container IDs and 32
temporary roots recorded by the final matrix and WAL ordering reproducer runs;
none remained. Each backup scenario below passed on both PostgreSQL
17.11 and 18.6:

| Scenario | PostgreSQL 17 | PostgreSQL 18 |
| --- | --- | --- |
| Initial FULL plus WAL, schema and sequences | Passed | Passed |
| Two switchovers, old/new targets, history use and retention cleanup | Passed | Passed |
| Abrupt primary loss and independent recovered writes | Passed | Passed |
| Interrupted FULL with a ready new primary | Passed | Passed |
| Interrupted INCREMENTAL with a ready new primary | Passed | Passed |
| FULL failure recheck during routing outage and scheduled retry | Passed | Passed |
| INCREMENTAL failure recheck during routing outage and scheduled retry | Passed | Passed |
| Ignored inactive slot, reconciliation, resumed progress and recovery | Passed | Passed |
| Absent external slot on the promoted node | Passed | Passed |
| Actual removal of an unprotected external slot | Passed | Passed |
| Identified required WAL recycled before archival, target refused after later FULL | Passed | Passed |
| Independent recovery to an explicit target | Passed | Passed |
| Independent recovery to latest available WAL | Passed | Passed |
| Independent recovery of an explicit backup without extra WAL | Passed | Passed |
| External configuration diagnostic and target-only configuration preparation | Passed | Passed |
| Referenced HBA/ident files take precedence over unrelated root defaults | Passed | Passed |

All eight interruption cases observed positive archive data bytes, excluding
archive-name and progress metadata, alongside an active server-side backup
progress entry matched to the held connection's backend PID.
Slot ownership checks used Patroni's public pause/resume operation to reconcile
its cached inventory and observed completed management cycles.

The outage cases observe the failed worker's release of its database reservation
before reopening the route. Patroni's permanent-slot configuration on both nodes
excludes slot-rebuild requests in these dedicated schedule cases. They record a
cron due time through the API, process scheduler passes and check no later
backup or forced request exists before that boundary. The exact subsequent FULL
must have reason `full cadence is due`, dispatch and creation times at or after
the boundary, and a recovery set that boots successfully. Separate absent-slot
and lost-slot scenarios keep their original ownership modes.

The package belongs to `backend/test-slow-packages.txt`; the standard runner's
40-minute slow-package timeout accommodates this matrix. Other packages retain
15 minutes. Runner tests verify both limits and selection of the Patroni package
exactly once in the slow group.

No deployment, Kubernetes/operator, network-partition or multi-datacenter
coverage is claimed. `mise run test-full` was not run.

## Completed-backup target without a later transaction

Four booted regressions reproduced PostgreSQL's `recovery ended before
configured recovery target was reached` on the unfixed resolver: FULL and
INCREMENTAL recovery at the exact advertised backup completion point on both
majors. The source archived checkpoint/rotation WAL after backup completion
without a later COMMIT/ABORT. The unfixed package runs failed in 18.562 seconds
(PostgreSQL 17) and 18.654 seconds (PostgreSQL 18).

The resolver now omits extra WAL for an exact completed-backup point only when
no later transaction confirms time-based recovery. Latest requests preserve
their WAL; a confirming transaction one microsecond after the completion point
also preserves WAL for that explicit target. Later unsupported targets remain
refused. Recovery test helpers retain target microseconds rather than dropping
them while constructing the script command.

All four booted cases then passed in 18.612 seconds (PostgreSQL 17) and 18.665
seconds (PostgreSQL 18), including restored rows, writable recovery and no extra
archive WAL. The full restoring API suite passed in 3.115 seconds and stream
statistics in 17.328 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 10m \
  -run '^TestRestorePhysicalBackupCompletion$' \
  ./internal/modules/backups/e2e/postgresqlphysical/pg17 \
  ./internal/modules/backups/e2e/postgresqlphysical/pg18
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m \
  ./internal/modules/backups/physicalapitests/restoring \
  ./internal/modules/backups/physicalapitests/streamstatistics
```

The final shared-helper run included those four booted cases alongside existing
target-time, promotion, history, schema/sequence and WAL reconnect regressions.
It passed in 126.231 seconds on PostgreSQL 17 and 117.373 seconds on PostgreSQL
18:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -p 1 -parallel 3 -timeout 20m \
  -run 'TestRestorePhysicalBackupCompletion|TestRestorePhysicalBackupToTargetTime|TestPhysicalBackupAfterPromotion|TestPhysicalFullHistoryAfterPromotion|TestRestorePhysicalSchemaAndSequences|TestWalSourceReconnect' \
  ./internal/modules/backups/e2e/postgresqlphysical/pg17 \
  ./internal/modules/backups/e2e/postgresqlphysical/pg18
```

The final owning-package regression run passed in 33.657 seconds for the engine,
6.777 seconds for restoring and 20.272 seconds for stream statistics:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m \
  ./internal/engines/postgresql/physical \
  ./internal/modules/backups/physicalapitests/restoring \
  ./internal/modules/backups/physicalapitests/streamstatistics
```

## Fixture unit separation

After the complete matrix, fixture connection, identity, node lookup, readiness,
replay, switchover, slot configuration and HTTP request functions were moved into
separate files. All 13 declarations retained their bodies apart from formatting;
no runtime behavior changed. The affected controls were then exercised through
all eleven lifecycle cases and both majors' two-switchover and protected-slot
cases. All fifteen cases passed in 162.552 seconds, with no failures or skips:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 15m \
  -run '^TestPatroniFixtureLifecycle$|^TestPatroniPhysicalBackups/(recovers_after_two_switchovers|keeps_the_ignored_slot)' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

Backend lint and strict change validation passed after the move. A read-only
cleanup check found none of this run's 52 container IDs remaining, and no
fixture-labeled containers, networks or volumes remained.


## Maximum trusted WAL transaction time

Four real-decoder regressions failed on the unfixed reader in 2.571 seconds:
an earlier COMMIT held a greater timestamp than a later ordinary COMMIT or a
verified prepared COMMIT, on PostgreSQL 17 and 18. The reader returned the later
record's older timestamp in every case.

These are controlled WAL input fixtures rather than a claimed reproduction of
a live scheduling race. Each server creates real records; only a copied
segment's earlier COMMIT timestamp is raised. The test verifies the original
record checksum, recalculates CRC32C after the change, preserves page headers
and supplies the copied segment to the real matching pg_waldump. The source
cluster and its clock remain unchanged. PostgreSQL obtains the commit timestamp
before inserting its WAL record, so concurrent insertion need not preserve
timestamp order; see the [PostgreSQL transaction implementation](https://github.com/postgres/postgres/blob/REL_18_STABLE/src/backend/access/transam/xact.c#L1352).

The reader now preserves the maximum trusted timestamp during ordinary decoding
and when merging the independently verified candidate after PREPARE. All four
cases passed in 2.595 seconds:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 10m \
  -run '^TestWalTransactionTimeOrder$' ./internal/engines/postgresql/physical
```

The owning engine, restoring API and stream-statistics suites then passed in
33.016, 7.596 and 20.741 seconds, respectively:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m \
  ./internal/engines/postgresql/physical \
  ./internal/modules/backups/physicalapitests/restoring \
  ./internal/modules/backups/physicalapitests/streamstatistics
```

The 44 runner checks passed in 11.013 seconds. Backend and Python lint, strict
change validation and all 92 main specification validations also passed.

A separate read-only Codex consultation found the repair consistent with the
accepted PREPARE policy. Only the independently verified candidate can extend
the trusted prefix; earlier unverified post-PREPARE output remains conservative
evidence rather than being trusted as a record. It found both new regression
branches meaningful and recommended no further correctness change.

The final ordering-fixture pass completed in 2.579 seconds after deriving its
raised timestamp from the source clock observed after both original commits.
This removes any assumption about how quickly those commits execute. The full
HA matrix passed on the finished runtime repair in 1258.763 seconds, as recorded
above.
