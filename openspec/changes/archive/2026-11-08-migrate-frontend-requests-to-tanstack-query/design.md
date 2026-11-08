## Context

See `proposal.md` for motivation and scope. This design is constrained by root `AGENTS.md`, `frontend/AGENTS.md`, and the frontend ownership rules in `docs/architecture.md:332`. Screens retain product policy; shared code owns reusable request mechanics and transport. Dependencies still point downward and screen tests do not import application composition.

Current integration points:

- `frontend/src/shared/hooks/useResourceCollection.ts:13` already shares collection loading and local reconciliation between storages and notifiers; its `reload` still maintains request revisions and session checks.
- `frontend/src/pages/databases/ui/DatabasesComponent.tsx:69` reads databases, backup configurations and storages together, refreshes every 30 seconds, and handles workspace refusal. `frontend/src/pages/databases/ui/backup-config/useBackupConfigLoad.ts:12` separately owns configuration loading and acknowledgement.
- `frontend/src/pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.tsx:62` loads configuration before attempts, traverses every page, and starts one-minute polling only after a successful current-day load. A refresh failure keeps shown attempts and later polling continues.
- `frontend/src/pages/storages/lib/useStorageUsage.ts:18` polls every second, skips hidden-tab refreshes and stops after a failure. `frontend/src/pages/databases/ui/backup-config/useBackupCleanup.ts:18` schedules its next read two seconds after the preceding result and stops on zero pending files or an error.
- `frontend/src/shared/hooks/usePolledPagedList.ts:16` delegates list state to the existing pagination machinery and supplies visibility-aware first-page refreshes. Its loaded-tail behavior is specified in `openspec/specs/frontend-list-loading/spec.md`.
- `frontend/src/shared/api/apiClient.ts:12` already preserves intentional abort errors and associates a 401 with the session that authorized its request. Domain facades and `frontend/src/shared/lib/collectCursorPages.ts:3` do not currently forward cancellation.
- `frontend/src/shared/auth/session.ts:51` publishes account transitions with a new generation. `frontend/src/app/ui/App.tsx:30` recreates account screens on that generation; feedback and locale/theme providers remain outside them.
- `frontend/src/features/edit-storage/model/useStorageDraft.ts:38` and `frontend/src/features/edit-notifier/model/useNotifierDraft.ts:38` own editor acknowledgement. `frontend/src/shared/auth/useAuthOperation.ts:11` owns authentication intent. Their acceptance rules are not generic request-state rules.
- `frontend/src/shared/lib/test/ScreenConfiguration.tsx:7` is the common real-provider composition for standalone screen tests.

The earlier review file `temp/refactor/full-review.md` is absent. Current source has 22 calls to `useRequestRevision`, eight `let isActive = true` closures, and no `AbortController` construction or fetch `signal` option outside tests and test infrastructure. These are search baselines, not predicted deletions: several revisions protect domain state and must remain.

## Goals / Non-Goals

**Goals:** Remove repeated ordinary request-state management throughout the frontend; keep server refreshes authoritative; make cancellation reach HTTP; preserve observable screen and editor behavior; and measure the actual source and bundle changes.

**Non-Goals:** Reimplementing pagination, replacing domain state machines, creating a generic workflow framework, changing HTTP contracts, or meeting an invented line-count target. The exclusions in `proposal.md` remain binding.

## Decisions

### 1. Use TanStack Query v5 for ordinary reads and actions

Use `useQuery` for reusable ordinary reads and `useMutation` for explicit server actions, including tests and commands that use POST even when they do not persist settings. Prefer the library primitives directly. Add a focused shared helper only where consumers demonstrably repeat the same lifecycle responsibility; do not rebuild the library behind a universal wrapper.

App composes the production provider and owns the client lifetime and session-transition subscription. Shared code provides the client factory and domain-independent defaults. Query definitions stay with their existing domain API owner or closest page/feature consumer. Shared request infrastructure contains no storage, database, permission or editor policy. This preserves `docs/architecture.md` ownership rather than moving all endpoints into a central registry.

**Alternatives rejected:** Extending handwritten loading/revision hooks leaves a second request framework to maintain. Retaining a small pilot leaves eligible screens in several styles, contrary to the owner's decision. Wrapping every query and mutation hides useful library behavior and recreates the duplication at another layer.

### 2. Keep active request state without retaining responses for later visits

Apply these defaults through the shared factory:

| Setting                          | Value      | Reason                                                                                             |
| -------------------------------- | ---------- | -------------------------------------------------------------------------------------------------- |
| Query `staleTime`                | `0`        | An accepted response never provides a freshness exemption.                                         |
| Query `gcTime`                   | `0`        | Unused query results have no retention period.                                                     |
| Query `refetchOnMount`           | `'always'` | Opening a new consumer requires a current server read.                                             |
| Query `retry`                    | `false`    | A failed request is not silently repeated.                                                         |
| Query `refetchOnWindowFocus`     | `false`    | Existing screen visibility rules control refreshes.                                                |
| Query `refetchOnReconnect`       | `false`    | Connectivity changes introduce no new refresh behavior.                                            |
| Query and mutation `networkMode` | `'always'` | Explicit operations attempt HTTP and fail normally instead of waiting for a browser online signal. |
| Mutation `retry`                 | `false`    | An uncertain write is never replayed automatically.                                                |
| Mutation `gcTime`                | `0`        | Unobserved completed action state has no retention period.                                         |

Do not add persistence, hydration, prefetching, `keepPreviousData`, an offline mutation queue or optimistic writes. Current consumers can share the same active result and in-flight request. A new screen visit must still start or join a fresh read. Its initial presentation must wait for that visit's accepted read rather than treating a still-active sibling's older data as a completed load; use the observer's fetched-after-mount state where needed. A rapid unmount/remount must obey this rule even before a zero-delay garbage-collection timer runs.

While a screen remains open, keep accepted data during a background refresh where its existing policy requires that. This is necessary for stable lists and editing; it does not authorize reuse across contexts. Do not persist mutation variables or capability results, put credentials in query keys, or retain sensitive action results after their consumer closes.

**Alternatives rejected:** `staleTime: 0` alone retains inactive responses. Positive freshness or retention intervals conflict with the agreed behavior. Clearing displayed data on every polling tick creates flicker and loses the required refresh-failure behavior. A literal absence of any in-memory result would also remove the state needed to render an open screen; the owner accepted active-only storage during exploration.

### 3. Session generations and complete request parameters determine identity

Session-owned keys include the session generation, domain operation, workspace, resource and all server-side filters that affect the answer. For example, a workspace storage listing identifies `['session', generation, 'storages', workspaceId]`. Distinguish full configuration reads from cleanup-summary polling: they use the same endpoint but have different lifetimes and acceptance rules. Time-relative healthcheck ranges use the selected period as identity and compute the actual `since` value at each read, not on every render. No token, password, connection string or one-use capability appears in keys.

Subscribe to session transitions at application composition. As soon as a session ends or is replaced, stop old scheduled work, cancel ordinary reads and clear both query and mutation stores with `queryClient.clear()` before the next account can use them. Retain captured-session checks for every mutation completion effect, including invalidation, acknowledgement, feedback and navigation. Generation keys provide isolation even before React finishes cleanup. Clearing the client does not cancel server-side writes, and mutation callbacks can still run after its store has been cleared.

Keep the existing HTTP 401 ownership checks and authentication-intent rules. Theme, locale, feedback lifetimes and the version-reload cooldown remain outside account resets. A session transition is separate from unmounting the application provider; the latter also disposes its subscription and client.

**Alternatives rejected:** Keys containing only a workspace/resource can collide across sessions. User IDs do not distinguish two sign-ins by the same account. Relying only on provider unmount or client clearing leaves late mutation callbacks able to affect a newer session. Moving authentication acceptance into query infrastructure loses the existing attempt ownership.

### 4. Pass cancellation through ordinary reads, retaining domain acceptance guards

Pass the query function's `AbortSignal` into each migrated domain read, every corresponding `openapi-fetch` call, and complete-list traversal. Check cancellation before the next page and after each awaited page, so an obsolete complete listing cannot continue with credentials from another session. Replacing context, explicit supersession and loss of the final consumer cancel the relevant ordinary read. Cancelling one observer must not abort work still needed by another current observer.

When extending a facade, use named parameter objects where project parameter rules require them and update all callers in the same change. Do not modify the OpenAPI contracts or generated files. Preserve `ApiError` normalization and intentional-cancellation handling in the existing transport.

Retain revisions that distinguish domain events: draft acknowledgement, saved-settings test approval, authentication intent and one-use token subjects. They do not duplicate query state. Existing paginated and WAL controllers retain their request-acceptance behavior and are not routed through query cancellation just to make cancellation uniform.

**Alternatives rejected:** Dropping late responses without aborting HTTP wastes work and permits further cursor requests. Aborting all requests on every consumer unmount breaks active siblings. Treating abort as a server failure creates misleading notices. Removing all request revisions confuses request lifetime with whether a response is still valid for a draft.

### 5. Keep refresh policy local and preserve exact scheduling semantics

Represent each ordinary polling source with its current interval, prerequisites, visibility rule, in-flight handling and failure policy. Fixed schedules use `useFixedQueryPolling`, which keeps ticks independent of request completion and skips a pending read. The pinned QueryObserver resets its native interval after a fetch settles, so `refetchInterval` cannot preserve these schedules. Screens choose eligibility and visibility; session or query-scope replacement disposes the old timer. Backup cleanup retains its separate completion-relative scheduler. A scheduled tick never cancels and restarts an in-flight refresh.

| Source               | Preserved policy                                                                                                                                                                                                |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Databases            | Three complete listings accepted together; 30-second visible-tab refresh; immediate visible-tab refresh only after readiness; refresh failure keeps shown data silently.                                        |
| Notifiers            | Five-minute polling with its existing visibility and error behavior; do not silently adopt hidden-tab suppression from library defaults.                                                                        |
| Storages             | Initial read and action-triggered refresh; no new automatic list interval.                                                                                                                                      |
| Healthcheck attempts | Config prerequisite; every cursor traversed; fixed current-day refresh every minute after first success; slow responses do not move later ticks; historical periods do not poll; initial error requires retry; background errors retain attempts and polling. |
| Storage usage        | Fixed one-second visible-tab refresh; explicit retry restarts the schedule; no competing request; any failure stops polling until explicit retry; refusal clears protected values under existing policy.                                                    |
| Agent availability   | Fixed five-second refresh, including background tabs; skip pending reads and retain scheduled recovery after errors. |
| Backup cleanup       | Next read two seconds after the previous accepted result; publish cleanup counts only; stop on zero or error; retry reads immediately without replacing a configuration draft.                                  |
| Paged lists          | Existing `usePolledPagedList` scheduling and `PagedListController` merge/metadata rules remain authoritative.                                                                                                   |
| WAL                  | Existing refresh/live controllers remain authoritative.                                                                                                                                                         |
| Application version  | Existing production-only fixed five-minute interval, in-flight suppression and ten-second reload cooldown; late responses cannot reload a disposed lifecycle.                                                         |

Do not install global query-error toasts. Screens continue distinguishing blocking loads, ordinary action failures and silent refresh failures. An initial load failure must disable further scheduled attempts until explicit retry where required. Use separate result/error handling for an acknowledged write and its later refresh. Retaining accepted data after a failed refresh applies only when the failure does not establish access loss. A current workspace-access refusal clears protected list/detail content before reconciling accessible workspaces, including after a successful mutation; the mutation remains acknowledged and is never replayed. This preserves the access-refusal requirements of `frontend-workspace-screens` and `frontend-database-screens`.

**Alternatives rejected:** One shared polling interval or global error handler changes product behavior. Library focus/reconnect defaults add requests. A plain fixed interval cannot reproduce cleanup's delay after completion. Replacing pagination with `useInfiniteQuery` would refetch all loaded pages and lose the existing first-page/tail policy.

### 6. Reconcile successful writes without giving reads ownership of drafts

For an accepted mutation, first verify its captured session, resource and editor context. Cancel superseded reads before publishing returned server values, then update affected active query data and invalidate/refetch the necessary active scopes. Keep this in-flight transition ordered so an earlier response cannot overwrite the write. Delete or transfer removes the affected item/detail rather than refetching a removed resource. Refresh only affected scopes; unrelated tabs must not gain background requests.

Use the existing `PagedListController` reconciliation/refresh methods for actions affecting retained paged lists. Their post-action read failures remain refresh failures. Keep the editor's submitted snapshot and acknowledged baseline independent of query data. Refreshes never reset newer input or connection-test approval. After a screen unmounts within the same session, any still-required parent collection update uses that collection's current-context acceptance rules; screen-only feedback and close/navigation effects require a live interaction. A session change forbids both kinds of obsolete completion.

One-use capability issuance remains a mutation owned by its current subject, never a shared read or prefetch. Authentication forms may simplify ordinary pending/error state while retaining `useAuthOperation` and OAuth ownership. Do not replay capability issuance on focus, reconnect or a query retry.

**Alternatives rejected:** Invalidating everything adds unnecessary requests. Updating returned values without cancelling preceding reads leaves a save/read race. Resetting form values from query data loses user input. Relying only on per-call mutation observer callbacks can drop a necessary parent acknowledgement when a detail unmounts; putting all effects in unconditional callbacks affects disposed screens.

### 7. Complete the migration by request responsibility, with explicit exceptions

Maintain a request inventory during apply, covering every production API consumer in `app`, `pages`, `features` and widgets, not only `useRequestRevision` matches. Record each group as migrated or retained with a domain reason in this change's design. The planned coverage is:

| Consumer group                                                                                                           | Migration boundary                                                                                                                                                          |
| ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Application startup, workspace creation/selection reconciliation, disk, release and version reads                        | Ordinary queries/actions migrate; startup accepts essential data together and supplementary failures remain nonblocking.                                                    |
| Storages and notifiers, details, transfer adapters and editors                                                           | Reads/actions migrate; list selection, draft state and approval rules remain local.                                                                                         |
| Database list/detail, creation wizard, connection tests, limited-user operations, storage/notifier pickers and transfers | Ordinary reads/actions migrate; wizard progress, submitted snapshots and test approval remain local.                                                                        |
| Backup configurations, healthcheck and verification settings, usage and cleanup                                          | Reads/actions migrate with the policy table above; drafts remain independent.                                                                                               |
| Backup, restore, verification, audit and agent lists                                                                     | Pagination/controllers stay; ordinary detail reads and explicit actions around them migrate.                                                                                |
| Profile, instance settings/mail server, users, workspace settings/members/search                                         | Ordinary reads/actions migrate; dirty-state and debounced-search rules remain local.                                                                                        |
| Authentication and OAuth consumers                                                                                       | Ordinary configuration reads/submission state can migrate; intent, exchange uniqueness, popup messages, deadlines and session acceptance remain under their current owners. |
| WAL and one-use restore/download/install credentials                                                                     | WAL controllers and capability-subject ownership stay; surrounding ordinary actions migrate without sharing or retaining capabilities.                                      |

Remove `useResourceCollection` only after its collection consumers have migrated and any remaining reconciliation responsibility has a clear owner. Remove `useRequestRevision`, `RequestRevision` or `useSessionGuard` only if no genuine consumers remain; their survival is not a migration failure. Keep useful domain hooks as query compositions rather than deleting them merely to reduce the hook count.

**Alternatives rejected:** Migrating only the originally named screens is a pilot. Declaring every `useEffect` an eligible query includes clipboard work, feedback timers and state machines. Replacing every domain hook with inline query calls makes ownership harder to follow.

### 8. Verify behavior through real screen providers and measure the actual delta

The shared screen renderer creates an isolated client per render with production defaults and clears it on teardown after unmounting. Standalone screen tests consume shared configuration; full-App tests cover the production session bridge. MSW remains the HTTP boundary. No test mocks TanStack Query or domain APIs, shares a module-global client, or disables retries/timers differently from production to conceal behavior changes.

Use controlled responses for stale success/error and save/read races; observe abort at HTTP and prove traversal stops. Use fake timers only for scheduling, garbage collection and actual time-dependent behavior. Keep existing assertions for drafts, access refusal, loaded tails, metadata, visibility and explicit retries. Add HTTP-boundary screen tests using production query defaults that separately trigger focus and reconnect on a ready screen without either refresh rule and prove that neither event sends another ordinary read. Dispatch `visibilitychange` on window, observe the installed focus manager process hidden and visible states, and verify the focus case fails when focus refresh is temporarily enabled. Update request-count assertions only for justified shared in-flight work, cancellation or removed duplicate reads, documenting the reason. Exact interval and no-replay assertions remain meaningful.

Before implementation, record production TS/TSX line count excluding tests, test infrastructure and generated schemas, plus the summed gzip byte size of production JS assets from a build. Repeat with the same method after migration, including new infrastructure in the line count. Use the actual working-tree baseline rather than `HEAD`, since unrelated work may already be present. Record net changes and any unrelated concurrent changes that prevent attribution. The old 1,000-1,400-line and 15-KB estimates are not acceptance thresholds.

**Alternatives rejected:** Hook-only mocks do not prove screen/session behavior. Changing all tests to use different library defaults can hide production problems. Comparing deleted lines alone ignores the cost of new infrastructure; comparing against an unrelated commit attributes other work to this migration.

## Risks / Trade-offs

- Active results are still stored in memory. Mitigation: document this explicitly, use zero inactive retention, require fresh visit reads, and test immediate remounts and active sibling consumers.
- Query/mutation defaults differ from handwritten fetch behavior. Mitigation: centralize defaults, preserve each source's scheduling policy, and test connectivity, visibility, first failures and delayed refreshes.
- Clearing a client cannot stop an already submitted mutation or its callback. Mitigation: keep captured-session/context checks and distinguish domain acknowledgement from screen effects.
- Shared configuration reads can accidentally overwrite an editing baseline or let cleanup polling replace settings. Mitigation: separate full-config and summary query ownership and publish only the intended fields.
- A broad migration increases regression surface and adds bundle weight. Mitigation: implement in dependent slices within one change, keep the existing suites, run the full required verification before review, and report measured costs without inventing a savings threshold.

## Migration Plan

1. Capture the request inventory and comparable source/build baseline. Add the dependency, shared defaults, isolated test providers and production session cleanup.
2. Extend cancellation through migrated domain read facades and cursor traversal without changing the HTTP contract.
3. Migrate resource collections/details first, then database flows, polling/configuration, account/workspace/admin consumers and remaining application/authentication request state. These are implementation slices of one complete change, not independently shipped pilots.
4. Finish the request inventory, delete unused duplication and update `docs/architecture.md` and `docs/testing.md`. Do not edit agent instructions for an ordinary implementation change.
5. Run targeted tests, `mise run test`, frontend lint, production build and the existing browser UI suite. Record line/bundle measurements and resolve mandatory review findings before the working tree is ready to commit.

The change introduces no storage migration or server contract change. If rollback is explicitly chosen, revert this change's request migration and its dependency/lockfile edits together, preserving unrelated work. No dual implementation or feature flag remains in production.

## References

- [TanStack Query defaults](https://tanstack.com/query/latest/docs/framework/react/guides/important-defaults)
- [Query cancellation](https://tanstack.com/query/latest/docs/framework/react/guides/query-cancellation)
- [Network modes](https://tanstack.com/query/latest/docs/framework/react/guides/network-mode)
- [Infinite-query refetch behavior](https://tanstack.com/query/latest/docs/framework/react/guides/infinite-queries)

## Implementation baseline

Before implementation, only this change was untracked. The production build and typecheck passed. Production source contained 646 files and 55,305 lines; production JS totaled 714,580 gzip bytes. Count `.ts`/`.tsx` files under `frontend/src`, excluding names containing `.test.`, `.fixture.` or `.gen.` and paths under `shared/lib/test/`, with `splitlines()`. Sum `len(gzip.compress(asset.read_bytes(), mtime=0))` for `frontend/dist/assets/*.js` after each production build.

## Implemented ownership inventory

The audit covers domain API imports and calls, multiline calls, callback adapters, direct external `fetch` calls, request revisions and asynchronous effect closures throughout `app`, `pages`, `features`, `widgets` and shared UI. Facades retain transport responsibilities; their callers own query identity and screen policy.

| Consumers | Implemented request owner | Retained responsibility |
| --- | --- | --- |
| `useApplicationStartup`, `CreateWorkspaceDialogComponent` | Session queries for coordinated startup, independent disk usage and workspace-access reconciliation; workspace creation mutation | Shell acknowledgement, selected workspace and access-reconciliation revision; essential results are accepted together |
| `useVersionCheck`, `useIsNewGitHubVersionAvailable`, `StarButtonComponent` | Session queries with cancellable HTTP; version interval remains five minutes | Reload cooldown; release/star presentation; no persisted release-response cache |
| `StoragesComponent`, `NotifiersComponent`, their detail and transfer components, `useStorageUsage` | `useResourceCollection`, session queries and session mutations | Selection/search, refusal handling, notifier five-minute schedule, storage-usage visibility and stop-on-error policy |
| `useStorageDraft`, `useNotifierDraft` | Explicit create/update/test mutations | Submitted snapshots, acknowledged baseline, field merging, approval revisions and current editor context; saving flags spanning domain acknowledgement remain local |
| `DatabasesComponent`, `DatabaseComponent`, `DatabaseConfigComponent` | Aggregate collection/detail queries; copy/delete/rename/test mutations | Thirty-second visible polling, sorting, coordinated acceptance, wizard-close and parent acknowledgement |
| `BackupStorageSelection`, `DatabaseNotifiersFields`, `DatabaseTransferDialogComponent`, shared `ResourceTransferDialog` | Complete-list queries and transfer mutations | Nested editor overlays, choice/target ownership, focus return and transfer progress |
| `CreateDatabaseComponent`, `startFirstBackup`, `CreateLimitedUserComponent`, `useDatabaseConnectionTest`, connection/notifier editors | Explicit mutations, including first-backup dispatch | Wizard recovery, connection approval, submitted snapshots; `DatabaseConnectionForm` keeps its submission guard around a caller-owned asynchronous workflow |
| Backup configuration readers/editors and `useBackupCleanup` | Separate full-config, selected-storage and cleanup queries; save mutations | Draft merging and completion-relative cleanup scheduling; cleanup never replaces settings |
| Healthcheck configuration/editors and attempts | Config prerequisite query, complete cursor traversal query and save mutation | Current-day period identity, one-minute polling after success, historical behavior and silent later failures |
| Verification configuration/detail/agent-count screens | Config/detail/count queries and explicit mutations | Five-second availability polling; verification list remains under `usePolledPagedList` |
| Logical/physical backup and restore screens | Explicit action/capability mutations | `usePagedList`/`usePolledPagedList` and `PagedListController` still own pages, loaded tails, metadata, refresh errors and reconciliation |
| Physical restore commands and logical download | Capability issuance mutation; URL construction remains pure | Subject revisions, one-use issuance and current-interaction acceptance; capabilities are never query results |
| WAL monitoring | Explicit start action mutation | `walStreamRefreshController` and live controller keep sequencing, visibility, period and last-success behavior; ordinary query polling cannot substitute for these policies |
| Profile, instance settings and mail server | Initial queries and explicit mutations | Dirty input, accepted baseline, permissions, password-change feedback and session transition |
| Workspace settings/members, user management, verification agents | Queries for settings, deletion check, members and debounced search; explicit actions | Settings/confirmation context, pending target identity, membership acknowledgement; user and agent cursor lists remain paged controllers |
| Workspace/instance/user audit views | Existing paged-list API adapters | Existing cursor traversal, filtering, page ownership and list error policy |
| Authentication forms and `AuthPage` | Registration-options query and sign-in/signup/reset/code mutations | `useAuthOperation` still owns intent, challenge and session acceptance |
| `OAuthCallback`, `OAuthSignInAttempt`, `useGoogleDriveAuthorization`, `StorageOAuthPage` | Existing single-exchange state machines | Popup origin/message validation, deadlines, callback uniqueness and authorization-attempt ownership; their exchanges are intentionally not replayable queries |

`useRequestRevision` retains ten revision instances in eight domain consumers: resource-detail test/context ownership, transfer interaction ownership, workspace reconciliation/settings, user search pagination and physical restore-token subjects. It no longer owns ordinary read loading. `useSessionGuard` remains necessary for post-mutation effects and retained controllers.

### Infrastructure details

Coordinated startup and database collection reads publish their first failure immediately for recovery, but keep the aggregate query pending until every branch settles. Waiting for all branches in `finally` preserves the library's cancellation ownership after the first rejection. Retry, supersession and unmount therefore still abort unfinished siblings and cursor traversal. A current sibling's access refusal remains authoritative; a synchronous guard permits only one workspace reconciliation per aggregate load, even when several siblings refuse access concurrently. An aborted older sibling cannot reconcile or clear a recovered workspace. Controlled-response tests cover initial failure, successful retry, obsolete 403/404, unmount and a current late refusal. The complete-list cancellation case accepts a nonempty first page, holds a second page with continuation and proves that partial data remains hidden, that the pending second page is aborted and that no third page is sent.

Startup applies every successful data revision, including results structurally shared with the previous payload. Access refusal can remove the selected workspace from local state before reconciliation fails; an unchanged successful startup retry must restore that workspace once access returns. The full-App regression covers this sequence with a single workspace and an identical retry payload.

Workspace Settings also applies every successful data revision after clearing its accepted workspace and form on a child access refusal or mutation 404. Keeping the screen mounted and returning the original workspace on Retry restores the form even when the query retains the same object reference. Three HTTP-boundary cases cover member reads returning 403/404 and a rename returning 404, followed by an unchanged successful retry payload and no false mutation acknowledgement.

Storage and notifier details hide their accepted content as soon as a current read returns 403/404, before requesting parent reconciliation. That refusal remains latched through a pending or failed parent refresh and through pending or failed detail retries. A successful detail data revision restores the content, including when the payload is structurally shared with its earlier value. HTTP-boundary cases cover both resource types and both statuses while the parent response is held, followed by transient parent/detail failures and successful recovery.

Backup usage also keeps its authorization clearing latched until a successful current data revision. A transient failed retry cannot reveal the total retained internally from before access was refused. Four screen cases cover 403/404 followed by HTTP 500 or a network error, pending retry, stopped polling and successful recovery with the original total; the last step also verifies that polling resumes.

Transfer notifier-usage counts, target resources and backup-editor config/history use `readConcurrentRequests`. These groups have no later refusal-reconciliation policy, so the first required failure immediately aborts unfinished siblings through a signal composed with the query signal and preserves the original failure for the screen. Optional history failure remains nonblocking. The audit covers every production `Promise.all`; the startup and database groups retain the pending aggregate policy above, and all remaining ordinary groups use sibling cancellation. HTTP-boundary regressions cover failed target reads followed by close, retry or session replacement, failed notifier usage with unfinished cursor traversal, and failed config reads with pending history for both backup types.

`useSessionQuery` adds generation identity, rejects obsolete completions and hides pre-visit results until `isFetchedAfterMount` and a data update in that visit. An error also marks a query fetched, so the captured initial data-update count separately prevents an initial failure from revealing a sibling’s earlier data; timestamps cannot distinguish updates under a frozen clock. Explicit reload first cancels the exact request, including an initial request with no accepted data; native refetch alone can join that initial request. Scheduled refreshes pass `cancelRefetch: false` to join current work. The cleanup query identifies the accepted full-config revision rather than just cleanup values. A save can return counts equal to an earlier configuration after polling has changed them; its acknowledgement must still replace the previous summary and cancel the old poll. Two screen regressions reproduce this for logical and physical backups and pass with revision-scoped cleanup. Custom timers stop when their captured session is inactive. Storage usage and application-version consumers explicitly follow a replacement session with a new generation; the application remounts protected account screens on that generation.

`useSessionMutation` captures only the generation at invocation and checks it before dispatch, so a queued microtask cannot submit with replacement credentials. Domain owners still guard completion. Query notifications publish synchronously so pending UI disables an action before the next interaction; existing double-submission screen tests exercise this behavior. Both production and test clients use the same factory.

Workspace deletion owns a distinct confirmation attempt across its prerequisite database listing and DELETE. The confirmation stays busy throughout both requests, and synchronous reentry protection prevents duplicate writes. Cancel, Escape or close during the prerequisite invalidates the attempt and cancels its HTTP read; an obsolete completion cannot delete the workspace or close a newer confirmation. Dismissal is disabled after DELETE starts. Screen regressions hold each HTTP response and verify one listing, one DELETE and one completion for a confirmed workflow, plus cancellation without DELETE for all three dismissal paths.

Workspace membership accepts the returned member before refreshing the collection for both ADDED and INVITED outcomes. A transient follow-up failure preserves that acknowledged row and the existing members. HTTP-boundary regressions cover both outcomes.

The fixed polling helper serves healthcheck attempts, storage usage, application version and verification-agent availability. Controlled-response regressions verify the next original tick after a slow response: usage at two seconds after a response at 1.5 seconds, version at ten minutes after a response between ticks, and availability at ten seconds after a response at seven seconds. Healthcheck cases verify the next minute tick after a 20-second response and skip an intervening pending tick during a 70-second response. Initial-success prerequisites, explicit-retry schedule resets, hidden-tab behavior, stop-on-error and continued recovery remain local screen policies.

`ScreenTestProviders` supplies the common real provider composition to screen rendering and the connection-test hook suite. Each mounted composition creates its own client, unsubscribes and clears it on disposal. The MSW lifecycle treats abort as request settlement because aborted requests do not necessarily emit `response:mocked`; cancellation assertions use `finishCancelledApiResponse`, while current writes/reads keep `finishApiResponse`. Strict Mode can abort a facade call before HTTP begins, so relevant tests assert the aborted facade signal and fresh accepted HTTP rather than requiring two HTTP hits. Mutation dispatch is asynchronous; fake-timer scenarios flush dispatch before changing context. Tests now require protected rows to disappear when a session ends. The workspace search/language test counts non-polling requests: a legitimate one-second usage poll can occur during user input under machine load, so including it in an unchanged total made that assertion unstable. The assertions against search-triggered requests and lost drafts remain.

A full-App polling scenario advances to a scheduled database refresh, holds its HTTP response and replaces the session. It verifies abort, releases an obsolete collection, advances through two more old intervals and observes no additional database request or stale selection. The same scenario keeps French localization and dark theme active and verifies that a mismatching production version cannot bypass the persisted reload cooldown after session replacement.

Version checking guards browser storage access. A cooldown read or write failure leaves the interface mounted and usable and skips automatic reload; checking continues on its existing interval. Screen regressions exercise SecurityError on cooldown reads and QuotaExceededError on timestamp writes, then restore storage and verify that a later version check can reload normally.

## Final measurements

The same production-source count after formatting contains 654 files and 55,423 lines: **118 more lines (0.21%)** than the working-tree baseline, including all new production infrastructure and facade signal plumbing. The final production build's JS assets total **729,702 gzip bytes**, an increase of **15,122 bytes (2.12%, 14.77 KiB)**. The pinned dependency is `@tanstack/react-query@5.107.1` with the matching query-core lock entry.

These measurements cover the complete agreed migration and the verification repair below. Excluding that repair's 59 production lines, the migration contains 55,364 lines, an increase of 59 lines; the combined total grows by 118 lines. The migration does not achieve the estimated source reduction: domain drafts, interaction revisions, paged lists and WAL controllers remain, while session and cancellation infrastructure adds production code. The owner requested the complete migration rather than a savings threshold or automatic rollback. No unrelated implementation edits were present in the baseline or final diff. Gzip is measured with Python's identical compression method for both builds; it is the sum of JS assets, excluding CSS, fonts and HTTP transfer framing. Vite's displayed gzip number uses a different method and is not the comparison baseline. Builds retain Vite's existing large-chunk advisory.

## Verification evidence

- `mise run test` passed, including all selected backend packages and the full frontend Vitest run: 193 files, 2,796 tests. Coverage includes isolated harness-failure probes, production-default request lifecycle cases, full-App session isolation and the same-account sign-in case.
- Controlled HTTP regressions failed before their respective repairs and passed afterward for aggregate cancellation and refusal ownership, later-page cancellation without partial publication, fresh-visit failure, unchanged-payload recovery, protected-data clearing, deletion workflow ownership, member acknowledgement and fixed polling cadence. The latest ten new cases cover all three deletion dismissal paths, both membership outcomes and five slow-polling scenarios.
- The focus regression observes the real window visibility listener. Temporarily enabling production focus refresh made the isolated regression fail on an extra HTTP read; restoring the production default made it pass.
- The production browser UI suite passed: 323 scenarios, with harness probes and unchanged immutable visual references.
- `mise run frontend:lint` passed, including formatting, architecture, style and TypeScript checks; `pnpm --dir frontend build` passed. Strict OpenSpec validation and `git diff --check` passed.


### Repair exposed by the browser verification

The unchanged physical-backup filter scenario exposed an existing DateInput focus race in the French narrow layout. Moving focus into the calendar closed its internal dropdown on input blur. A delayed automatic focus return could reopen it after Escape, so the test observed a visible calendar. Both logical and physical backup filters now use the page-owned `useBackupDatePopover`: it controls opening, keeps the calendar open for focus inside, and returns focus and commits the closed state together without a delayed reopen. Outside focus and date selection still close it. The original browser Escape assertions remain intact. Screen regressions for both filters failed before the repair and passed afterward, covering calendar focus, Escape, keyboard date selection and return to the input. This localized repair is included under the standing verification-repair authorization; it adds no request or cache behavior.
