## Why

API handlers must manually replace a reusable router context with the context of the current HTTP request. The code has 104 calls to that conversion, but all four system handlers omit it; separately, internal-database queries inside a transaction ignore a caller's shorter child deadline.

## What Changes

- Bind transactional queries to the caller's context while retaining the current transaction.
- Add one actor lookup that requires an authenticated actor, implement it in the users module's identity package, and expose it through `users.RequireActor`. Replace the eight local implementations without changing existing error responses.
- Fix the four system handlers before changing generation, keeping the initial code changes independently usable.
- Override the strict-server template so every generated user and agent API adapter passes `ctx.Request.Context()` to its application handler.
- Connect the shared template to generation and stale-output checks, regenerate the adapters, and then remove manual context conversion and its helper.
- Add regression coverage for transaction deadlines and request isolation through generated adapters; retain existing API test expectations.

## Capabilities

### New Capabilities

- `internal-database-cancellation`: Internal-database queries honor the caller's deadline and cancellation while joining an existing transaction.

### Modified Capabilities

- `api-http-server`: Handler execution uses the current request's cancellation, metadata and authenticated identity without retaining the router's reusable context.

## Impact

- Governed by root `AGENTS.md` and `backend/AGENTS.md`.
- Affects `backend/internal/lib/database`, `backend/internal/lib/httpserver`, the users module's identity and public contract, system handlers, and the user and agent API adapters.
- Generation work includes a shared template under `backend/api/templates`, the strict-server generator configurations, `mise run generate-api` and the generated-output check. Update the HTTP section of `docs/architecture.md` with template ownership and the generator upgrade procedure.
- Deliver the work in two independently verified groups: application code and tests first; generator configuration, regenerated adapters and dependent cleanup second. The design defines the commit boundaries; the implementation task list ends before committing.
- No dependency upgrades or external HTTP contract changes are intended. Deleting the internal context helper updates every repository caller in the same group.

## Out of Scope

- Changing paths, payloads, access marks, workspace authorization, or existing missing-actor HTTP responses.
- Replacing the router, switching server generation targets, or introducing middleware to impersonate a request context.
- Changing transaction ownership, nested-transaction behavior, after-commit hooks, or adding new global deadlines.
- Broader handler refactoring, CI redesign, or editing generated source by hand.
