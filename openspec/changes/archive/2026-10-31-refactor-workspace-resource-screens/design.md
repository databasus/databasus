## Context

Implement after the two credential-flow changes. Evidence below describes the investigation baseline; re-map paths against their implemented results before moving tests.

- `frontend/src/app/model/useApplicationStartup.ts:30` owns essential startup and validates the remembered workspace; `frontend/src/app/ui/ApplicationScreen.tsx:146` already composes the selector into shell slots.
- `frontend/src/features/storages/ui/edit/EditStorageComponent.tsx:77` accepts save completion without a submitted-revision check, and line 95 accepts connection success without checking current settings.
- `frontend/src/features/notifiers/model/useNotifierDraft.ts:87` already checks settings/request revisions; line 186 distinguishes persistence from completed editing.
- `frontend/src/features/storages/ui/StoragesComponent.tsx:40` and `frontend/src/features/notifiers/ui/NotifiersComponent.tsx:41` duplicate scoped selection and list/detail layout. `frontend/src/shared/hooks/usePersistentState.ts:23` reads only during initialization and is not safe unchanged for a moving storage key.
- `frontend/src/features/workspaces/ui/WorkspaceSettingsComponent.tsx:83` replaces all name input after save, and line 182 retains a spinner after an initial load failure.
- `frontend/src/features/workspaces/ui/WorkspaceAuditLogsComponent.tsx:31` repeats presentation already available in `frontend/src/entities/audit-logs/ui/AuditEventList.tsx:32`.
- `frontend/src/features/backups/logical/ui/EditLogicalBackupConfigComponent.tsx:57`, `frontend/src/features/backups/physical/ui/EditPhysicalBackupConfigComponent.tsx:38`, `frontend/src/features/databases/ui/edit/EditDatabaseNotifiersComponent.tsx:8` and `frontend/src/features/databases/ui/DatabaseTransferDialogComponent.tsx:19` import sibling editors. Their fake storage options/remount keys appear at logical line 544, physical line 421 and transfer line 311.

Root `AGENTS.md` and `frontend/AGENTS.md` apply. Decisions implement `docs/architecture.md#frontend`: downward imports, page-first ownership, app composition, public slice APIs and locally owned lifecycle policy. Use the repository FSD, testing, design and localization skills. Do not edit agent instructions to record migrated file inventories.

## Goals / Non-Goals

**Goals:** complete each moved workflow with preserved coverage; use small shared selection/layout units and two focused reusable interactions; give all current consumers valid composition before deleting old paths.

**Non-Goals:** see the proposal. Existing database/backup controllers and configuration policy are opaque consumers except for the narrow callbacks and selection props required here.

## Decisions

### 1. Page-owned resource screens and existing flat entities

Create `pages/workspace-settings`, `pages/storages` and `pages/notifiers`. Each owns its lists, details, local cards, transfer/migration dialogs, loading, error/retry and filters. Workspace members and audit remain local to the settings page. Move workspace creation into app-local UI because app is its only current composition owner. Preserve `entities/workspaces`, `entities/storages`, `entities/notifiers` and `entities/audit-logs` and their domain facades; use public entries externally. The final editor owners are `features/edit-storage` and `features/edit-notifier`, containing their drafts, provider validation and repaired Google authorization unit. Initially leave implementations at their existing owners and expose those local implementations through public APIs to the new pages. Compose every feature-layer consumer above features before extraction; then move each editor and all its page-level importers in one verified task. No old feature forwards imports to a new feature.

Rejected alternatives: keeping entire screens in reusable features hides page ownership; one feature per field/action adds boundaries without independent consumers; renaming flat entity slices for cosmetic consistency adds unrelated churn.

### 2. Selection and layout share mechanics, not requests

Introduce a small generic selection hook with scope key, loaded IDs, loading/readiness and initial desktop/mobile policy. Its state includes the scope that owns the selected ID; never expose an old-scope ID while the new preference is being read. A scope transition reads before writing, and only validated selections are persisted. Failed loads do not look like authoritative empty lists. Use ordinary IDs and revalidate on accepted refreshes. Explicit removal selects the first remaining item; mobile initial load stays on the list. Keep user preference best-effort if localStorage is unavailable.

Use a presentational list/detail layout under `shared/ui` with list/detail/action slots, responsive visibility and a single scroll owner per region. Data loading, search, ordering and the notifier's refresh interval stay on the pages. Apply it to storage/notifier pages now; database selection remains a milestone-5 consumer. Keep the shell's viewport boundary and remove height props only along migrated resource-screen paths.

Rejected alternatives: changing a React key to reinitialize selection hides scope races; changing `usePersistentState` globally would touch unrelated preferences; a universal resource page would absorb domain-specific behavior.

### 3. Storage adopts submitted-snapshot acknowledgment

Reuse `RequestRevision` for identity only. Put storage draft/baseline, persisted ID, settings revision, test revision and request acceptance in the storage interaction, following the existing notifier guarantees without a generic form state machine. Scope work to the current session, workspace and editor identity. Snapshot submitted values. An accepted create always publishes its assigned ID and persisted resource through `onPersisted`; call `onChanged` to complete editing only when no newer draft remains. Once an ID is assigned, subsequent saves update it. Failed or obsolete work cannot close the editor or mark newer values clean.

Returning to baseline makes a form clean. Name-only changes retain a valid settings test. A→B→A still invalidates an earlier in-flight test. Acknowledgment also invalidates tests started during saving, including completed tests, because persisted credentials become the new baseline for omitted secret fields. Reconcile submitted secret input field by field using provider-aware policy: unchanged submitted secrets can be acknowledged/omitted, newer replacements remain unsaved, and SFTP method metadata follows its prerequisite contract. Independent concurrently mounted editors have independent lifetimes. Preserve notifier tests and do not automatically repeat notifier delivery.

Rejected alternatives: boolean-only approval cannot identify which settings were tested; whole-draft replacement loses newer input; freezing all input during save changes the established interaction; a shared domain-aware form engine would couple provider policy.

Detail pages allow one editing mode at a time. A pending rename cannot overlap a settings draft with a hidden older name. The other mode stays unavailable until the current editor closes. An unchanged acknowledged rename closes its editor; settings editing retains its explicit close action. Synchronizing names into independently dirty provider drafts would add state coordination that this restriction avoids.

### 4. Compose existing database consumers in a narrow page boundary

Add `pages/databases` as a thin composition owner around the existing database screen. Place its resource-creation coordinator, database transfer dialog and screen-local database notifier selection editor there. Pass explicit request/render callbacks through the existing database/config components; they may forward opaque slots but may not import reusable editor features. Perform this composition work while storage/notifier editors still reside at their original feature owners, using their public APIs from the page. Keep all connection, schedule, retention, polling and backup logic where it is until its planned milestone.

The page owns storage/notifier choices and nested creation for each active caller. Config forms keep their selected real IDs and config policy, receiving refreshed choices and an explicit create action. A domain picker can be simple presentational entity UI; it cannot import an edit feature. Coordinate `onPersisted` separately from completed creation so refreshing choices does not discard newer editor input. A completed nested create selects its resource only if the captured parent request, target workspace and caller are still current. Cancel restores the prior selection. Retain the existing storage-replacement confirmation and accepted transfer/migration progress.

Translate every current consumer in this change: logical config, physical config, database notifiers and database transfer. Remove their editor imports before extraction; delete `create-new-storage` and picker remount keys during composition. Once every editor importer is a page, move each interaction and change all its page imports atomically, including the repaired callback's public relay entry point. Delete original editor paths and obsolete exports in that same task. Each boundary must pass frontend lint without a forwarding export or a new same-layer baseline allowance.

Rejected alternatives: changing deep imports to a feature barrel still violates same-layer dependencies; lifting the entire database/config implementation early expands milestones 5–7; a global resource modal would mix workspace and caller lifetimes.

### 5. App receives acknowledged workspace events

Keep startup, workspace preference and shell composition in app. Workspace settings receive callbacks for acknowledged rename/deletion and workspace unavailability; app updates its accessible list and selection. Preserve newer name input on a late save while publishing only the acknowledged label. A confirmed missing/inaccessible workspace clears its data before refreshing accessible workspaces; a general network failure remains a local retryable error rather than erasing membership. The settings page and lists render explicit loading/error/empty/success states. Use local request revisions in addition to the session guard so cleanup/setup and workspace replacement reject old responses.

Explicit workspace selection invalidates a pending access refresh, just as acknowledged workspace mutations do. Its older list or error cannot replace the newer navigation context or unmount a current draft. Accepting only successful lists would still let a stale failure replace the whole application with recovery.

Rejected alternatives: full-page reload loses unrelated app state; using the remembered ID as authority bypasses availability validation; checking only session identity cannot reject older requests within one account.

### 6. Finish audit and presentation reuse with shrinking baselines

Workspace audit keeps `useAuditEventPages`, filters, page size and scroll ref, and delegates rows/cards/footer to `AuditEventList` with author shown, workspace hidden, short time and workspace-specific empty copy. Keep literal event content and existing instance/user modes. Add bounded read-failure recovery where absent. Apply existing AntD controls, semantic roles and context feedback to migrated screens, update all six dictionaries for changed copy and preserve drafts on locale/theme changes. Remove exact repaired entries from `frontend/fsd-baseline.json` and `frontend/style-baseline.json`; no new cross-feature import or broader exception is permitted.

Rejected alternatives: a second audit renderer repeats established domain UI; moving it into shared would put domain presentation below its owner; broad baseline suppression would conceal new violations.

## Risks / Trade-offs

- A broad move can silently drop scenarios. Create `coverage.md` before edits, mapping each existing API/editor/list/member/transfer/config/OAuth test to its final owner and recording new gaps; justify every removal.
- Concurrent refresh can unmount a dirty editor. Parent lists accept persistence separately from completion, and tests exercise creation, refresh, newer input and cancel together.
- Resource coordinator plumbing crosses unmigrated components. Keep callbacks narrow, trace every caller and preserve config replacement/transfer tests; do not use a new dependency exception.
- Geometry is not established by happy-dom. Browser UI scenarios cover narrow/desktop layouts, both themes, long translations, 200% text, focus and scroll ownership; real-API browser tests cover workspace creation, local storage read-back and workspace isolation.

## Migration Plan

Land the two prerequisite fixes first. Map coverage and move screens to page owners while editors remain at their original owners. Repair relocated style occurrences rather than transferring baseline exemptions. Add shared mechanics, then compose database/notifier/backup-config consumers through pages and callbacks before moving either editor. Extract each editor together with all of its remaining page importers and deletion of its old paths in a single verified task; add storage lifecycle safeguards after that mechanical move. Every task boundary passes its named lint/tests without unresolved imports, feature-to-feature forwarding exports or expanded dependency exceptions. Update the architecture's current composition description and affected documentation; leave unrelated baseline residue for its assigned milestones. Mark milestone 4 complete only after this change's checks and mandatory review pass. All tasks stop with reviewable uncommitted work; no commit, push or archive is part of implementation.
