## Regression checks

- `go -C backend test -count=1 -run TestConn ./internal/lib/database` failed before the fix: an expired child deadline and an explicitly cancelled child still permitted writes, and the running query reached PostgreSQL's statement timeout instead of its child deadline. After rebinding transactional queries, `go -C backend test -count=1 ./internal/lib/database` passed, including transaction, savepoint and after-commit coverage.
- `go -C backend test -count=1 -run TestSystemRequestContext ./internal/app/systemhttpapi` failed before the system-handler fix because the dependency log sink received a Gin context. It passed after the fix; the complete system HTTP package also passed.
- `go -C backend test -count=1 -run TestStrictRequestContext ./internal/lib/httpserver ./internal/modules/verification/...` failed before connecting the template. Reusing one Gin object made the first captured context report the second request's values, actor, deadline and cancellation. Both generated-adapter regressions passed after regeneration.

## First implementation group

- Targeted tests of users and its internal packages, workspaces, notifiers, storages, databases, backups, verification and healthcheck passed with existing API assertions unchanged.
- `mise run backend:format` and `mise run backend:lint` passed.
- `mise run test` passed before template/configuration changes. The frontend reported 189 test files and 2616 tests passing.

## Generation and final checks

- `mise run generate-api` succeeded from the repository root and from `backend/internal/lib/httpserver`. Repeated generation preserved the hashes of all ten changed generated servers.
- Compared every changed generated file with the base version: the only differences are passing `ctx.Request.Context()` to application handlers.
- `mise run backend:check-generated` rejected a controlled extra newline in the system adapter, named that stale file, and passed after the original bytes were restored.
- `mise run backend:format`, `mise run backend:lint`, `shellcheck mise-tasks/backend/check-generated`, `git diff --check` and `openspec validate isolate-api-request-context --strict` passed.
- Targeted HTTP/library tests across all affected modules passed after removing manual conversions. No existing API assertion was changed.
- The final `mise run test` passed across both backend groups and the frontend (189 files, 2616 tests). Full implementation review accepted the change. After formatting adjustments, targeted `TestConn` and `TestStrictRequestContext` checks and backend lint passed again.
