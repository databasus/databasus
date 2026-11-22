## Context

See proposal.md for the motivation. The current state that shapes the approach:

- The frontend parser turns a connection string into form fields in several paths: URI, JDBC and libpq key-value. The URI path takes the host straight from the URL (`frontend/src/entities/databases/model/postgresql/PostgresqlConnectionStringParser.ts:102-104`), and none of the paths look at the host's meaning.
- Every use of a physical database starts in the provider's `openConnectedDatabase`, which opens the SSH tunnel before anything else (`backend/internal/modules/databases/internal/providers/postgresqlphysical/connected_database.go:31-37`). Saving then runs `Detect`, which opens an ordinary inspection connection, and only then the connection test (`backend/internal/modules/databases/internal/userdatabase/service.go:455-477`). The connection test opens a replication connection first (`backend/internal/engines/postgresql/physical/replication_readiness.go:34-37`). A refusal containing "no pg_hba.conf entry" becomes `PG_HBA_NO_ENTRY` with advice to edit `pg_hba.conf` (`replication_readiness.go:186-205`). Neon answers a replication connection exactly that way, which a manual probe confirmed.
- The frontend explains reason codes through `frontend/src/entities/databases/model/postgresql/physical/ConnectionErrorCode.ts` and `physicalConnectionErrorContent.ts`. The codes come from `DatabaseConnectionFailure` in `backend/api/openapi.yaml`, and the backend constants live in `backend/internal/engines/postgresql/shared/errors.go:13`.
- The logical backup end-to-end tests live under `backend/internal/modules/backups/e2e/`. The CI `logical` group selects that whole tree except the physical suites (`mise-tasks/backend/test-group-packages:19-20`), so a new package nested under it joins `logical` without a selection change. `scripts/test.sh:124-127` sources `.env` for local runs.
- No test in the repository reaches an external service; every source and target runs in a container.

The approach answers to the root `AGENTS.md`, `backend/AGENTS.md` (engine code without tables in `internal/engines`, errors in `errors.go`, tests through the harness and the API), `frontend/AGENTS.md` (copy only in dictionaries, feature-sliced layout) and `website/AGENTS.md` (page copies per locale). Nothing here changes `docs/architecture.md`.

## Goals / Non-Goals

**Goals:**

- One place in the parser decides the address of a direct or session-mode connection, for every input format.
- The Neon refusal happens before any connection opens, to the source or to an SSH bastion, and without guessing from error text.
- The Neon test is repeatable on a shared project from any number of concurrent runs, and leaves nothing behind.

**Non-Goals:**

- A general registry of managed PostgreSQL providers. Two host rules do not justify one.
- Exercising the pooler in the end-to-end test.

## Decisions

### The parser normalizes the address after any format has parsed

One function takes a successful parse result and returns it with the host and port of a direct or session-mode connection. Every format calls it on its way out, so a Neon pooler host in a JDBC or libpq string is treated like one in a URI. The Neon rule removes the `-pooler` suffix from the first label of a host under `neon.tech` only. The Supabase rule changes port 6543 to 5432 for a host under `pooler.supabase.com` only.

Rejected alternatives:

- Rewriting only in the URI path. Neon shows URIs, but a user can still paste a JDBC or key-value string, and that string would then keep the pooler.
- Matching any host containing `pooler`. It would rewrite addresses we know nothing about, including a self-hosted PgBouncer someone named `pooler.example.com`.
- Rewriting on the backend. Hosts typed by hand would also change silently, and the backend would start guessing what the user meant. The proposal leaves typed hosts alone.

### The physical provider refuses a Neon host before it opens anything

The physical engine gets a check of the configured source host, not the tunnel address: a host equal to `neon.tech` or ending with `.neon.tech` yields a `ConnectionTestError` with the new code `ConnErrPhysicalReplicationUnsupported`, defined in `engines/postgresql/shared/errors.go` next to the others, and a message saying that Neon supports only logical backups. `OpenTunnel` calls it first, next to the existing refusal of Databasus's own internal cluster and before it dials a bastion. The provider's `openConnectedDatabase` opens the tunnel before anything else, and every path into a physical database passes there, so the connection test answers the reason, and a create or update goes through `ToConnectionError` to `400 database_connection_failed` with that message as its detail, before the tunnel, `Detect` or the replication connection run. The code reaches the API as `PHYSICAL_REPLICATION_UNSUPPORTED` through the existing mapping, and the frontend explains it like the other codes.

Rejected alternatives:

- Checking inside `TestReplicationConnection`. The SSH tunnel and, on save, the `Detect` connection would already have opened, and a save would fail with whatever `Detect` hit instead of the new reason.
- Mapping the `pg_hba.conf` refusal to the new code when the host is Neon. The verdict would depend on Neon keeping its error text, and the test would still make a pointless connection.
- Detecting Neon by the `neon_superuser` role. That needs a successful ordinary connection first, and the role name is just as much a Neon implementation detail as the host.
- Refusing in the provider's `Validate`. The error would arrive as an invalid setting rather than as a connection reason the form already knows how to explain.

### The Neon test is its own package in the logical tree, with a container twin

The test lives in `backend/internal/modules/backups/e2e/postgresql/neon`. A separate package lets a developer run just this test with one `go test` command, and keeps the skip logic out of the container-based package next to it. The cycle is one function that takes a connection URL. The Neon test calls it with `TEST_NEON_DATABASE_URL`, and a sibling test calls it with a PostgreSQL 18 container from the harness, so setup, comparison, naming, the sweep and cleanup run on every `test-full` and every CI run even without Neon. Both use the application from the test harness and the generated API client, as the other end-to-end tests do, and talk to the server directly only for setup, verification and cleanup.

Rejected alternatives:

- A Neon-only test. Its own logic would be verified only where the secret exists, and an agent or contributor without a Neon project could not check a change to it.
- A new CI group. It would need its own runner, matrix entries and release gate wiring, while `logical` already gates release on both architectures.
- A test file inside the existing `e2e/postgresql` package. A skip there would read like a skip of the container tests, and the package would gain a dependency on an outside service.

### Isolation through databases and roles named per run

The test creates a source database and restore target databases with the prefix `databasus_e2e_`, followed by the Unix creation time and a random suffix. The limited user gets its name from Databasus, so the test marks it with `COMMENT ON ROLE` holding the same prefix and creation time. Cleanup runs through `t.Cleanup`, so it also runs after a failed test. It drops the run's databases with `DROP DATABASE ... WITH (FORCE)` first, which removes the limited user's grants with them, and then drops the role. Each object that cannot be dropped fails the test with its name. A process killed by a signal or by the `go test` timeout runs no cleanup; the sweep at the start of later runs handles that case. The sweep reads the time from database names and role comments, removes only marked entries older than one hour and never touches an unmarked object.

Rejected alternatives:

- Neon branches per run through the Neon API. Branches isolate better, but they need a second secret, an API client and the API's own limits, all for a check that one project already serves.
- One fixed database reset by each run. Concurrent runs from two architectures and several worktrees would overwrite each other.

### `TEST_IS_NEON_REQUIRED` decides between skip and failure

The test reads `TEST_NEON_DATABASE_URL` and `TEST_IS_NEON_REQUIRED` from the environment, in a test-support helper of the package that returns the URL and `isNeonRequired`. The boolean follows the naming rule for booleans in the root `AGENTS.md`. CI sets the flag only for the `logical` group. Developers can set it too, for example before a release check on their machine.

Rejected alternatives:

- Detecting CI through `GITHUB_ACTIONS`. A reader of the test would have to know why CI behaves differently, and nobody could reproduce the CI behavior locally.
- Failing everywhere without the string. Every contributor without a Neon account would see `mise run test-full` fail.

### CI passes the secret through the step environment by group

The step that runs the backend tests sets `TEST_NEON_DATABASE_URL` to the secret and `TEST_IS_NEON_REQUIRED` to `true` only when `matrix.group` is `logical`, and to empty values otherwise. Job-level `env` would expose the secret to every step of every backend group.

## Risks / Trade-offs

- [Neon outage or slow cold start blocks a release] → Neon wakes a suspended compute on connection within seconds, under the existing 30-second client connect timeout. A real outage blocking a release is the price the user accepted for a mandatory check.
- [Free-tier limits of the Neon project] → Each run creates three small databases and removes them. Storage and compute hours stay far below the free limits even with both architectures and local runs.
- [A cleanup failure leaves objects in the project] → A failed cleanup fails the test, because the spec promises an empty project after each run. The age-based sweep removes what a killed run could not.
- [Neon changes its pooler host format] → The parser then stops rewriting, and an imported string keeps a working but transaction-mode host. Nothing breaks, and the parser test documents the expected format.
- [A legitimate non-Neon host under `neon.tech`] → Neon owns the domain, so a non-Neon PostgreSQL there is not expected.
- [The password leaks through test output] → The test never prints the connection string, and it passes every error it reports through a redaction of the password and the full string. A test with a refused password checks the failure message. Host, user and database name stay visible, because they tell which project failed; GitHub masks only the exact secret value.

## Migration Plan

Nothing to migrate. Implementation is verified by the container twin and does not wait for a Neon project; a run against Neon is an extra check when `TEST_NEON_DATABASE_URL` is in `.env`. Before the change lands on `main`, the repository owner creates a dedicated Neon project and adds its direct connection string as the repository secret `TEST_NEON_DATABASE_URL`. Without the secret the first CI run after the merge fails the `logical` check, which is the intended behavior.
