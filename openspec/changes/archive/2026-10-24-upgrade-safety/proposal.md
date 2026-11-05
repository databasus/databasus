## Why

Milestone 4 of the backend rebuild starts renaming tables and adding constraints, and every later milestone migrates the schema of instances that hold users' only record of their backups. Nothing checks today that an instance of the last release keeps its data after an upgrade, and nothing stops an older image from starting on a schema it does not understand. Both protections have to exist before the first such migration lands.

## What Changes

- The server and every console command that opens the internal database refuse to start when the internal database has been migrated by a newer Databasus version than the running one. The message states the newest migration version in the database and the newest one the binary carries, and says that going back to an older version after an upgrade is not supported. **BREAKING** for operators who roll back an image after an upgrade: such a start used to proceed and fail later on missing tables, and now stops at once. The release notes say so.
- An upgrade test starts the image of the newest release, writes data through its API, runs a logical backup, then starts the image built from the current code on the same volume. Through the API it checks that every item is still there and that the backup restores into a fresh database with the same rows.
- A shared helper waits for a condition in tests instead of a fixed sleep. The upgrade test is its first user.
- The upgrade test runs as its own CI check, and publishing and release wait for it. `just test` runs it after the backend packages.
- `docs/architecture.md` records that an upgrade is one-way.

## Capabilities

### New Capabilities

- `upgrade-test`: what the upgrade test writes before an upgrade and what it checks after it.

### Modified Capabilities

- `internal-database-migrations`: the application refuses to start on a schema newer than its own.
- `ci-pipeline`: the upgrade test runs as its own check outside the backend groups, and publishing waits for it.
- `local-development-workflow`: `just test` includes the upgrade test.

## Impact

- Code: `backend/internal/lib/database` (the version check), `backend/internal/app` (the start-up error), a new `backend/internal/lib/testwait`, a new test package `backend/e2e/upgrade` with its comparison helper `backend/e2e/upgrade/upgradecheck`.
- Tooling: `scripts/test.sh`, `backend/Justfile` (lint covers `./e2e/...`), `backend/.golangci.yml` (the target layout includes `e2e/`), a new `test-upgrade` job in `.github/workflows/ci-release.yml` that reuses the storage candidate image.
- Docs: `docs/architecture.md` ("Backend layout" gains `e2e/upgrade/`, "Migrations"), `docs/testing.md` ("Runs").
- Answers to the root `AGENTS.md` and `backend/AGENTS.md`.
- Depends on change `isolated-test-databases`, which gives test containers their run label.

### Out of scope

- The Docker filesystem suite's `upgrades` group. It keeps checking that the data volume, its permissions and the internal PostgreSQL cluster survive upgrades from fixed older releases.
- Physical backups and WAL in the upgrade data. They are added in milestone 11, whose migration merges the full and incremental tables.
- A way to roll an instance back to an older version.
- `lib/clock`, moved to milestone 4.
