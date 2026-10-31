## Context

See `proposal.md` for motivation and scope. The working tree was clean and there were no active changes at the start of planning.

Relevant current owners:

- `frontend/src/app/ui/App.tsx:19` combines provider configuration, session subscription and routing. Its subscription already has cleanup at line 30; preserve that behavior rather than treating it as missing.
- `frontend/src/widgets/main/MainScreenComponent.tsx:55` groups essential account loads with disk usage in one `Promise.all`. It owns workspace persistence at line 81, capability calculation at line 119 and screen composition at line 302.
- `frontend/src/entities/users/api/userApi.ts:14` owns session listeners alongside account operations; line 147 removes listeners with an unchecked array index. `frontend/src/shared/api/apiClient.ts:23` clears the current token and reloads on an authorized 401 without comparing sessions.
- `frontend/src/features/users/ui/oauth/GithubOAuthComponent.tsx:21` and `GoogleOAuthComponent.tsx:23` use constant state; `frontend/src/pages/OAuthCallbackPage.tsx:37` chooses the exchange provider from it.
- `frontend/src/features/users/ui/ProfileComponent.tsx:112` saves and then reloads fields; `frontend/src/features/settings/ui/SettingsComponent.tsx:85` replaces the draft after save. `frontend/src/features/users/ui/UsersComponent.tsx:105` accepts responses without query-generation checks.
- `frontend/src/features/settings/ui/AuditLogsComponent.tsx:26` and `frontend/src/features/users/ui/UserAuditLogsSidebarComponent.tsx:26` already share `useAuditEventPages` but duplicate event presentation.
- `frontend/src/shared/hooks/useVersionCheck.ts:3` imports an entity from shared; its cleanup stops the timer but does not invalidate an in-flight reload. `useScreenHeight.tsx:15` already handles visualViewport changes.
- `backend/internal/app/apptest/apptest.go:55` constructs the real application and opens an isolated test database at line 83. `backend/internal/app/server.go:98` creates its real handler; its SPA fallback uses `./ui/build`. `frontend/tooling/runBrowser.ts:16` currently permits only the fixture-based UI project.
- `mise-tasks/backend/test-group-packages:17` assigns remaining packages to `other`; `scripts/test.sh:66` runs a broad backend package selection. Both must exclude the dedicated browser owner. `.agent-hooks/review_gate.py:265` permits only the existing UI browser command in change-verification sections.

Root `AGENTS.md`, `frontend/AGENTS.md` and `backend/AGENTS.md` constrain this design. Follow `docs/architecture.md#frontend` for downward imports, public APIs, page ownership and app composition; follow its backend boundaries for test wiring. Presentation and localization follow `docs/frontend-design.md` and `docs/frontend-localization.md`. Tests follow `docs/testing.md` and the testing skill. No new architecture exemption or production dependency is needed.

## Goals / Non-Goals

**Goals:** separate independently changing owners, make session and async acceptance explicit, preserve existing workflows, and prove a small real-API browser lifecycle before expanding its coverage.

**Non-Goals:** a general application state framework, a universal form engine, a generic audit-table framework or a new backend serving mode. Scope exclusions are listed in the proposal.

## Decisions

### 1. Compose screens in app and give the shell slots

Create app-owned startup/navigation units and `widgets/app-shell` with content, workspace-selector and auxiliary-action slots. App owns the selected tab and workspace, loads essential account data, derives capabilities and composes the selected content. The shell owns header/sidebar presentation and drawer interaction, not account loading or screen imports. Keep existing tabs and routes. Migrate auth, auth-callback, profile, users and instance settings to their own `pages/*/index.ts` public APIs. Put sign-in, sign-up, emailed-code and reset panels inside the auth page because they have one screen owner.

Move the storage callback mechanically to `pages/storage-oauth`, preserving its URL and behavior until milestone 4. App may directly compose the existing domain features through their public APIs until their milestones; do not add empty page wrappers or new baseline exceptions to disguise that intermediate state. Instance settings composes verification-agent UI from its existing feature public API, fixing the present feature-to-feature dependency without rewriting verification.

Essential user/settings/workspace loads share an accepted startup revision bound to the session. Store errors as data and expose retry. Disk usage and GitHub release/star information are supplementary; their failure cannot gate essential content. Dispose or invalidate all branches on retry, session change and unmount. Derive capabilities from accepted values and default to denied while they are unavailable. Preserve global-admin override, workspace roles and workspace-creation settings. Current profile/settings acknowledgements update app's accepted user/settings through explicit callbacks guarded by that session. If a capability disappears, app stops presenting its protected screen.

Remembered workspace selection is accepted only if present in the loaded list. Clear the existing workspace preference when a session ends or is replaced; ordinary reload of an unchanged session can restore an accessible selection. Select only a workspace ID in app and derive the current object from the accepted list, so refreshed roles do not retain an obsolete object.

**Alternatives rejected:** retaining screen imports in the shell preserves mixed ownership; a global store adds another synchronization mechanism; new deep links change established navigation. Moving every domain page now would expand the agreed milestone.

### 2. Make session state infrastructure and domain APIs side-effect free with respect to login

Place token/user identity, a monotonic in-memory session generation and subscription management in `shared/auth`. Retain the existing persisted token/user keys and initialize from them without issuing requests. Expose one coherent session snapshot, activation, conditional invalidation and a subscription returning an idempotent disposer. Use a set of listeners and a React external-store subscription rather than separately maintaining an authorization boolean in app. Keep account API operations and generated aliases with their domain owner; they return responses rather than activating sessions inside `userApi`.

Auth flows own acceptance of a completed sign-in using a current intent/request revision and lifecycle. Only a still-current flow activates its returned session. Session end/replacement increments the generation, invalidates outstanding auth intent, and causes app to remove session-owned content. Remove the old listener array, split session setters, explicit `notifyAuthListeners` calls and their exports; update production/test consumers, keeping no aliases. A successful password change ends the current session and retains its bounded sign-in-required notice in the surviving application notification context.

Transport captures the session generation and token when attaching Authorization. On a 401 it conditionally invalidates only that same active generation/token. Signed-out sign-in refusals remain ordinary screen errors. End the session once even if several requests are refused. Remove transport's auth-triggered reload. Mounted screen work also compares its lifecycle/session identity before publishing data or feedback; transport does not invent domain acceptance policy. Version mismatch remains the separate allowed automatic reload in decision 7.

**Alternatives rejected:** comparing only the presence of an Authorization header can clear a newer session; using reload as the reset mechanism hides lifecycle defects; putting current user, settings and workspace state in shared would move domain ownership downward. A server revocation endpoint or token-storage migration is unnecessary for these guarantees.

### 3. Share a focused OAuth sign-in interaction between initiation and callback pages

Use `features/oauth-sign-in` for the two provider buttons, attempt metadata and callback exchange controller because auth and callback pages both consume that interaction. Use `crypto.getRandomValues` for at least 256 bits of state and `sessionStorage` for one active record containing state, provider, creation time and redirect URI. A new initiation replaces the previous record. Use a ten-minute TTL; reject malformed records, timestamps in the future, missing/incorrect state, a different redirect URI and legacy constant states before exchange. Read the provider from validated metadata, never from untrusted callback text. Store no code or secret in the record. If generation or storage fails, stay on the page with translated feedback.

Add this entry to the Product table in `docs/glossary.md` when implementing the interaction; keep the existing Pending sign-in definition:

| Term | In code and API | Meaning |
| --- | --- | --- |
| OAuth sign-in attempt | `OAuthSignInAttempt` | A single-use, time-limited browser record binding an external-provider callback to sign-in started in that tab. Unlike a server-side pending sign-in awaiting an emailed code, it binds OAuth state to the provider and redirect address and is consumed before code exchange. |

Consume the matching record synchronously before the first exchange, including a matching provider refusal. Mismatched callbacks do not exchange a code or silently choose a provider. A failure requires a new explicit authorization attempt; never replay an exchange automatically. After capturing validated callback inputs, replace the browser address to remove code/state/error parameters before ordinary navigation or diagnostic capture, keeping the existing callback path.

The callback controller is created without side effects and is retained by the active callback page. Its first explicit start consumes the attempt and retains the single exchange promise. Effect setup-cleanup-setup reattaches to that operation rather than consuming again or creating a second request. Acceptance belongs to the current attachment and sign-in intent; true unmount, replacement callback inputs or session replacement prevents the old completion from activating a session or emitting feedback. No global cache of completed codes is needed.

This is a browser callback-binding repair, not a claim to complete a wider OAuth/PKCE redesign. The guidance for a one-use browser-bound state is in [RFC 9700 section 2.1](https://www.rfc-editor.org/rfc/rfc9700.html#section-2.1). Existing exchange operations retain their payloads.

**Alternatives rejected:** constant state permits unrelated callbacks; localStorage outlives the initiating tab; provider secrets or codes in metadata increase exposure; a new server attempt API is unnecessary for this bounded repair; a simple effect ref that skips all repeated setup can lose the only completion under StrictMode.

### 4. Keep controlled drafts and acknowledge snapshots

Profile and instance-settings pages keep local controlled drafts and a last-acknowledged baseline. Derive dirty state instead of maintaining a separate `hasChanges` flag. Reuse `RequestRevision` for identity, not notifier-specific policy. Save captures the submitted snapshot, session identity and editing lifecycle; pending state prevents a duplicate save but does not disable ordinary editing.

On current success, accept the returned values as the baseline. For each field whose current value equals its submitted value, apply the server's returned normalized value; otherwise retain the current value. Publish accepted user/settings changes to app through the session-guarded callbacks from decision 1, even if the form has newer edits. Derive subsequent dirty state from the resulting draft and acknowledged baseline. Another save is available only if they differ. A newer profile edit matching the normalized response, or a settings toggle returned to its acknowledged value, leaves the form clean. Cover both equality cases alongside a newer draft that still differs. Remove the profile's unconditional post-save refetch that overwrites fields. Reset, where already offered, restores the acknowledged values. Failed operations preserve the draft and expose translated feedback. Initial load failure gets a recovery action. Password change remains a separate operation and never reuses a previous password as a draft baseline.

Use existing AntD/shared field presentation, labels, help and context-bound feedback; no second form store or generic async form controller. Persistent errors are rendered from keys/API errors; success notices retain the bounded language lifetime defined by frontend-localization. Retain existing validation and server policy, including mail prerequisites for enabling the second factor and sign-in after password replacement.

**Alternatives rejected:** replacing the whole draft loses newer input; disabling every field while saving avoids the race by changing interaction; storing translated errors freezes visible language; forcing all fields into a form-store abstraction duplicates current state.

### 5. Reuse list lifecycle and domain audit presentation

Move user management and its drawer into `pages/users`. Adapt the existing paged-list machinery to user search rather than keeping a second cursor lifecycle. Keep the 500 ms search debounce and page size 20; query changes reset the generation/cursor and unmount clears debounce, scroll subscription and callbacks. Preserve per-user pending operations and local row updates. Ensure a pending old page cannot undo an acknowledged role/status action for the current row. Use explicit request/response signals for stale success/error tests.

Add domain UI under `entities/audit-logs/ui` that receives events, a view descriptor and loading state. View descriptors explicitly select author/workspace visibility, short/full timestamp presentation, empty text and table/card mode. Fetch filters, authorization, page size 50 and scroll ownership remain with the instance/user views through the existing `useAuditEventPages`. Preserve captured author/workspace semantics and literal event text. Remove the duplicated renderers only from these two consumers. Leave workspace audit rendering unchanged until milestone 4.

**Alternatives rejected:** copying the list lifecycle duplicates already-tested race handling; moving audit filters into a generic UI obscures authorization; a shared-layer audit table would contain domain knowledge; migrating all three audit consumers now expands the milestone.

### 6. Let bounded layout determine content height

Use shell flex/grid tracks with `min-h-0` and `min-w-0`, semantic surfaces and a single scroll owner for each content region. Auth layout uses a viewport boundary with its own reachable scrolling. Preserve the visualViewport keyboard adjustment by giving the shell/auth boundary one resize subscription and a scoped CSS height property with a viewport-unit fallback. Dispose listeners and restore scoped values on unmount. Avoid hard-coded header-height subtraction in the migrated pages.

Existing unported domain components still require `contentHeight`. Measure their allocated content region at the app composition boundary and pass that value directly to those legacy consumers until their own milestones. Remove that numeric prop from migrated user pages. This is a temporary caller adaptation, not an alias for an old module or a second layout engine. Keep existing browser UI scenarios running across this boundary, including SMTP and WAL geometry.

Use accessible role/name queries in moved tests. Browser geometry tests cover long persistent errors, dirty/pending forms and reachable primary actions at 320 CSS pixels and actual 200% text enlargement. Verify sidebar selection/dismissal, restored focus, open-drawer retention and light/dark roles without replacing draft lifecycles.

**Alternatives rejected:** propagating viewport subtraction through new pages repeats the original coupling; deleting all old height props now would pull later domains into scope; replacing the keyboard workaround without browser evidence risks mobile usability.

### 7. Move version checking into app lifecycle without changing its cadence

Move `useVersionCheck` into `app/model` with its current five-minute interval, production-only activation and ten-second persisted reload cooldown. Invalidate an outstanding response on cleanup and avoid overlapping polls while one request is unresolved. Keep the last-reload preference outside session cleanup because it prevents loops across page loads. Network failures stay non-disruptive. Test cooldown and interval with a controlled clock and real HTTP boundary; test delayed mismatch after unmount and setup-cleanup-setup explicitly.

**Alternatives rejected:** leaving the hook in shared retains an upward import; coupling version reload to auth restores the reload loop risk; removing the interval or cooldown changes existing behavior.

### 8. Give the real-API browser lifecycle a Go test owner

Create the test-only package `backend/internal/app/browsertests`. It owns `apptest.New(t)`, API-created accounts, gateway, child browser runner and cleanup. Use the existing test database service and `testdb` isolation/retention policy; do not seed by SQL or call `apptest.New` from Node. Test-only support is split by gateway, child-run lifecycle and scenario owner; no production route or app constructor change is needed.

The public Node entry `pnpm --dir frontend test:browser --project=e2e` dispatches to the Go package with race checks. Go invokes a distinct internal runner for preparation and browser execution, never the public entry again. Reuse the pinned browser dependency image and process execution unit. Build into this run's temporary directory before starting browser scenarios. A test-only gateway binds `127.0.0.1:0`, proxies `/api/*` to the real apptest URL and serves that temporary `dist` with SPA fallback and a nonsensitive runtime-config script loaded before the bundle. It never touches `./ui/build`, changes process cwd globally or changes the frontend production origin logic.

On the supported Linux local/CI environment, the pinned browser container uses host networking to reach the loopback gateway. No fixed port or Docker socket inside the browser is required. API-origin, callback fallback, runtime-config ordering and readiness have focused gateway tests. The browser project has a separate directory and fixture from synthetic UI scenarios; it does not install `HttpFixture` for application APIs. External GitHub metadata may use explicit finite fixtures because it is not our API. OAuth-provider success/refusal security cases stay at the screen boundary, without live third-party accounts.

The outer supervisor waits for the complete owned process group after the driver exits and retains bounded forced shutdown. Container removal must be followed by a successful absence check; an already absent container is acceptable, while unresolved removal fails the run.

The Go owner propagates context cancellation to the inner runner, closes the gateway and stops apptest before returning. The outer entry forwards interruption, bounds shutdown and uses the run identity to clean up any owned container/directory left by a failed child. Resource ownership must be registered before waiting for readiness. Negative probes use the actual public command and real owner: scenario failure after readiness, readiness failure and interruption all require a nonzero exit, their intended diagnostic and absence of owned processes/ports/containers/temp directories. Keep these isolated probe supervisors in frontend tooling tests; their child commands run the Go application owner and selected probe scenario, never the supervisor tests again. Use a private test-only probe selector rather than expanding accepted public CLI flags. Keep unrelated service probes alive to prove isolation. Follow `testdb`'s retained failing-database policy rather than pruning it.

The real-API ownership probes use `vitest.browser-e2e.config.ts` and run in the separately provisioned E2E CI job. The default Vitest selection excludes that file so screen and tooling checks keep their backend-free lifecycle.

Disable E2E trace/HAR/video and automatic authentication screenshots, since these can record passwords or Authorization headers. Retain sanitized phase/status logs and explicitly safe shell screenshots only. Synthetic UI trace/reference behavior stays unchanged. No retry masks lifecycle failures.

**Alternatives rejected:** a Node-created imitation of the backend bypasses real wiring; starting the shared dev app loses isolation; serving a global build directory causes concurrent-run races; giving Node a fake `testing.TB` cannot satisfy the Go lifecycle; putting the browser in ordinary backend selection introduces an unrelated runtime dependency.

### 9. Keep browser E2E selection and verification explicit

Add `browser-e2e` to backend package classification and exclude that subtree from `other` and the default broad selection in `scripts/test.sh`. Capture package discovery in a checked assignment before filtering, so partial output from a failed discovery cannot start an incomplete suite. Keep classification exhaustive and non-overlapping; do not use a silent environment skip or helper build tag. The dedicated CI check `test-frontend-browser-e2e` uses the public frontend entry, pinned Go/Node/pnpm, frozen dependencies and its own database setup, independent of backend lint/group outcomes. Add it to release-version selection prerequisites alongside current gates and verify failed prerequisite semantics locally through CI tests.

Extend the review gate's exact-command allowlist, its help/docstring and `openspec/config.yaml` to recognize exactly `pnpm --dir frontend test:browser --project=e2e`. Reject missing/unknown projects, extra flags and shell syntax. Keep UI verification and explicit reference updates separate. Hook tests exercise parser acceptance/rejection without running browser commands.

The planned change-verification section uses the already allowed backend package command to run the real browser owner, plus the frontend suite and UI suite. Tasks also explicitly run the public E2E entry. This keeps the proposal valid under the current parser before its implementation extends the whitelist; package execution and the public command must exercise the same owner, not different test sets.

**Alternatives rejected:** adding the browser owner to `other` makes ordinary API tests require frontend tooling; skipping it under an unset flag hides missing coverage; relying on a hosted CI run cannot provide local task completion; widening the parser to arbitrary browser arguments weakens verification validation.

## Risks / Trade-offs

- [Moving screens and behavior together can hide regression] -> Move existing scenarios with their owners, record them in `coverage.md`, and reproduce each bug with a failing focused test before changing behavior. Keep mechanical moves distinguishable in the diff.
- [StrictMode repeats callback setup after consuming state] -> Retain one controller/promise and test actual root replay, successful completion, failure, route replacement and true unmount with notifications/session observers retained.
- [Session cleanup can erase unrelated preferences] -> Clear only session-owned state and workspace selection; preserve language/theme and version cooldown, and test the next sign-in with a different account.
- [New E2E owners can leak child resources on failure] -> Establish the pilot and real failure probes first, with bounded cancellation and owned-resource assertions. Do not assume a child exit proves cleanup.
- [Linux host networking is less portable] -> Document the supported host/CI environment, as existing project tooling does; do not add a second network orchestration layer without a demonstrated need.
- [Audit variants can accidentally converge] -> Keep view descriptors and tests explicit about time, columns, page size and empty text; retain workspace's existing consumer for its own milestone.
- [Presentation migration can expand into later domains] -> Limit semantic styles to moved/account/shell/audit units, remove their obsolete baseline entries, and retain later-domain entries without renaming them into new exceptions.

## Migration Plan

1. Implement the isolated browser owner, public dispatcher and failure probes first, proving sign-in/shell/logout before the larger screen move.
2. Introduce shared session infrastructure, request-bound 401 handling and guarded auth completion; update all callers/tests and delete superseded exports.
3. Move composition and account screens in small dependency-ordered steps, preserving existing scenario ownership. Add the draft/query/callback corrections and browser coverage beside those moves.
4. Shrink repaired FSD/style entries, synchronize affected docs and command checks, and complete local verification and required review. This is the pre-commit boundary; there is no commit or release task.

No database migration or HTTP contract change is required. Existing persisted tokens remain usable. In-flight legacy OAuth attempts must restart after the update. Rolling back the application code restores its previous frontend behavior without a data conversion; no compatibility shim is retained.
