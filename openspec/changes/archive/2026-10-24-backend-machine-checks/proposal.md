## Why

`docs/architecture.md` lists the rules that a machine checks, and almost none of those checks exist yet. The linter skips test files, the architecture linter does not exist, the production server binary links `testing`, testify and testcontainers through 27 helper files, and backend tests run without the race detector. Modules start moving into the new layout in the next milestones, and every module that moves before the checks exist repeats the old mistakes, with the reviewers arguing against documents that nothing enforces.

## What Changes

- The backend lint checks test files and gains the linters `docs/architecture.md` names: `forbidigo`, `sloglint`, `depguard`, `exhaustive`, `exhaustruct` for `Deps` literals, `revive`, `nolintlint`, `usetesting`, `testifylint`, `thelper`, `gochecknoinits`. The new rules, and the existing linters on test files, apply to code in the target layout. Code outside it keeps exactly the golangci-lint checks it has today, until its module moves; three `archlint` rules below reach it too.
- A new architecture linter, `archlint`, runs in `just lint` and in the CI backend lint job. It checks layer direction, the module boundary, file roles, errors in `errors.go`, bare `go` statements, cascading foreign keys to another module's tables, and the absence of test libraries in the production binary. It takes over the guard test that keeps backup code from writing to storage providers directly.
- Test helpers leave the production binary. The old `testing.go` and `mocks.go` files, the two test-only `shared` packages under `internal/features/tests` that call them, and the production method `DeleteForTest` compile only with the `testhelpers` build tag. Testcontainers leaves the binary entirely; `testing` and testify stay only through two packages that rclone's backends link, which `archlint` accepts by name with the reason. Every command that runs or lints backend tests passes the tag, and so does the gopls wrapper, while production builds never do. The backend lint and the review hook's backend check also cover `backend/tools`.
- The backend lint configuration is stored with LF line endings and states the target layout expression once; an `archlint` test keeps it equal to the pattern `archlint` uses. The verification agent keeps its own configuration until it moves into the backend module, because golangci-lint does not read `go.mod` for gofumpt and one shared `gofumpt.module-path` fits only one of the two modules.
- Backend tests run with the race detector in CI and in `just test-package`. A data race fails the test.
- CI checks the backend and the agent with `govulncheck` and `go mod verify`. Go moves from 1.26.3 to 1.26.10, and `golang.org/x/net` and `github.com/moby/go-archive` are upgraded, so the scan starts with no reachable vulnerability.

## Capabilities

### New Capabilities

- `backend-code-checks`: which rules the backend lint enforces, where they apply during the move to the new layout, and that the production binary contains no test code.

### Modified Capabilities

- `ci-pipeline`: backend tests run with the race detector, the backend lint job runs the architecture check, and the lint jobs check dependencies for known vulnerabilities and verify module checksums.

## Impact

- `backend/.golangci.yml`, `backend/Justfile`, `scripts/test.sh`, `.github/workflows/ci-release.yml`.
- The Go version in `mise.toml`, both `go.mod` files, `Dockerfile`, `agent/verification/e2e/Dockerfile.*` and `.github/workflows/codeql.yml`, and the upgraded `golang.org/x/net` and `github.com/moby/go-archive`.
- New `backend/tools/archlint` with its tests; `golang.org/x/tools` becomes a direct dependency of the backend module.
- The 22 `testing.go` and 5 `mocks.go` files under `backend/internal`, the files of `internal/features/tests/logical/shared` and `internal/features/tests/physical/postgresql/shared`, and `DatabaseService.DeleteForTest` in `backend/internal/features/databases/service.go`.
- The dependency paragraph of the README's security section, in `README.md` and the five `assets/readme/README.<locale>.md` copies, which names the new Go checks.
- `tools/lsp` (a gopls wrapper that passes the build tag), `.claude/settings.json` (the tag for Claude Code's gopls), `.agent-hooks/review_gate.py` (the backend and agent lint checks), and the documents with test commands: `AGENTS.md`, `backend/AGENTS.md`, `docs/testing.md`.
- `backend/internal/features/backups/no_direct_provider_writes_test.go` is removed once `archlint` covers it.
- Data races that the race detector finds in the tests this change runs are fixed in this change.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md`, the testing skill and `docs/architecture.md`.

## Out of scope

- Moving code into the new layout, and lifting the exclusions for any module.
- `gochecknoglobals`, which is enabled once no package-level dependencies remain.
- `importas`, which is enabled with the generated API packages that need fixed aliases.
- Fixing what the new linters would report in code outside the target layout.
- The contract linter and the generated-code check, which arrive with the contract.
- Agent instructions, review hooks and documents other than the ones listed in Impact, which the `agent-rules-and-reviews` change covers.
- One lint configuration for the backend and the agent, which arrives with the agent milestone, when one `go.mod` remains.
