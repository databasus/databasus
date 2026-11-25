# Design

## Context

See proposal.md for the motivation and scope.

Databasus decides whether it can serve a PostgreSQL server from the major version the server reports. It then runs the client programs from that major's bundle under `assets/tools/<arch>/postgresql/postgresql-<major>/bin/`. The "System" section of `docs/architecture.md` explains why the clients are vendored: the build is reproducible and works offline. Today that decision is spread over several hand-written lists, and each of them refuses 19:

- `backend/internal/engines/postgresql/shared/version.go:11-23` declares the constants `Version12` to `Version18` and the list of bundled versions. The startup check builds one fatal bundle per entry (`shared/client_tools.go:42-60`).
- `shared/client_tools.go:67` requires the physical programs only `if version == Version17 || version == Version18`.
- `physical/source.go:122-136` maps the reported major with a `switch` that knows only `"17"` and `"18"`. Its error text, `physical/errors.go:18`, is "physical backup requires PostgreSQL 17 or 18". The issue reporter saw this message.
- `logical/source.go:110-127` maps the reported major with a `switch` over `"12"` to `"18"`.
- `logical/dump.go:98-107` lists 12 to 15 as the versions without zstd.
- `backend/internal/agent/runner/runner.go:38-40` lists the majors the agent verifies. That list duplicates the keys of `backend/internal/agent/container/postgres_versions.json`, which the agent already loads to choose its restore image (`container/pinned_postgres_versions.go:25-69`).
- `backend/api/openapi.yaml:6133-6138` declares the readOnly enum `PostgresqlVersion`.

Some code already works for 19:

- The frontend recovery commands choose the Docker data layout with `Number(pgVersion) >= 18` (`frontend/src/pages/databases/physical-backups/lib/restoreCommands.ts:10-19`). The `postgres:19` image keeps the PostgreSQL 18 layout, `PGDATA=/var/lib/postgresql/19/docker` with `VOLUME /var/lib/postgresql`.
- The physical backup metadata derives `190000` from the number (`backend/internal/modules/backups/internal/physical/backuping/metadata.go:83-94`).
- The downgrade check compares numbers (`shared/version.go:25-37`).

The PostgreSQL 19 release notes list no change to `pg_basebackup`, `pg_combinebackup`, `pg_receivewal`, the replication protocol, backup manifests, WAL summarization or recovery settings. The additions that touch this area are additive: `pg_stat_progress_basebackup.backup_type`, tar input for `pg_waldump` and `pg_verifybackup`, new slot-synchronization columns, and `wal_compression = on` choosing the best method. The bundled `pg_waldump` 19 decodes that last one.

The verification image (`backend/cmd/verification-agent/images/postgres/Dockerfile`) has one pinned `FROM` stage per major and installs a curated extension list that must exist for every major it builds. Version `case` lists add `pg_uuidv7` from 13, documentdb for 15 to 18 and VectorChord for 14 to 18. The release builds and publishes the image for the majors in two hardcoded matrices (`.github/workflows/ci-release.yml:1417,1482`) and reads each pinned minor from `postgres_versions.json`. Agent end-to-end tests restore into the published image, addressed by the digest the test catalog records (`backend/internal/agent/e2e/verification_images_test.go:42-61`), and `scripts/image_pins.py:14-34` checks the bases and the catalog records for majors 12 to 18.

Upstream availability on 2026-11-25:

- `postgresql-client-19` and `postgresql-19` `19.0-2.pgdg12+1` exist in `bookworm-pgdg` for amd64 and arm64. Their files are dated 2026-11-20.
- `postgres:19` and `postgres:19.0` exist on Docker Hub.
- CloudNativePG has published no PostgreSQL 19 operand image.
- PGDG has no PostgreSQL 19 build of rum, mysql_fdw, tds_fdw or documentdb, VectorChord 1.1.1 has no PostgreSQL 19 package, and TimescaleDB has no `pg19` image. Every other curated extension and the procedural languages exist for 19.

Constraints: `backend/AGENTS.md` keeps engine code free of Gin and GORM and limits the agent's imports to `lib`, `engines` and the generated agent client. The `client-tool-bundles` skill requires a bundle and its version identity to land together, both architectures to be verified in the runtime image, and no version to be added to a lookup table. `docs/how-to-upgrade-deps.md` sets a fourteen-day waiting period for database clients and container images. This change takes an exception to it that the maintainer approved.

## Goals / Non-Goals

**Goals:**

- In the backend's version decisions, a new PostgreSQL major needs only a bundle, an identity constant, an enum value and test matrix entries, and no version check needs editing. Verification of a new major additionally needs a recipe stage, a pinned minor and an entry in the two release matrices of `.github/workflows/ci-release.yml`, which stay lists.
- Every behavior the specs claim for 19 runs against a real `postgres:19` server in a test.

**Non-Goals:**

- The test catalog record of the published PostgreSQL 19 verification image and the generalization of `scripts/image_pins.py`. Both need the digest that the release of this change publishes, so a follow-up commit adds them right after that release.
- TimescaleDB on PostgreSQL 19, which has no upstream build.
- Any change to how a client is chosen for a restore, or to the recovery script.

## Decisions

### Bundle 19.0 from `bookworm-pgdg` now

The bundle takes `postgresql-client-19` and `postgresql-19` (for `pg_waldump`) at `19.0-2.pgdg12+1` for both architectures, the same source and build family as every other PostgreSQL bundle. The test catalog pins the current `postgres:19` digest.

Both releases are taken before the fourteen-day waiting period ends. This exception falls outside the critical-fix ground in `docs/how-to-upgrade-deps.md`, which covers only a release that fixes a security issue or a defect breaking backups, restores or verification, and only that one release. It rests on the maintainer's explicit decision and applies to both releases in the table below. Without the bundle Databasus cannot back up PostgreSQL 19 at all, so users who have already upgraded stay without backups until it ships.

| Release | Published | Normally eligible |
|---|---|---|
| `postgresql-client-19` and `postgresql-19` `19.0-2.pgdg12+1`, amd64 and arm64 | 2026-11-20, file dates on `apt.postgresql.org` | 2026-12-05 |
| `postgres:19` | 2026-11-24, tag push date on Docker Hub | 2026-12-09 |

Both were downloaded on 2026-11-25, which shortens the waiting period by 10 days for the packages and by 14 days for the image. The package hashes matched the `bookworm-pgdg` index, and the image digests are those Docker Hub served for `postgres:19` that day.

The final report and the commit message name the reason, these releases, their publication dates and how many days the period was shortened on the day of download, with the maintainer's decision as the basis, following the reporting that `docs/how-to-upgrade-deps.md` requires for an exception.

Alternatives:

- Wait until 2026-12-05. Rejected, because PostgreSQL 19 users would stay without backups for ten more days, while the release notes change none of the backup tools and the packages come from the official signed PGDG repository.
- Wait for 19.1. Rejected, because 19.1 is expected in February 2027 and the release notes give no reason to distrust 19.0 for backups. Moving to 19.1 is an ordinary patch refresh.
- Take `19.0-1.pgdg12+1`. Rejected, because PGDG superseded it with `-2`, and every bundle follows the newest build of its patch.

### CI and the development VM take `libpq5` from the PostgreSQL repository

The bundled `psql` links against the system `libpq5`. The runtime image installs it from `bookworm-pgdg` (`Dockerfile:106-120`), where it already is 19.0. CI and the development VM install Ubuntu's own `libpq5`: 18 on CI's Ubuntu 26.04 runners and 16 on the VM's Ubuntu 24.04. On the VM, `psql` 17 and 18 already fail with `undefined symbol: PQchangePassword` and `PQsendPipelineSync`, so `TestShippedBundles` fails there, while `mise run doctor` passed, because it only checked that libraries are found. `psql` 19 would fail the same way on CI.

The database client step in CI and the client library step of `docs/how-to-setup-vm.md` install Ubuntu's `postgresql-common` and run its `apt.postgresql.org.sh -y`, which adds the PostgreSQL repository for the running Ubuntu release with the signing key the package ships, then install `libpq5` from it. The `doctor` bundle check runs `ldd -r`, which also resolves symbols, and names an unresolved symbol next to a missing library.

Alternatives:

- Ship `libpq.so.5` inside the PostgreSQL bundles, as `mysql-26` ships its OpenSSL. Rejected, because the runtime image already provides a current `libpq5`, and a second copy would have to be refreshed with every major.
- Add the repository by hand with a downloaded key, as the runtime image does. Rejected, because `postgresql-common` carries the key in a package Ubuntu already signs, and its script picks the codename of the running release.

### The bundle list is the only list; other decisions read the number

`shared/version.go` keeps the list of bundled versions, because it names the directories that must exist and the startup check reads them. Every other decision derives from the major number:

- The physical source accepts a reported major when it is at least 17 and in the bundle list.
- The logical source accepts a reported major when it is in the bundle list.
- A bundle requires the physical programs when its major is at least 17.
- `pg_dump` compresses with zstd when the major is at least 16.

A `GetNumber` method on `pgshared.Version` feeds these comparisons. A reported major outside the list keeps producing a "not supported" error, so PostgreSQL 20 stays refused until it has a bundle. Constant names such as `Version19` stay, because tests and the internal database refer to concrete versions.

Alternatives:

- Add `"19"` to each `switch` and comparison. Rejected, because it leaves five places to edit for PostgreSQL 20. The `client-tool-bundles` skill calls such a table the bug.
- Accept any major of 17 or more and pick the newest bundle for an unknown one. Rejected, because this change's `database-version-support` delta requirement "A PostgreSQL server on a major without a shipped client is refused" forbids backing up a server with a client of an older major.

### Two distinct physical refusals

The physical source first checks that the major has a bundle, then that it is at least 17. A major without a bundle, below 12 or above 19, gets the unsupported-version error. A bundled major from 12 to 16 gets the below-17 error. The first says physical backups need PostgreSQL 17 or later and suggests a logical backup, as the `physical-backups/backup-method` spec has required all along. Today a single message, "requires PostgreSQL 17 or 18", covers both cases, and it misleads a PostgreSQL 19 user into thinking their server is too new by design. The second names the version found and the physical range: from 17 to the newest bundled major, so it never offers PostgreSQL 12 to 16 for physical backups. The logical refusal names the version found and the whole bundle list from 12.

Alternative: one generic "unsupported version" error. Rejected, because the refusal for a server below 17 has a remedy, a logical backup, that the user should see.

### The agent reads the majors it verifies from its pinned versions

Today only the container manager loads the pinned versions (`backend/internal/agent/container/manager.go:68`), and the runner checks a major against its own `getSupportedMajors` (`runner.go:38-40,349-363`) without importing `container`. `runner.Deps` gains a `VerifiedPostgresMajors` field, and `runner.go` drops its list. The wiring in `backend/internal/agent/start/start.go:88-101` fills the field from a new `(*Manager).GetPinnedPostgresMajors`, backed by `PinnedPostgresVersions.GetMajors`. For an uncovered major, the job fails before any container starts and reports, as today, without a restore result. The backend's `classifyAgentReport` (`backend/internal/modules/verification/internal/verifications/retry.go`) then counts it as an agent-side setup failure and requeues it up to `MaxAgentSideAttempts`, so a newer agent may pick it up. The message the `backup-verification` delta specifies becomes the report's short failure summary (`FailMessage`) without the current `resolve pg major:` prefix (`job_execution.go:152-164`). Then `postgres_versions.json` is the single place where an agent release states what it verifies, and this change adds `"19": "19.0"` there.

Alternative: a method on the `Spawner` interface. Rejected, because the spawner starts containers, and the question of which majors are verified is decided before any container exists.

Alternative: keep the list and only reword the message. Rejected, because two lists in one agent can disagree, and then the agent would claim a major it has no image for.

### The verification image builds PostgreSQL 19 with the extensions that exist for it

The recipe gains a `postgres-19` stage pinned to the catalog's `postgres:19` digest. rum, mysql_fdw and tds_fdw leave the common list and join the version conditions with documentdb and VectorChord, all limited to 18 and below. `pg_uuidv7` is installed from 13 upward. The conditions compare numbers instead of listing majors, so a new major only needs its stage. When upstream publishes one of the five extensions for 19, its upper bound moves. A dump that creates a missing extension is already handled: the verification counts the missing extension as an ignorable error.

Both release matrices gain `'19'`, and the agent pins 19.0. The release then publishes `databasus/verification-postgres:19.0` together with the agent that restores into it, in the same release run.

Alternatives:

- Wait until every curated extension exists for 19. Rejected, because nobody knows when that happens, and PostgreSQL 19 users would have no verification until then.
- Build the missing extensions from source. Rejected, because the recipe deliberately installs only packages from PGDG and checksummed vendor releases.

### The PostgreSQL 19 agent test builds the image locally until the release publishes it

The agent end-to-end tests restore into the published verification image by its catalog digest, and `databasus/verification-postgres:19.0` does not exist before this change is released. The PostgreSQL 19 case of `TestVerifyBackup` therefore builds the image from the recipe for the host architecture with `docker buildx build --load`, with `PG_MAJOR=19` and `PG_SERVER_VERSION=19.0`, and tags it into the agent's own repository, as `provideVerificationImage` does with catalog images. The recipe's own check fails the build if its server is not 19.0. The follow-up commit after the release records the published digest in the catalog, switches this case to it, removes the local build, and makes `scripts/image_pins.py` read its majors from `postgres_versions.json`. Until then that script keeps checking 12 to 18. Making it read the pinned versions now would fail `mise run verify-deps`, because the catalog cannot hold a record for an image that is not published yet. The build uses `docker buildx` rather than the Engine API, because the API's classic builder would pull every pinned base stage of the recipe instead of only the one it builds. The Podman subset of the agent tests keeps PostgreSQL 18 until the follow-up, because it restores into a catalog image too.

In the meantime, raising a pinned version deviates from the procedure in `docs/how-to-upgrade-deps.md`, which changes the list, the recipe's base digest and the test catalog together. The follow-up commit closes that gap.

Alternatives:

- Publish the image by hand before this change. Rejected, because nobody on the development machine holds the registry credentials, and the maintainer prefers the release to publish it.
- Ship verification without a PostgreSQL 19 agent test until the follow-up. Rejected, because the first release would then publish an image and an agent that no test had run together.

### Tests follow the existing per-major layout

PostgreSQL 19 validates `recovery_target_xid` and refuses `1`, which the Patroni independent recovery fixture planted as a stale inherited target (`backend/internal/app/apptest/physicaltest/patroni_independent.go`, `patroni_inherited_configuration.go`). The fixture now plants `3`, the first normal transaction ID, which every supported major accepts and which stays stale for the recovery. The CloudNativePG fixture keeps `1`, because it runs only on 17 and 18.


- The matrices that run every supported major gain 19: the logical end-to-end engine matrix, the logical source tests, the physical source tests, the WAL summarization check, and the `physicalapitests` loops over `{"17", "18"}`.
- A new package `backend/internal/modules/backups/e2e/postgresqlphysical/pg19` mirrors `pg18`, including its PgBouncer test, and runs on `postgres:19`. It is registered in `backend/test-slow-packages.txt`, as `docs/testing.md` requires for packages that start containers, and the schema and sequence run command in `docs/testing.md` gains it next to `pg17` and `pg18`.
- `TestTestDatabase` gains a PostgreSQL 19 logical case that saves the database through the API and reads back version `19`.
- The Patroni suite gains `Postgres19` variants of each scenario group. CloudNativePG keeps 17 and 18.
- Single-server tests that use `postgres:18` as a representative server keep it.

Alternative: replace 18 with 19 in the matrices and keep only the oldest and newest versions. Rejected, because CI runs every supported major today, and dropping 18 would leave a supported version untested.

### The API enum gains `"19"`

`PostgresqlVersion` is a readOnly response enum marked `x-engine-values: true`. Adding a value breaks no request, and the frontend types are regenerated in the same change, which the HTTP API compatibility policy in `docs/architecture.md` allows for the user API. The agent API carries the version as a free string (`backend/api/agent.yaml:223-228`), so an older agent receives `"19"` without a decoding error and fails the job with its existing message.

## Risks / Trade-offs

- [The pinned Patroni 4.1.5 may not manage a PostgreSQL 19 cluster, and no Patroni release note confirms 19 support.] → The Patroni 19 variants run in the task that adds them. If Patroni itself cannot bootstrap or switch over a 19 cluster, the 19 variants are removed, and every requirement that only the Patroni suite proves returns to "17 and 18". In `postgresql-physical-backup-failover` these are the five Patroni requirements. In `physical-backups/operations` they are "Physical recovery is independent of inherited replication settings", whose every scenario runs only in the Patroni suite, and "Cleanup preserves recovery points around repeated Patroni switches". The final report says so. A defect in Databasus that only the 19 variant exposes is fixed, not removed.
- [The arm64 verification image for 19 is first built by the release, while the local agent test builds only the host architecture.] → The development machine cannot run arm64 builds, so task 5.1 checks that every package the 19 image installs exists for arm64 in `trixie-pgdg`, the repository of the `postgres:19` base. The release's arm64 build remains the first real build.
- [Agents older than this release cannot verify PostgreSQL 19 backups.] → Agents update themselves to the release they talk to. An agent that has not updated yet fails such a verification with its existing message.
- [Some client behavior on 19 differs in a way the tests miss, for example `pg_dump` now emitting extended statistics.] → The logical end-to-end matrix restores into PostgreSQL 19 and compares the restored data, which covers dump output that a 19 server refuses.
- [The new bundle binaries need a library or symbol the runtime image lacks.] → The bundle task runs each binary's `--version` in `debian:bookworm-slim` for both architectures, as the `client-tool-bundles` skill requires. The backend's startup check runs every program of the host architecture's bundles in the tests. `mise run doctor` adds only a shared-library check of the host architecture's bundles.
- [The enum, the bundle list and the identity constants drift apart for the next major.] → API tests validate every response against the contract, so a 19 database returned with a stale enum fails them.

## Migration Plan

No data migration. Existing databases keep their recorded versions. A rollback to the previous release refuses PostgreSQL 19 databases again: their backups fail with the old message, and their stored backups stay readable.
