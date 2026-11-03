## Context

See `proposal.md` for the motivation. The candidate list came from a name search in the post-refactor review. It was rechecked against the current tree with text search and `deadcode`, and several of its line numbers and claims had drifted:

- `databases.Module.DeleteDatabase` (`backend/internal/modules/databases/module.go:131`) has no caller at all, not even in tests, which delete through HTTP.
- `noRowLock` is unused only in `backend/internal/modules/backups/internal/config/config_repository.go:36`. The `databases` and `storages` copies are in use.
- The 13 dictionary keys listed in `tasks.md` have no full-path use in `frontend/src`. The TypeScript check in the tasks settles each one.
- CI starts all compose services in `.github/workflows/ci-release.yml:305` and `:404`, and `scripts/test.sh:67` does the same locally.

`deadcode` proves functions dead but stays silent on some methods: it did not flag `GetChainSpan`, which has no reference at all. A method's absence from its report therefore proves nothing, so LSP references settle every removal.

The approach is constrained by these rules:

- `backend/AGENTS.md` and the root `AGENTS.md`: no shims, no issue numbers in production code, blank-line and comment rules for code the change touches.
- `docs/architecture.md`: `module.go` holds the whole public contract of a module. The HTTP layer already imports `physical/catalog` (`backend/internal/modules/backups/internal/httpapi/physical_backup_handler.go:8`), so using its constants adds no new dependency.
- `frontend/AGENTS.md`: every other dictionary is typed `typeof en`, so a key is removed from all six at once.
- `website/AGENTS.md`: the font files served must match `website/app/globals.css` and `website/app/components/SiteDocument.tsx`.

## Goals / Non-Goals

**Goals:**

- Every removal is confirmed through LSP references first and leaves no dangling mention in code, docs or skills.
- Behavior for Databasus users stays as it is.

**Non-Goals:**

- Finding more dead code than the agreed list. New candidates that LSP confirmation turns up are reported, not removed.
- Reworking the `module.go` alias lists. They are a larger task, and some aliases are needed by signatures.

## Decisions

### Delete test-only code and let tests read through what exists

`ParseUniqueNames`, `FormatUniqueNames`, `WaitUntilEmpty` and `Store.Clear` are deleted, because nothing needs what they do. The splitting, trimming and empty-input cases in `namelist_test.go` are real behavior of `NormalizeUniqueNames`, so they are rewritten to call it rather than deleted.

Three more have only test callers, and those tests still check the state they read:

- `CountByReferences` has one assertion (`store_test.go:343`). The test creates its own storage, so it lists through `ListByStorage`, which production already uses (`deletion_job.go:109`), and expects one row.
- `mysqlfamily.HasPrivilege` backs two absence checks (`privileges_test.go:148`, `:185`). `EffectivePrivileges` is already a slice, so they become `assert.NotContains`, and its own test at `:241` goes with it.
- `FindByReference` has one caller, `findPending` (`fixture_test.go:153`), which runs the query itself. No existing method finds one pending deletion by its reference.

Rejected: moving all three into test files, which keeps helpers the tests do not need. Rejected: leaving test-only code in production files, which keeps the misleading surface this change exists to remove. Rejected: asserting through HTTP in the pending deletion tests, because no operation exposes that state.

`WalSegmentNo.GetFilename` stays, because tests in two packages call it and a `_test.go` file cannot be shared across packages. Rejected: duplicating it in each test package.

### Use the constants instead of deleting them

`restore.StatusFailed` and `StatusCanceled` are the real values the repository writes as literals (`restore_repository.go:143`, `:148`, `:153`), together with `IN_PROGRESS` at `:153` and `:170`. The queries take them as bound parameters.

`catalog.BackupTypeWal` names a value the list query really returns. `physical_backup_list_row.go:32` compares against the same three literals, so it switches on `backup.Type` with the three constants.

Rejected: deleting the constants, which leaves string literals as the only record of those values. Rejected: `PhysicalBackupListRowType.Valid()` at the row mapping, a sound check that would still leave `BackupTypeWal` unused.

### Delete `BackupErrorCanceledByDbRemoval`

The migration `20261027214332_move_physical_backups_into_module.sql` keeps this value in old rows, and no code writes or compares it. The `error_reason` column is plain text with no `CHECK` constraint, and `ErrorReason` is a string type, so an old row reads as before.

Rejected: keeping the constant for old rows, since nothing reads it by name. The migration stays untouched because applied migrations are immutable.

### Keep `containsIgnoreCase` copied

It is one line, repeated in three packages (`mongodb/client_tool.go:80`, `postgresql/logical/client_tool_failure.go:227`, `mysqlfamily/client_tool_failure.go:116`). It is duplication rather than dead code.

Rejected: a shared helper, because a new dependency between engine packages or a new library package costs more than the copies.

### Keep `clean-testcontainers`

The exit trap in `scripts/test.sh:25` removes only its own run's containers, and backend suites run with Ryuk disabled (`scripts/test.sh:115`, `:120`). After a `SIGKILL` this task is the only ready way to remove leftovers.

Rejected: deleting it as the review proposed. Its danger is real, but its description warns about it, and a developer would otherwise type the same `docker` filter by hand.

### Keep the Postgresus volume check

`docker/start.sh:24` refuses to start on a populated `/postgresus-data`. The December 2026 date in its message describes support for the old image, not the check. Without the check, a user who only renamed the image would likely get a fresh empty instance, which looks like lost backups.

Rejected: removing it at that date. Only the message needs editing once the old image stops being supported, which this change does not do.

### Keep `assets/logo-square.*`

The repository has no references, but external pages may link to them, and that cannot be checked from here. Keeping two small files costs almost nothing, while removing a file something external links to would break it.

### Remove the fixtures nothing uses

`test-physical-postgres-17` and `-18` appear only in `docker-compose.yml`, `scripts/dev.sh:8` and `.env.example`. The seed reads the logical fixture and the WAL backup source (`backend/cmd/seed_dev/main.go:57`, `:62`). Seed tests own their source containers, as `docs/testing.md` says. `test-logical-postgres-16` stays because the seed backs it up.

Rejected: keeping them for manual checks, because the owner confirmed nobody uses them by hand.

The comment above `test-physical-postgres-17` explains the replication rule, and `wal-backup-postgres-18` still needs that explanation, so the comment moves there.

### Remove `backend-victoria-logs`

Nothing in code, scripts, docs or `.env.example` refers to this service. `deploy/helm/README.md:123` mentions VictoriaLogs only as an example target of `OPEN_TELEMETRY_URL`. Neither `scripts/dev.sh:8` nor `:9` starts it, so only the bare `docker compose up -d` runs it, on the floating `latest` tag. The owner confirmed nothing is run by hand.

A developer checking external logging locally loses the ready service and starts any OpenTelemetry receiver themselves, then points `OPEN_TELEMETRY_URL` at it. The logging settings tests need no such service.

Rejected: pinning the tag and placing the service behind a compose profile, which keeps a service nobody starts and adds a version to maintain.

### Start only the test PostgreSQL server in test runs

`scripts/test.sh:67` and both CI jobs change to `docker compose up -d backend-db-test`, the same command `test-frontend-browser-e2e` already uses (`ci-release.yml:227`). Tests take their server from `TEST_DATABASE_DSN` (`backend/internal/lib/testdb/testdb.go:21`), and `apptest` sets the database after loading the configuration (`backend/internal/app/apptest/apptest.go:372`). The review's "`backend-db-test` is enough" also needs the `backend-db` wait removed at `ci-release.yml:310` and `:409`.

Rejected: keeping `backend-db` in CI. Nothing found connects to it during tests, and a wait for a service that is not needed hides that.

### One test pins the command

`scripts/test_sh_test.py` asserts the exact compose call and the CI lines, so a later edit cannot silently widen them. The check at `scripts/test_sh_test.py:523` drops ports `5007` and `5008`, which no longer exist.

Rejected: no test, because lint would not notice the command widening again.

## Risks / Trade-offs

- [A CI job needs a service the narrowed command no longer starts] → The evidence above comes from reading the code; a local machine usually has `backend-db` running from other sessions, so a local pass does not prove this. A failing CI run on the pull request is the proof, and the fix is one word in the compose command.
- [A dictionary key is used through a path the search cannot see] → The TypeScript server and `pnpm --dir frontend typecheck` catch a typed key. Only `storages.edit.sftp.` is built dynamically in the code, and it holds no removed key.
- [A removed icon or font is requested at runtime] → Icon paths in production code are literal, and the sidebar's dynamic path exists only in its test. `pnpm --dir frontend test` covers that test, and fonts are named explicitly in the two website files above.
- [A method is reached by reflection, so references look empty] → The removed contract methods have no callers, and removing one from an interface breaks only compile-time users. Handlers bound by name are generated from the API contract, which does not change.
