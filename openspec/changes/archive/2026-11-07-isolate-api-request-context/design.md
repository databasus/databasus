## Context

See `proposal.md` for motivation and scope. The current implementation has these relevant properties:

- `backend/internal/lib/httpserver/request_context.go:13` unwraps a router context in each caller. Its existing tests exercise the helper, not the generated adapter.
- `backend/internal/modules/workspaces/internal/httpapi/workspace_handler.go:14` shows the repeated conversion followed by actor lookup. Eight API packages define their own lookup.
- `backend/internal/app/systemhttpapi/handler.go:11` starts the four system handlers that pass the context through unchanged. `health_service.go:85` derives a child deadline from it and calls the database.
- `backend/internal/app/systemhttpapi/api.gen.go:895` shows that the generated adapter passes `*gin.Context` to the application handler. Its strict middleware also requires `*gin.Context`.
- `backend/internal/app/server.go:111` enables `ContextWithFallback`. It exposes request values through Gin but does not stop Gin from reusing the context object.
- `backend/internal/lib/database/transaction.go:78` returns the transaction without rebinding its statement context.
- `backend/internal/modules/users/module.go:22` imports the users HTTP package, while `module.go:53` exposes identity context functions. Importing the public users package from its own HTTP package would create a cycle.
- `backend/internal/modules/healthcheck/internal/httpapi/errors.go:5` and `backend/internal/modules/verification/internal/httpapi/errors.go:5` return a missing-actor Problem; the other six lookups use an ordinary internal error.
- `mise.toml:35` runs package-local `go generate` for backend generation. `mise-tasks/backend/check-generated:21` instead regenerates from the backend root with temporary configuration files.

Root `AGENTS.md` and `backend/AGENTS.md` govern the change. The module boundary, handler/service/repository roles, transactions carried in context, and generated-source ownership follow `docs/architecture.md`, especially "How modules talk", "Roles inside a module", "Data" and "HTTP API".

## Goals / Non-Goals

**Goals:** Make the adapter own request-context conversion, keep actor lookup in its owning module, and honor the caller's context without changing the transaction connection. Keep both implementation groups independently correct and verifiable.

**Non-Goals:** Do not add context wrappers, a second generation framework, a new timeout policy, dependency upgrades, or unrelated handler abstractions. The proposal lists the external behavior excluded from this change.

## Decisions

### Bind queries to the current context inside the existing transaction

Return `transaction.WithContext(ctx)` from the transactional branch of `database.Conn`. This retains the transaction's connection while updating the context used by the next query. Do not mutate the shared transaction's statement or replace the transaction with the pool.

Keeping the transaction's opening context would continue ignoring child deadlines. Moving deadline handling into each repository would repeat the same omission risk and violate the shared connection convention. Changing how services open transactions is unnecessary.

### Keep one required actor lookup in the identity package

Implement `RequireActor(ctx)` alongside `GetActor` in `users/internal/identity`, with its missing-actor sentinel in that package's `errors.go`. Expose it through `users.RequireActor` in `module.go`. The users HTTP package calls the identity implementation directly; other modules use the public contract. Delete the eight local `getActor` functions and the redundant ordinary missing-actor errors.

Preserve current error responses: the common lookup returns the ordinary invariant error used by six packages. Healthcheck and verification translate the failed lookup to their existing unauthenticated Problem at their HTTP boundary. This small error conversion is not another actor lookup. Keep authentication middleware responsible for rejecting missing or invalid sessions before a handler runs.

Implementing the lookup only in `module.go` creates an import cycle for users handlers. Moving it into `lib/httpserver` gives a transport library ownership of user identity. Returning a new HTTP error from every caller silently changes the existing missing-actor behavior.

### Convert the context in the generated adapter

Add one local `strict-gin.tmpl` under `backend/api/templates`, based on the exact template shipped with the pinned `oapi-codegen` version, currently v2.8.0. Keep the override minimal: change the application call from `sh.ssi.Operation(ctx, request)` to `sh.ssi.Operation(ctx.Request.Context(), request)`. Retain `*gin.Context` for decoding, strict middleware, error hooks and response writing.

Reference this template through `output-options.user-templates` with the exact key `strict/strict-gin.tmpl` in every configuration that enables strict Gin server generation, including system operations and the agent API. The generator retains the upstream template's directory in its name; the bare filename would silently register an unused template. Do not attach a server template to client-only configurations. Document the upstream template location/version and the requirement to compare and rebase the one intentional difference when upgrading the generator in `docs/architecture.md`.

Strict middleware cannot pass a plain request context because its handler type requires `*gin.Context`. Keeping manual conversions does not prevent omissions. Switching generation targets or wrapping every strict interface method would expand the change and duplicate operation declarations.

### Use the package directory consistently when generating and checking output

Make template paths relative to each owning generator package. Keep `mise run generate-api` as the public command; its existing package-local `go generate` already provides that working directory. Update `backend:check-generated` to invoke each generator from the corresponding package directory as well, with absolute temporary output/config paths and a contract path resolved for that invocation. Keep temporary-file cleanup reliable on failure.

This makes template lookup identical in normal generation and lint, despite temporary configs living outside the package. Do not rewrite the public command merely to change its spelling. Backend-root-relative paths would work for the current checker but break `go generate`; absolute checkout paths would not be portable.

### Keep the two commit boundaries safe

The user requested separate code and configuration commits. Treat them as these implementation groups; committing is outside the task list:

1. **Code and regression repairs** — proposed subject: `FIX (backend): Preserve request and transaction contexts`. Fix `database.Conn`, consolidate actor lookup, and add the missing manual conversions to the four system handlers. Keep existing conversions and the helper because generated adapters still pass Gin contexts. Add and verify the database and system-handler regressions in this group.
2. **Generation and dependent cleanup** — proposed subject: `REFACTOR (api): Generate handlers with request contexts`. Add the template and generator configuration, repair generated-output checking, regenerate all server adapters, add adapter regressions, then remove every manual conversion and delete the helper. Update architecture documentation and the comment describing context forwarding in `app/server.go` to agree with the final behavior.

The second group necessarily contains regenerated Go files and handler cleanup along with settings: removing conversions in the first group would leave it unsafe. A settings-only second commit cannot deliver the requested safety without coupling it to those code changes. No generated file is edited manually, and no failing adapter regression is left in the first group. OpenSpec completion and archival remain at the normal pre-commit boundary; this plan does not authorize an early partial archive.

### Test observable context behavior through real boundaries

For each defect, add its regression before the fix and record failure for the intended reason. Use the real internal database for `TestConn`: exercise an expired child deadline, explicit child cancellation, and writes through a live child context followed by rollback. A query with an already-expired deadline gives a deterministic failure on the unfixed implementation without sleeps. Also cover a deadline expiring during a running query, with a server-side query limit bounding the unfixed run; use the actual slow query as the operation under test rather than a fixed sleep in test code.

In the first group, exercise the system health handler through HTTP and inspect the context delivered to an existing observable dependency boundary, such as its log sink. Assert that it is the request's own context rather than the reusable Gin object. Use the real service/application, not a mocked service.

In the second group, add a transport regression under `lib/httpserver` in an external test package using the existing public system generated adapter. A small strict-interface probe captures the context passed by the generated adapter; it replaces no domain service. Feed requests with distinct values, actor identities and cancellation signals, explicitly reuse the same Gin object after the first dispatch, and prove the first captured context retains its own values and cancellation. Also assert the captured object is the request context and never a Gin context. This test fails before the template is connected, even if application handlers still unwrap their contexts.

Exercise the actual generated agent adapter with an equivalent transport probe in its owning verification test scope, where Go permits importing that internal package. Existing HTTP authentication tests remain the acceptance surface for missing sessions and agent tokens. Run existing module API tests without weakening or changing their expectations.

Testing only `GetRequestContext` would pass while a caller forgets it. Relying on nondeterministic pool reuse or repeated race runs would make the regression unreliable. A handwritten substitute adapter would not test the template's output.

## Risks / Trade-offs

- The full upstream strict template must be kept locally for a one-line semantic change. Mitigate drift with pinned tools, a documented upstream comparison on upgrades, adapter regressions and generated-output lint.
- Template path resolution differs between package-local generation and the current checker. Align working directories before relying on the override; verify generation and stale-output detection from both the repository root and a nested directory.
- Consolidating actor lookup can accidentally change HTTP errors. Keep the two existing boundary conversions and run authentication/access API regressions without changing their expectations.
- Removing the helper too early exposes unsafe handlers. Keep all existing conversions through the first group and remove them only alongside regenerated adapters in the second group.
- A large generated diff can hide an unrelated change. Review it for the intended context argument replacement and require a second generation to leave the tree unchanged.

## Migration Plan

No database migration or external client migration is needed. Prepare and verify the two groups in dependency order. The first remains usable with the old generator; the second establishes the final invariant across both contracts. Reverting the second group must restore its handler conversions together with its adapters and settings. Reverting only template settings or only handler cleanup is not a valid rollback.
