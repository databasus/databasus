## Context

See proposal.md for the motivation. The state this design starts from:

- `database.Migrate` builds a goose provider with out-of-order migrations allowed and a session lock, then calls `provider.Up` (`backend/internal/lib/database/migrate.go:21-64`). It never looks at what the database already records. `app.New` logs a migration error and returns it, and `cmd/databasus` exits non-zero (`backend/internal/app/app.go:34-43`).
- Two migrations were shipped and later removed from the repository: `20251210053223_add_schemas_column.sql` (first released in v2.1.0, renamed by `fda3bf9b`) and `20260526120000_add_telegram_http_proxy.sql` (first released in v3.47.0, deleted by `1f4cf296`). Instances that ran those releases record both versions in `goose_db_version`.
- Release tags live on `main`. From `develop`, `git describe --tags` reaches `v3.60.0`, while the newest tag is `v3.63.1`.
- The Docker filesystem suite already has an `upgrades` group (`e2e/docker-storage/run.sh:786-839`). It starts fixed old releases, writes raw markers into the database and the volume, and checks after the upgrade that the files, the secret key, the log and the cluster identity survive. It writes nothing through the API and restores nothing.
- CI already builds the candidate image once per commit and uploads it as an artifact (`.github/workflows/ci-release.yml:468-528`). `test-filesystems` downloads it (`:530`). `determine-version`, which gates image publishing and the release, waits for the lint, the backend groups and the filesystem suite.
- The dev seeder `backend/cmd/seed_dev` talks to the current API with a hand-written client (`backend/cmd/seed_dev/api_client.go`).

The change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `docs/architecture.md` ("Migrations") and `docs/testing.md` ("Layers": upgrade tests in `backend/e2e/upgrade/`; "Waiting").

## Goals / Non-Goals

**Goals:**

- An image older than the schema refuses to start, without blocking instances that applied migrations which were later removed.
- A test, run in CI on every pull request, that upgrades the newest release with real data and restores its backup.

**Non-Goals:**

- Physical backups in the upgrade data (milestone 11).
- Testing upgrades from several releases. The filesystem suite covers older releases for the volume, and each release is tested against its predecessor in turn.

## Decisions

### The check compares the newest versions

Before `provider.Up`, `database.Migrate` reads the highest applied version through the provider and compares it with the highest embedded version. When the database is ahead, it returns `ErrSchemaNewerThanBinary` with both version numbers (`goose_db_version` stores numbers, not file names), applies nothing, and `app.New` turns it into the message the spec requires. The check uses the same provider and session lock, so it sees the state left by a concurrent migrating process.

Rejected alternative: refusing whenever the database records a version the binary does not carry. The two versions removed after release would then stop every instance that ran v2.1.0 or v3.47.0 from upgrading. Keeping a list of retired versions instead would make every future deletion a hand-maintained exception.

The cost of the chosen check is one case it misses: a newer release whose only new migration has an older timestamp than the running version's newest one. Out-of-order merges happen inside one release, so this only matters when a release ships nothing but such a migration.

### `docs/architecture.md` states that an upgrade is one-way

A bullet in "Migrations" says that migrations only go forward, and that a binary refuses to start on a schema newer than its own, because an older binary on a renamed table fails at the first request instead of at start-up. The proposal marks the refusal as breaking, so the release notes mention it.

### The upgrade test is a Go test that drives two images

`backend/e2e/upgrade` holds only test files, and `upgradecheck` is imported only by them, so nothing in it reaches the production binary. `docs/architecture.md` "Backend layout" gains `e2e/upgrade/`: the upgrade test, which may import only `lib`, `api` and its own `upgradecheck`, and which nothing imports. The test:

1. creates a Docker network, a volume and a PostgreSQL 16 source container with a few rows. The container code stays in `e2e/upgrade` until a second consumer needs it, because `lib/testcontainers` does not exist yet and moving the old `containers` helpers there belongs to the engine milestones;
2. starts the previous release on the volume, attached to the network;
3. writes the data through that release's API and waits for one encrypted logical backup to complete;
4. stops the old container and starts the current image on the same volume;
5. checks every item through the current API, tests the connection of the stored database, which decrypts the password stored before the upgrade, then restores the backup into a new database on the source server and compares the rows.

The test reads the images from `DATABASUS_PREVIOUS_IMAGE` and `DATABASUS_IMAGE`. Without them it fails at once and names `just test-upgrade`. Containers, the network and the volume get unique names, and the label `databasus.test-run` from change `isolated-test-databases` when a run ID is set. The flow removes them itself before it returns, on success and on error, so a caller sees no leftover resources once the flow has returned; a `t.Cleanup` would run only after the calling test function ends. `scripts/test.sh` extends its exit trap to labelled volumes, so a Ctrl+C during the upgrade test removes the volume too.

The comparison of the items written before the upgrade with the items read after it lives in `e2e/upgrade/upgradecheck`, a package with its own unit tests that need no image. `just test-package ./e2e/upgrade/upgradecheck/...` runs them, while the image-driven test stays in `e2e/upgrade`. The flow itself is a function that returns an error, and on failure the error carries the end of the log of the instance that failed. The test functions turn the error into a test failure. This lets a test drive the flow into a failure on purpose and check the error and the cleanup without failing itself. The flow returns the unique names it generated, so that test can check the cleanup by name whether or not a run label is set.

Rejected alternatives:

- Running the current code in the test process against the old instance's data. That would skip `docker/start.sh`, the image's users and the volume permissions, which is where upgrades have broken before.
- Extending the bash runner of the filesystem suite. Writing API data, polling a backup and comparing restored rows is far easier to read and maintain in Go, with the HTTP client and the wait helper.
- Turning the dev seeder into the upgrade writer. The seeder follows the current API. The upgrade writer follows the API of the previous release, which differs from the current one for one release after every breaking change.

### Writing against the old API and reading against the new one live in separate files

`previous_release_client_test.go` holds the requests the old release understands. `current_release_checks_test.go` holds the reads of the current code. Until milestone 4 both speak today's API. When a milestone breaks an operation, it changes only the reads. The next release then brings the writes up to date, because its predecessor speaks the new API. This keeps each upgrade test valid for exactly one pair of releases.

### The previous release is the newest release tag by version

`backend/Justfile` gets `test-upgrade`. It picks `git tag --list 'v*' --sort=-v:refname | head -1`, sets `DATABASUS_PREVIOUS_IMAGE=databasus/databasus:<tag>`, builds `databasus-candidate:local` from the checkout unless `DATABASUS_IMAGE` is set, and runs `go test -race -count=1 ./e2e/upgrade/...`. The tag is chosen by version, not by reachability, because releases are tagged on `main` and `develop` does not contain the newest tags. CI checks out with `fetch-depth: 0` so the tags exist.

### `testwait` arrives with its first consumer

`backend/internal/lib/testwait` gives `Until(t, description, condition)`. It polls with a growing interval up to the test's deadline. On timeout it fails with the description and the last message the condition returned. `Poll` waits the same way but returns the timeout as an error instead of failing the test. The upgrade flow waits through `Poll` for the health endpoint, for the backup to complete and for the restore to finish, as `docs/testing.md` "Waiting" requires, so it can add the end of the log of the instance it waited for before returning the error. The linter already forbids `time.Sleep` in the target layout, and the target layout grows to include `e2e/`.

### CI and the full run

- A `test-upgrade` job needs `build-storage-image`, downloads the candidate, checks out with tags and runs `just --justfile backend/Justfile test-upgrade` with `DATABASUS_IMAGE` set. `determine-version` adds it to `needs`.
- `scripts/test.sh` runs `just --justfile backend/Justfile test-upgrade` after the backend packages pass, under the same run ID.
- `backend/Justfile` `lint` and `format` cover `./e2e/...`, and `backend/.golangci.yml` lists `e2e/` in the target layout.

## Risks / Trade-offs

- [Building the image makes `just test` several minutes longer locally] → Docker layer caching keeps rebuilds incremental. A developer who has an image passes it as `DATABASUS_IMAGE`.
- [Docker Hub is unreachable, or the newest tag is not published yet] → The pull fails, and the test names the image it could not pull. CI runs after a release is published, because the release job pushes the image before tagging.
- [A breaking API change leaves the writes and the reads speaking different APIs for one release] → That is intended. The separate files keep each side small, and the test fails loudly when a side is forgotten.
- [An out-of-order-only release goes undetected by the version check] → Accepted, as the decision explains.

## Migration Plan

Nothing to migrate. The version check changes nothing for an instance whose database is at or behind its binary. Rollback is a revert.
