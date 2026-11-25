# Proposal

## Why

PostgreSQL 19 became generally available on 2026-11-19, and Databasus refuses every PostgreSQL 19 server. The connection test of a physical backup fails with "physical backup requires PostgreSQL 17 or 18", and a logical backup fails with "unsupported PostgreSQL version: 19" (issue #923). Users who upgrade their servers lose their backups until Databasus accepts the new release.

The PostgreSQL 19 release notes change nothing in `pg_basebackup`, `pg_combinebackup`, `pg_receivewal`, the replication protocol or the recovery settings Databasus relies on. Supporting 19 is therefore mainly a question of shipping its client programs and proving the existing behavior on a real 19 server.

## What Changes

- Ship the PostgreSQL 19 client programs (`pg_dump`, `pg_restore`, `psql`, `pg_basebackup`, `pg_receivewal`, `pg_combinebackup`, `pg_waldump`) for amd64 and arm64. The bundle holds 19.0 from the PostgreSQL Debian repository and is taken now, before the fourteen-day waiting period ends, as an exception the maintainer approved.
- Accept PostgreSQL 19 servers for logical backups, physical backups and restores. The API's PostgreSQL version enum gains `"19"`.
- Derive every PostgreSQL version decision in the backend from the version number and the list of shipped client programs, instead of from per-version lists. Physical backups need major 17 or later. The physical client programs are required in bundles from 17. zstd dump compression is available from 16. A server on a major line with no shipped client, such as a future PostgreSQL 20, stays refused, and the refusal names the supported versions.
- A physical source older than 17 is refused with a message that suggests a logical backup. A source newer than the newest shipped line is refused with a message that names the supported range.
- Verify PostgreSQL 19 backups. The verification image gains a PostgreSQL 19 build on `postgres:19`, without the five curated extensions that have no PostgreSQL 19 build yet: rum, mysql_fdw, tds_fdw, documentdb and VectorChord. The agent pins PostgreSQL 19.0, and the release publishes `databasus/verification-postgres:19.0` together with the agent that uses it.
- The verification agent takes the majors it verifies from its pinned versions. For a major outside them, such as a future PostgreSQL 20, it explains that its release cannot verify that major and that a newer agent release may add it, instead of reporting a bare "unsupported postgres major".
- Run every test matrix that boots each supported major on 19 as well. This includes a physical end-to-end package for 19 and the Patroni failover matrix.
- Install `libpq5` from the PostgreSQL repository in CI and in the development VM, as the runtime image already does, and make `mise run doctor` report a library that lacks a symbol a bundled program needs. Ubuntu's `libpq5` is older than the `psql` of recent majors needs: 16 on the VM's Ubuntu 24.04 already breaks `psql` 17 and 18 there, and 18 on CI's Ubuntu 26.04 would break `psql` 19.
- Update the supported-version statements in `README.md`, its six translations, the website pages and `assets/tools/README.md`. The README states logical backups from 12 and physical backups from 17, replacing its incorrect "14 … 18 (physical and logical)".

## Out of scope

- **The test catalog record of the published PostgreSQL 19 verification image.** The release that ships this change publishes `databasus/verification-postgres:19.0`, so its digest exists only after that release. A follow-up commit right after the release adds that digest to the test catalog, moves the PostgreSQL 19 agent test from a locally built image to the published one, and makes `scripts/image_pins.py` read its majors from the pinned versions instead of checking 12 to 18.
- **CloudNativePG failover guarantees on 19.** CloudNativePG has published no PostgreSQL 19 operand image yet. Its failover requirements stay on PostgreSQL 17 and 18 until it does.
- New PostgreSQL 19 features, such as `REPACK`, logical replication of sequences or reading WAL from tar archives.
- Restoring into PostgreSQL 19 a dump that an older `pg_dump` took from a server with `standard_conforming_strings = off`. PostgreSQL 19 refuses such dumps, and this change neither detects nor converts them.
- Refreshing the bundled minors of PostgreSQL 12 to 18.
- The browser end-to-end scenario and the upgrade test, which keep their PostgreSQL 18 and 17 sources.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `database-version-support`: adds the PostgreSQL server versions Databasus accepts for logical backups, and the refusal of a logical source on a major no shipped client serves. Physical acceptance and refusals stay in `physical-backups/backup-method`.
- `physical-backups/backup-method`: the PostgreSQL 17 requirement gains an upper bound at the newest shipped client and a PostgreSQL 19 scenario.
- `physical-backups/operations`: the recovery guarantees stated for PostgreSQL 17 and 18 extend to 19.
- `postgresql-physical-backup-failover`: the promotion and Patroni guarantees stated for PostgreSQL 17 and 18 extend to 19. The CloudNativePG requirements stay unchanged.
- `development-environment-check`: the client tool bundle check also reports a symbol that an installed library lacks.
- `backup-verification`: PostgreSQL 19 backups are verified, and an agent asked to verify a backup of a PostgreSQL major its release does not cover reports that clearly.

## Impact

- Rules this change answers to: the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` for the regenerated API types, and `website/AGENTS.md` for the website copy and its translations.
- Code: `backend/internal/engines/postgresql/shared`, `logical` and `physical`; the agent's `backend/internal/agent/runner`, a getter and the pinned versions in `backend/internal/agent/container` and the wiring in `backend/internal/agent/start`; the verification image recipe `backend/cmd/verification-agent/images/postgres/Dockerfile` and the two verification image matrices of `.github/workflows/ci-release.yml`; `backend/api/openapi.yaml` and the code generated from it; a PostgreSQL 19 case in the frontend recovery command tests.
- Client bundles: new `assets/tools/{x64,arm}/postgresql/postgresql-19/bin/` with seven programs per architecture, about 2.4 MB each, and their records in `tools/dependencies/clients.json`.
- Tests: the stale recovery target planted by the Patroni independent recovery fixture, which PostgreSQL 19 refuses as `1`; the matrices that boot each supported major, a new `backend/internal/modules/backups/e2e/postgresqlphysical/pg19` package registered in `backend/test-slow-packages.txt` and listed in the schema and sequence run command of `docs/testing.md`, the Patroni matrix, a PostgreSQL 19 agent verification test on a locally built image, and the test-image catalog entry `postgres:19`.
- Development environment: the database client step in `.github/workflows/ci-release.yml` and its test `scripts/backend_test_preparation_test.py`, the client library step of `docs/how-to-setup-vm.md`, and the bundle check in `scripts/development_environment_check.py`.
- Release: the next release publishes `databasus/verification-postgres:19.0`, `:19` and `:19-v<version>`.
- API: the readOnly enum `PostgresqlVersion` gains a value. Requests do not change. The frontend's generated types are regenerated in the same change.
