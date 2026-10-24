## Context

`docs/architecture.md:256-271`, the section "Automatic checks", lists the rules a machine enforces and the tool for each. The current state, see proposal.md for why it matters now:

- `backend/.golangci.yml:5` sets `tests: false` and enables only the standard set plus nine linters. The file has CRLF line endings, while `.gitattributes` asks for LF.
- `agent/verification/.golangci.yml` repeats the backend file line for line except `gofumpt.module-path` (`databasus-verification-agent` instead of `databasus-backend`). The agent is a separate Go module.
- With test files included, today's linters alone report about 530 issues in the existing code, mostly `errcheck` (275) and `noctx` (192).
- The existing Go code lives in `internal/features`, `internal/util`, `internal/config`, `internal/middleware`, `internal/storage` and `cmd/` in the backend, and in `internal/` and `cmd/` of `agent/verification`. Nothing exists yet in the target layout from `docs/architecture.md` ("Backend layout"): `cmd/databasus`, `cmd/verification-agent`, `internal/app`, `internal/modules`, `internal/engines`, `internal/lib`, `internal/agent`, `tools/archlint`.
- The production server is `backend/cmd` (package `main`). Its dependency graph contains `testing`, testify and testcontainers through helper files in 12 packages under `internal/features`, which in turn pull in the test-only packages under `internal/util/testing`. There are 22 `testing.go` and 5 `mocks.go` files in all, for example `internal/features/restores/core` and `internal/features/backups/backups/usecases/physical/postgresql`. Most of these helpers are used by tests of other packages, and many reach unexported state: `DrainForTest` on the storage deletion worker (`internal/features/storages/files/testing.go:29`), the counters set by `SetStorageDatabaseCountersForTest` (`internal/features/storages/testing.go:26`). One production method exists only for tests outside these files: `DatabaseService.DeleteForTest` (`internal/features/databases/service.go:301`).
- `internal/features/backups/no_direct_provider_writes_test.go` parses backup code and fails when it calls a provider's save or delete method directly.
- CI runs backend tests with `go test -p=8 -count=1 -failfast -timeout 40m` (`.github/workflows/ci-release.yml:443`), without `-race`; the agent's tests already use `-race` (`ci-release.yml:248`). `just test-package` and `scripts/test.sh:59` run without `-race`. Neither lint job runs `govulncheck` or `go mod verify`.

The change is constrained by `backend/AGENTS.md` ("Dependencies", "Long-running work", "Errors"), the testing skill and the root `AGENTS.md` rule "Security": a check is never weakened to make a build pass, and a suppression carries its reason.

## Goals / Non-Goals

**Goals:**

- Every rule in "Automatic checks" that can apply today is enforced on the target layout, so the first module that moves meets them at once.
- Old code keeps the golangci-lint checks it has today, so no unrelated work is blocked; only the production-binary, provider-write and build-tag rules of `archlint` reach it.
- The production server binary links no test library from our own code, and from third-party code only through the two packages rule 6 accepts by name.
- Races fail backend tests.

**Non-Goals:**

- `importas`. No package has a fixed alias yet; the generated API packages that need one arrive with the contract milestone, and the linter is enabled there.
- `gochecknoglobals`, enabled once the old global singletons are gone.
- Fixing the issues the linters would report in old code.

## Decisions

### Strict rules apply to an explicit list of target paths

The configuration names the target layout once, as a regular expression over paths: `^(api|cmd/databasus|cmd/verification-agent|internal/(app|modules|engines|lib|agent)|tools)/`, the Go part of "Backend layout" in `docs/architecture.md`. The contract milestone adds `./api/...` to the lint recipes when the package appears. Every new linter is excluded for paths outside it through `linters.exclusions.rules` with `path-except`. For test files outside it, every linter is excluded, so they stay unchecked as today. Inside the target layout, all linters run on production and test files.

Alternatives:

- List the old paths and exclude them. The old code is spread over at least seven roots, and a root left out of the list, such as `internal/middleware`, would start failing on the new rules. Code added to an old root by mistake would also pass silently in the target layout's place.
- `new-from-rev`. Moving a file makes all of it new, so the lint would fail on old violations exactly at the moment of the move, as the refactor plan already notes.
- Fix the old code. About 530 issues in test files alone, in code that every later milestone rewrites, with no change in behavior.

### The rule set

- `forbidigo` forbids `time.Sleep`, `Eventually`, `fmt.Print*`, `os.Exit`, `os.Getenv` and `context.Background()`, each with the allowed places from "Automatic checks", and `*gorm.DB` and `database.Conn` outside repository files.
- `sloglint` requires attributes only, `...Context` methods, constant messages and `snake_case` keys.
- `depguard` keeps testify and testcontainers in test files and test-only packages, and Gin and GORM out of the verification agent's target paths.
- `exhaustruct` checks only types named `Deps`, through its `include` setting, as `backend/AGENTS.md` "Dependencies" requires. Checking every struct literal would force zero values to be written out everywhere.
- `exhaustive`, `revive`, `nolintlint` with a required explanation and no unused directives, `usetesting`, `testifylint`, `thelper`, `gochecknoinits`.

The exact patterns are part of the implementation. They are proven with a temporary probe: a new directory `internal/modules/lintprobe` with a production file and a `_test.go` file that break one rule of each linter, plus a production file with an ignored error under an old root. The probe shows that every rule fires in a new module directory without a configuration edit, that old code still fails on what it failed on before, and that old code passes the new rules. It is deleted after its output is recorded, because a permanent violation inside the lint's scope would fail `just lint`.

### The agent keeps its own lint configuration for now

The agent keeps `agent/verification/.golangci.yml` until the agent milestone moves it into the backend module. The backend file is rewritten with LF endings. A shared configuration was tried and does not work: golangci-lint passes `gofumpt.module-path` to gofumpt as it is and never reads `go.mod`. Without the setting, gofumpt treats `databasus-backend/...` and `databasus-verification-agent/...` as standard library imports, because their first path element has no dot, and regroups them. One value of the setting is right for only one of the two modules.

Alternatives:

- Rename both modules to paths with a dot, such as `databasus.dev/backend`. gofumpt then needs no setting, but every import in both modules changes, in code the later milestones move anyway.
- Merge the agent into the backend module now. That is the agent milestone's work and moves code, which this change does not do.

The cost is that every linter added before the agent milestone goes into the agent's copy too, if the agent should get it; until then the agent has no code in the target layout, so the copy needs none of the new linters.

### The target layout expression is stated once and checked

`backend/.golangci.yml` defines the plain target layout expression behind a YAML anchor and reuses it wherever a rule needs it. The exclusion for old test files cannot reuse it: golangci-lint forbids `path` and `path-except` in one rule and RE2 has no lookahead, so that rule spells out "outside the target layout and not a production file" as one expression with a comment. `archlint` holds the same expression as a Go pattern. A test in `archlint` reads `backend/.golangci.yml` and fails when any target layout expression there differs from the pattern, so the lint and the architecture check cannot drift apart.

### `archlint` is a Go program in `backend/tools/archlint`

The tool loads packages with `golang.org/x/tools/go/packages`, already an indirect dependency, and reports each violation as `file:line: rule: message`. `just lint` in `backend/` runs it after `golangci-lint`, and the CI `lint-backend` job gets a step that runs `go run ./tools/archlint` after its `golangci-lint run` step (`ci-release.yml:61-64`), because that job calls golangci-lint directly rather than `just lint`. Without the step the provider-write guard, which CI runs today as a test, would leave CI with the deleted test. The lint and format recipes of `backend/Justfile` and the backend check in `.agent-hooks/review_gate.py` add `./tools/...` to `./cmd/... ./internal/...`, so the tool's own code is linted under the target layout's rules. Its rules:

1. Layer direction and module boundary: imports follow `cmd → app → modules → engines → lib`; a module's root package holds only `module.go`; another package imports a module only through its root package; the agent's target paths import only `lib`, `engines` and the agent client. Target layout only.
2. File roles: methods of a type whose name ends in `Handler`, `Service` or `Repository` live in files whose names end in `handler.go`, `service.go` or `repository.go`; package-level `err...` variables and error types live in `errors.go`. Target layout only.
3. A bare `go` statement appears only in `internal/lib`. Target layout only.
4. Direct provider writes: the check of `no_direct_provider_writes_test.go` moves here unchanged and keeps its scope over the old backup code in `internal/features/backups`; the guard test is deleted. The target layout needs no copy of it: providers live inside the storages module there, and rule 1 keeps other modules out of its internals.
5. Cascades: a migration newer than the newest migration at the time of this change that declares `ON DELETE CASCADE` from a table to a table of another owner fails. A table's owner is a module name from the module table in `docs/architecture.md` when the table name equals it or starts with it and an underscore, or `lib_<name>` for a shared library table. A key that involves a table without an owner, such as today's `postgresql_databases` or `backup_configs`, is skipped, so an interim feature on old tables is not blocked. The cutoff version is a constant in the tool, because older migrations predate the scheme. The module names are a list in the tool next to the cutoff.
6. Production dependencies: the packages that `go list -deps` of each production entry point lists must not import `testing`, `github.com/stretchr/testify/...` or `github.com/testcontainers/...`. The entry points are a list in the tool: `./cmd` today, and `./cmd/databasus` and `./cmd/verification-agent` when they appear. Packages are loaded without build tags. Two third-party packages are accepted by name, each with its reason in the tool: `github.com/Files-com/files-sdk-go/v3/lib` and `github.com/rclone/Proton-API-Bridge`, which rclone's Files.com and Proton Drive backends link and which import `testing` or testify in production files. Removing them would drop two storage backends users have; any other importer, ours or third-party, fails the check.
7. Files with the `testhelpers` build tag appear only outside the target layout.

Rules 1, 2 and 3 check production files only: a module's API tests build the whole application through `apptest`, which sits above the modules, and tests start goroutines of their own to drive concurrent calls. A file role matches a type named after the role, such as `StorageService` or `storageRepository`, and the bare unexported `service`, `handler` and `repository`; the file is `<role>.go` or `*_<role>.go`.

Each rule has a test with a fixture tree that violates it and one that passes.

Alternatives:

- Express the rules as `depguard` and `forbidigo` patterns. They see one import or one call at a time and cannot check file roles, the module root package or a dependency graph.
- A shell script over `go list`. It can check imports and the graph but not file roles or receiver types, and it is harder to test.

### Old test helpers move behind a build tag

The 27 `testing.go` and `mocks.go` files, `DatabaseService.DeleteForTest` moved into the `testing.go` of its package, and the files of the test-only packages `internal/features/tests/logical/shared` and `internal/features/tests/physical/postgresql/shared`, which call those helpers, get `//go:build testhelpers`. Tagging the two packages keeps `go build ./...` and `govulncheck ./...` working without tags. Every command that runs or checks backend tests passes `-tags testhelpers`: `just test-package` and the other test recipes in `backend/Justfile`, `scripts/test.sh`, the CI test step, and `run.build-tags` in the lint configuration. The gopls server gets the tag through a wrapper in `tools/lsp`, the way the TypeScript server already gets its settings, so an editor started from a shell where mise put `tools/lsp` on `PATH` sees the helpers. Claude Code's gopls plugin starts `gopls` through the mise shim, which runs the installed binary directly and never looks in `tools/lsp`, so `.claude/settings.json` sets `GOFLAGS=-tags=testhelpers` for Claude Code sessions. Every `go` command in such a session sees the tag too; `archlint` passes `-tags=`, which overrides it, so its production check is unaffected. Production builds, including the image build and `archlint`'s dependency check, pass no tags and so link none of these files. Helpers in the target layout follow `docs/testing.md` instead: `_test.go` files or packages that only tests import, never the tag, which rule 7 enforces.

Alternatives:

- Move cross-package helpers into new test-only packages. Many helpers read unexported state, so they would have to be exported as production API first, in code that the module milestones replace with `apptest`. The diff would touch every test that imports a helper.
- Move them into `_test.go` files. Test files cannot be imported from another package, and most helpers are used across packages.
- Leave them until each module moves. The production binary would link test libraries for eleven more milestones, and rule 6 could not be enabled.

### The race detector everywhere tests run

`-race` is added to the CI backend test step, to `just test-package`, to the other test recipes and to `scripts/test.sh`. CI keeps `-timeout 40m` at first; the shard weights in `test-shard-weights.txt` are re-measured if a shard comes close to it.

Alternatives:

- Race only in a nightly job. A race introduced by a pull request would then surface after merge, detached from the change that caused it.

### Dependency checks in the lint jobs

`go mod verify` runs right after `go mod download` in both lint jobs, before golangci-lint is installed. `govulncheck` is installed with `go install golang.org/x/vuln/cmd/govulncheck@<version>`, pinned to the release current at implementation time, the same way both jobs install golangci-lint (`ci-release.yml:51` and `:206`). It runs as `govulncheck ./...` in its default symbol-level mode, which reports only vulnerabilities in code the module calls. It stays out of `just lint`, because it needs the network and the vulnerability database, while `just lint` runs offline in the review hooks.

The first scan with Go 1.26.3 reports reachable vulnerabilities in both modules: 27 in the backend and 22 in the agent, almost all in the standard library, the rest in `golang.org/x/net` and `github.com/moby/go-archive`. The change therefore moves Go to 1.26.10, the newest 1.26 release, wherever it is pinned: `mise.toml`, the `go` directive of both `go.mod` files, the `golang` images in the Dockerfiles and every `go-version` in CI. It upgrades the two modules to their fixed versions. Go 1.27 is a new minor release and stays out of this change.

Alternatives:

- A third-party GitHub Action for `govulncheck`. Another action to pin and trust, for a command `go install` already provides.
- Pin it in `mise.toml`. The lint jobs do not use mise, and adding the mise action to them only for this tool adds a step and a second way to install tools in one job.

## Risks / Trade-offs

- [The race detector finds races in old code, and CI fails on a test this change did not touch] → A race is a real bug: it is fixed in this change, never hidden with a skip or a suppression. Races the change's own test runs find are fixed before the change is handed over; races that only CI finds are fixed in this change before it is archived.
- [Tests under `-race` run several times slower, and a shard reaches the 40-minute timeout] → Re-measure the shard weights; raising the timeout is the fallback and is noted in the change.
- [A forgotten `-tags testhelpers` in some command breaks compilation of the old tests] → The failure is a compile error naming the missing helper, never a silent pass; `just test-package` and CI share the tag through the recipes.
- [`-race` needs cgo and a C compiler on the developer's host] → Without one, `go test -race` stops at once with an error that names cgo, so the cause is visible on the first run.
- [The cascade rule's cutoff constant is forgotten when the first prefixed tables arrive] → The rule is tested with a fixture migration after the cutoff, and the module milestones rename tables in migrations that are newer than the cutoff by construction.
- [The broad exclusion for old code hides a new violation written into an old path] → The architecture review checks that new code goes into the target layout, and each module milestone lifts the exclusion for its module.
