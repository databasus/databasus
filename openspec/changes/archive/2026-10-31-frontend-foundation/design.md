## Context

See `proposal.md` for motivation and scope. Root `AGENTS.md`, `frontend/AGENTS.md` and the testing skill constrain implementation. The frontend architecture added here must agree with the HTTP contract and error rules in `docs/architecture.md:247-252`; architecture changes and their reasons belong in that document in the same change.

The relevant starting points are:

- `frontend/AGENTS.md:168-241` describes page-first placement, public APIs and downward imports, but also places domain CRUD in shared infrastructure. `frontend/AGENTS.md:249-252` ties verification to a line-count threshold. The architecture document has no frontend layout section (`docs/architecture.md:29`, `docs/architecture.md:245`).
- `frontend/src/app/ui/App.tsx:15-65` composes locale, theme, AntD configuration and browser routing. `frontend/src/shared/lib/test/renderScreen.tsx:10-22` only cleans up rendering/local storage and wraps locale plus AntD App.
- `frontend/src/shared/lib/test/mockApi.ts:35-54` builds typed JSON responses and one GET-only empty response. Its lifecycle resets handlers, but has no independent unexpected-request ledger (`mockApi.ts:72-90`). Blanket pending handlers occur in `frontend/src/features/databases/ui/DatabasesComponent.test.tsx:69-71` and `frontend/src/features/databases/ui/DatabaseConfigComponent.test.tsx:33-37`.
- The API barrel imports the old helper (`frontend/src/shared/api/index.ts:1-4`), which creates a limiter at import time (`apiHelper.ts:8`); the limiter starts an interval without disposal (`RateLimiter.ts:12-15`). Active calls use `apiClient.ts` and `requireData.ts`. The client currently installs session middleware only (`apiClient.ts:11-38`), and response normalization begins after the request resolves (`requireData.ts:21-44`).
- Unknown API codes fall back to raw error text (`frontend/src/shared/i18n/translateApiError.ts:62-68`), and two tests expect that behavior (`translateApiError.test.ts:43-62`). The contract explicitly permits connection diagnostics and required/free disk bytes (`backend/api/openapi.yaml:1633-1637`, `backend/api/openapi.yaml:2689-2693`).
- Full-list cursor traversal is repeated, for example in `frontend/src/entity/workspaces/api/workspaceApi.ts:10-25`. The paging controller increments a generation on reload but retains the old cursor (`PagedListController.ts:43-53`); next-page loading does not check initial loading, and its finally block always publishes state (`PagedListController.ts:100-118`). The hook has no request invalidation on unmount (`frontend/src/shared/hooks/usePagedList.ts:38-56`).
- Frontend lint already checks formatting, Oxlint, types and generated freshness (`mise.toml:130-138`). Node and pnpm are pinned in `mise.toml:5-6`, but the test job specifies Node 20 (`.github/workflows/ci-release.yml:144-152`). The architecture prompt points only to frontend AGENTS (`.agent-hooks/architecture-prompt.md:5-7`); verification forms currently allow the frontend Vitest command only (`.agent-hooks/review_gate.py:236-280`).

These observations come from source inspection. The earlier analysis recorded 595 passing tests; implementation must measure its own starting inventory and results rather than treating that historic count as fresh evidence.

## Goals / Non-Goals

**Goals:** make the later domain moves verifiable, remove unused infrastructure, and repair demonstrated error/list lifecycle defects while preserving existing screens, addresses and domain operations.

**Non-Goals:** see the proposal's exclusions. In particular, this change must not move full screens merely to make the baseline smaller, redesign tokens while adding providers, or introduce runtime validation of every successful response against an entire schema.

## Decisions

### 1. Give architecture, agent conventions and workflow separate owners

Add a Frontend section to `docs/architecture.md` covering layer direction, page-first placement, composition in app, slice public APIs, state ownership, domain API versus transport, and the reasons for these choices. Keep UI conventions and existing localization rules in `frontend/AGENTS.md`; replace its FSD exposition with links and skill triggers. Remove its line-count verification threshold in favor of checks based on the files and behavior changed. Do not move or rewrite the full localization workflow in this milestone.

Use `app -> pages -> widgets -> features -> entities -> shared`. App owns screen composition; reusable shells receive slots/children instead of importing pages. Domain HTTP operations remain in the domain or scenario that owns their meaning; shared owns the transport and domain-independent algorithms. A generated schema alias is not a second handwritten DTO. Closely coupled shape aliases may share a logical unit; independently meaningful state types remain separate under the existing naming rules. Place hooks by responsibility: generic lifecycle machinery may be shared, domain behavior stays with its owner, and application lifecycle belongs in app. The existing `useVersionCheck` upward dependency is recorded for milestone 3 rather than moved here.

Add `.agents/skills/feature-sliced-design/` from the official `feature-sliced/skills` upstream, recording the full source revision, license and attribution alongside the vendored material. Inspect it before adoption and keep project-specific overrides explicit; upstream methodology cannot override root or module rules. Link to architectural policy rather than duplicating it in the skill. Updating the architecture review prompt is part of the same move.

**Alternatives rejected:** leaving architecture only in AGENTS keeps review tied to an instruction inventory and duplicates policy; copying the whole upstream methodology into AGENTS creates another competing source. Moving every documentation topic now would expand this milestone into design/localization work.

### 2. Use an architecture checker with a finite, independently tested baseline

Prototype Steiger and `@feature-sliced/steiger-plugin` against the pinned Node/TypeScript stack. Record the selected exact versions and compatibility evidence in `coverage.md`, then add a `lint:fsd` script and invoke it from `frontend:lint` alongside existing checks. Keep Oxlint as the syntax/style checker. Do not upgrade the application stack to make the prototype pass. If an individual required import form is unsupported, add only the narrow missing check using the existing TypeScript parser; do not replace the checker or relax the requirement.

The baseline stores each existing finding by rule, importer and target or structural location, plus owner and removal milestone. Match semantic identity, not line number; enforce occurrence counts so copying the same violation within a file is not free. A new occurrence, changed dependency or stale baseline entry fails. No whole-layer or whole-file suppression is allowed. Store the checker configuration/baseline under `frontend/`; keep executable fixture tests under `frontend/src/` so the existing Vitest command includes them. Define each existing nested domain group's concrete current boundaries in the migration inventory, and enumerate every dependency the target flat-slice check would reject; neither Steiger treating a group as one slice nor the baseline may silently exempt its descendants. The baseline covers those exact current findings until their domain milestone flattens the group. New sibling directories inside those migration groups automatically form separate boundaries. Deduplicate a core and supplemental report only when they describe the same AST dependency occurrence; a re-export beside an import remains a second violation. Keep dependency discovery and its syntax/resolution helpers in `frontendDependencies.ts`; keep project boundary checks and their layer/slice helpers in `checkProjectDependencies.ts`. The runner composes these independent units.

Fixtures exercise allowed downward imports and forbidden upward, feature-to-feature and private deep imports, plus type imports, dynamic imports and laundering through re-exports. A slice's public API does not authorize re-exporting a forbidden dependency. Do not require new entity cross-import exceptions: new cross-slice dependencies remain forbidden in this change.

Rename the entire `entity` directory and its import specifiers to `entities` as a behavior-preserving step, then configure `@/*` in TypeScript and `@/` resolution in Vite/Vitest. Keep valid existing relative imports; new checker coverage must resolve both. Validate all four consumers with the same alias fixture, not only a TypeScript compile. Nested groups are not flattened here, and removed import paths receive no compatibility alias.

**Alternatives rejected:** broad glob ignores permit new violations; line-number baselines churn on unrelated edits; a second general-purpose lint framework duplicates existing tools. A full domain reorganization would mix behavioral edits into the mechanical rename.

### 3. Strengthen the existing network mock instead of replacing it

Keep typed responses derived from generated operations. Extend the existing facade for JSON success, empty success, declared errors, network failures, malformed external responses and test-controlled delayed responses. Add typed request inspection for path/query/body and a release/abort lifecycle for pending handlers. Type checks must reject the wrong response body and non-empty bodies on empty success operations. Keep raw MSW handlers available for exceptional responses and dynamic scenario logic, but require an explicit HTTP method and finite pathname pattern. An origin wildcard and named path parameters are acceptable; an all-method handler or pathname wildcard is not.

The shared lifecycle validates declarations and keeps an independent per-test request ledger. Register an unexpected request when it starts or cannot match an allowed declaration, before a handler can hang indefinitely. A network rejection caught by the component cannot erase this record. Reject blanket handlers when registered, including a blanket delayed handler; the response-matching event alone is insufficient. Assert the ledger during teardown as well as reporting the request promptly.

Replace both database tests' blanket handlers with their finite scenario dependencies. A request can deliberately remain pending only through an explicit declaration; its controller is released or aborted at teardown. Do not globally swallow rejections, mute the console or skip failing consumers to migrate the harness. Match a declared pathname separately from an origin wildcard so it cannot consume additional path segments. Raw deferred handlers use the same cleanup-capable response controller. A real-clock cleanup deadline diagnoses unowned pending handlers, and a finally block always clears handlers, listeners, records and overrides.

Verify actual runner outcomes with an isolated fixture runner: one fixture catches an undeclared request and must still exit unsuccessfully; another registers a blanket pending handler and must fail; an isolation fixture pair proves that handlers, recorded requests and pending work do not leak into the next test. Run these expected-failure fixtures as child Vitest processes from a node test, with scenario and environment selected through `IsolatedScenarioOptions`, temporary configuration and cleanup, so the main suite stays green only when it observes the required nonzero exit and diagnostic. Do not add expected-failing fixtures to ordinary test discovery.

**Alternatives rejected:** `onUnhandledRequest: 'error'` alone only proves a rejected request, not that a screen test fails after a catch; checking only unhandled events misses blanket handlers; unbounded pending responses can leak across teardown.

### 4. Render screens with real providers below app

Extend `renderScreen` with locale, theme and router-entry options. Compose the existing LocaleProvider and ThemeProvider, AntD ConfigProvider/App and an in-memory router. Resolve AntD locale and light/dark algorithms through the same existing providers and values the production app uses. Reuse the current primary-color configuration through a minimal shared configuration unit if necessary; token redesign belongs to milestone 2. Shared test code must not import app, a page or a domain slice. Whole-application tests stay in app and may use a separate wrapper that avoids nested routers.

Add user-event and drive controls by accessible role/name where possible. Keep node as the default environment and happy-dom for screen tests. Teardown unmounts before disposing request handlers, restores storage, document language/title/theme class, global stubs, locale state, timers and listeners owned by that test, and checks there is no work left behind. Provider tests cover two locales, both themes and an explicit router entry, including locale changes without remounting drafts. Keep timer control local to the tests that need it.

Strict cleanup exposed late error callbacks and unhandled rejections in the selected database reader, the database configuration's backup-config reader, and the healthcheck/logical-backup configuration views. Their reads now ignore completions after unmount or selection changes and handle current failures. This follows the standing authorization to repair defects discovered during this milestone; the controlled unmount regression and complete suite verify the repair. Database loading/reset depends on the selection; its error effect event reads the current translation without making language changes reload the database or discard drafts. The logical-backup list configuration reader also handles rejection and ignores completion after unmount.

**Alternatives rejected:** importing App into shared reverses layer direction and triggers unrelated startup requests; duplicating full application bootstrap in every screen test drifts; moving all tests to a DOM environment adds cost without helping pure logic.

### 5. Keep one active request path and safe error presentation

Before deleting `apiHelper.test.ts`, map its useful network/status/Problem guarantees to tests of the active client and real domain API calls. Preserve session header and 401 behavior already covered by `apiClient.test.ts`; session-generation redesign remains in milestone 3 unless a defect is exposed by this work. Remove the helper, request options, limiter, their exports and obsolete comments; importing the API barrel must no longer start their interval. Verify no production references remain before removing the unused visibility hook and URL validator; keep the email validator.

Normalize fetch rejection at the active request boundary to `network_unreachable`. Preserve intentional cancellation separately so unmount/reset does not produce a failure toast. Normalize unreadable JSON, non-Problem error responses and required-data responses with no body to `request_failed` with the status when available. Empty success operations continue to succeed. Neither failure normalization nor the cursor collector repeats a mutation. Do not introduce a new generated client or a generic CRUD wrapper.

Known Problem codes select translated copy. Unknown or absent codes select `errors.unknown` and never use raw message/title/detail/URL as fallback. Pass technical detail only for the explicitly supported known diagnostic errors, currently `database_connection_failed` and `insufficient_disk_space`, as plain text with translated surrounding copy. This preserves the contract's diagnostic behavior without making arbitrary errors visible. Replace the two raw-fallback expectations with the new contract. Add a hostile unknown code such as an inherited object-property name to ensure lookup cannot bypass the fallback.

**Alternatives rejected:** deleting old tests without mapping their guarantees loses useful coverage; restoring the old limiter/retry helper preserves unused state and unsafe replay; deleting every diagnostic detail hides actionable connection and capacity information. Runtime validation of every successful DTO is unnecessary for this task because backend responses already have contract tests.

### 6. Extract cursor traversal without moving domain operations

Add one small typed collector for a callback returning items and the next cursor. It accumulates pages in order until the cursor is absent, including an empty final page; it propagates a failure without exposing a successful partial list or retrying the failed request. Domain callbacks retain their paths, filters, page limits and request bodies. Apply it to workspace, database, storage and notifier full-list operations only. Other repeated collectors are outside the planned extraction unless a discovered defect requires repair.

Keep the pure collector separate from interactive paging. Prove the traversal with pure tests and retain domain API tests that inspect the actual workspace/filter/cursor requests. Do not move domain API facades into shared to reuse the loop.

**Alternatives rejected:** a configurable CRUD factory hides domain policy and adds an abstraction broader than traversal; independent copied loops continue to drift; using the UI paging controller for complete collection couples two different lifecycles.

### 7. Give paging operations explicit generation and request ownership

Preserve `PagedListController`, `appendNextPage` and `mergeRefreshedFirstPage`. Reproduce the reset/scroll race before changing them. A reset or reload invalidates old requests, makes the old cursor unusable and starts a new first-page request. Do not start next-page or polling refresh requests until that first page settles successfully. On failure, end loading, report one current error and leave reload available; never revive the old cursor.

List pages carry optional typed metadata alongside rows and cursors. The physical backup list reads its total usage from the accepted page metadata; fetch callbacks do not publish that aggregate before ownership is checked. Refresh replaces the summary with its accepted first-page value, while appending retains the current summary.

Each request owns its loading flag and completion. An old generation cannot append rows, change loading flags, report an error or hold a refresh lock for the new generation. Refresh stays within the current list, preserves its valid loaded tail and cursor, and does not overwrite newer results from a later reload. Existing merge/deduplication guarantees remain covered. A pending refresh of the old generation must not prevent a refresh of the new list.

The `backup-list-paging` delta explicitly qualifies tail preservation. A nonempty refreshed page with a continuation cursor preserves loaded rows below its last row only when that row anchors a loaded tail. An absent anchor, no tail below it, an empty page or a complete page makes the refreshed page and its cursor authoritative. This matches the fallback in `frontend/src/shared/lib/pagedList.ts:24-50`; enough new rows can make the old tail impossible to align without additional requests. Refresh still fetches only one page, and subsequent scrolling can reach older surviving rows from the refreshed cursor. The corresponding generic-list requirement and tests use the same fallback conditions. A fallback invalidates pending pagination only when its continuation cursor changes or disappears. An unchanged cursor keeps that request and its loading flag valid, including when refresh finishes before a slow next page.

Unmount invalidates pending callbacks and removes subscriptions, scroll listeners and polling timers. An effect setup after cleanup, including development StrictMode replay, can start a fresh active lifecycle; permanently disposing the one controller created by `useState` is insufficient. Test activation/cleanup replay, completion after unmount, overlapping refresh/next, reload/refresh response orders and stale-finally behavior. Use deferred responses and controlled clocks, not fixed pauses. Audit effect dependencies in the touched hooks while preserving a stable poller and scroll callbacks.

**Alternatives rejected:** checking the generation only around data writes leaves flag/lock races; one global busy flag unnecessarily blocks compatible operations; replacing all lists with a query library adds a migration unrelated to the demonstrated defects.

### 8. Align verification, CI and review without prematurely adding browsers

Use the same existing mise action and pinned Node/pnpm tools in the frontend test job as in lint/build. Keep frozen-lockfile installs and existing release dependencies. Test the workflow configuration locally; no push or CI run is needed to complete the task. Inventory currently disabled hooks rules from `frontend/oxlint.config.ts:10-14`; fix dependency/cleanup defects in touched lifecycles without enabling every rule across the frontend in the rename diff.

Extend the review-gate whitelist with exactly `pnpm --dir frontend test:browser --project=ui`, with no extra arguments for now. Keep shell-character validation, required verification evidence and parse-only behavior. Add positive/negative parser tests and a reviewer-prompt test for the authoritative frontend architecture source. Update the gate's docstring/help, `openspec/config.yaml` and affected review specs together. Browser setup, script creation and execution belong to milestone 2; this change's own verification block therefore names only the tests it actually introduces.

**Alternatives rejected:** separate hardcoded Node versions drift; an arbitrary `pnpm` whitelist weakens command validation; running a nonexistent browser script or introducing Playwright now makes this milestone depend on the next one.

## Risks / Trade-offs

- **The checker may not resolve an import form or existing group as expected.** Run compatibility fixtures before baselining, and supplement only a demonstrated missing boundary. Baseline entries cannot exempt entire group subtrees.
- **Strict network accounting may expose additional undeclared dependencies.** Add the real finite scenario handlers, reproduce any actual bug and repair it; never restore blanket handlers to get a green suite.
- **Fixture tests that must fail can pass for the wrong reason.** Assert both exit status and the unexpected-request/blanket-handler diagnostic, and include a successful control fixture.
- **Provider cleanup changes global browser state.** Capture and restore what the test changed, and prove isolation across consecutive tests rather than relying only on a teardown call count.
- **A mechanical move can lose tests or multiply exceptions.** Record paths and scenario ownership in `coverage.md`, preserve existing findings without increasing their counts, and review rename and behavioral edits separately.
- **Error and paging repairs change observable behavior.** Pin each changed scenario and preserve existing explicit diagnostics, cancellation semantics, deduplication and tail behavior; do not retain the obsolete raw-error fallback for compatibility.
- **Discoveries can grow the repair work.** The user explicitly authorizes fixing bugs found along the way in the same milestone. Record the reproduction, cause, affected artifacts and verification. Fix the cause, including legitimate timeout changes, rather than masking it with retries or weaker assertions.

## Migration Plan

1. Capture the actual starting test inventory and results in `coverage.md`; establish documentation ownership and checker fixtures.
2. Rename `entity` mechanically, align aliases and baseline existing violations. Run lint, tests and production build before changing runtime behavior.
3. Strengthen network accounting and provider lifecycle, migrate existing consumers, and prove the isolated expected-failure fixtures.
4. Transfer useful transport coverage, remove dead code, repair error presentation and extract cursor traversal.
5. Reproduce and repair paging races, then align CI, review validation and documentation. Record every test replacement and newly discovered repair.
6. Run final formatting, frontend lint/tests/build and the relevant hook tests, record their outcomes and check off the implementation tasks. This prepares the completed implementation checklist required to start full review. Review acceptance is a subsequent completion gate, not an unfinished checkbox task. After mandatory review accepts the result, record that outcome and mark the milestone complete in the same implementation change before a later commit.

There is no data migration or new deployment step. Rollback is a normal code revert of this change as a unit; old import aliases, deleted helper shims and the unsafe error fallback are not retained in the new implementation.
