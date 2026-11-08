## Context

See `proposal.md` for motivation and scope. The implementation follows root `AGENTS.md`, `frontend/AGENTS.md`, and the frontend ownership and session rules in `docs/architecture.md:332-378`.

- `frontend/src/app/ui/App.tsx:23-37` already mounts BrowserRouter, two callback routes and the root application route. Its catch-all redirects to root.
- `frontend/src/app/ui/ApplicationScreen.tsx:30` owns the main tab in state and changes it through shell callbacks.
- `frontend/src/app/model/useApplicationStartup.ts:67-85` chooses an accessible current or remembered workspace. Its access reconciliation at lines 130-147 already refreshes workspaces after a refusal and rejects superseded work.
- `frontend/src/pages/databases/ui/DatabasesComponent.tsx:199-206` uses `useScopedSelection`; `frontend/src/shared/hooks/useScopedSelection.ts:14-40` restores per-workspace local selection, chooses the first desktop item and initially shows the mobile list.
- `frontend/src/pages/databases/ui/DatabaseComponent.tsx:41-68` holds a preferred tab separately from its displayed fallback. This is the behavior deliberately replaced by canonical tab navigation.
- `frontend/src/pages/auth-callback/ui/AuthCallbackPage.tsx:33-36` currently sends successful OAuth sign-in to `/`.
- `backend/internal/lib/httpserver/unknown_path.go:15-35` already serves the SPA entry for non-API nested paths. No backend change is planned; the built-image browser suite must verify this direct-entry path.

## Goals / Non-Goals

**Goals:** Give addressed selections one authority; keep account capabilities and request acceptance tied to the active session; resolve unavailable destinations without navigation loops. Implement the route and fallback contracts in `specs/frontend-navigation/spec.md`.

**Non-Goals:** Move domain loading or permission rules into the router; retain unmounted drafts in a global store; create a universal resource-page controller; introduce a second navigation system for storage and notifier details. Product exclusions are in `proposal.md`.

## Decisions

### 1. Extend the existing browser router with nested application routes

Use the address table in frontend-navigation. Keep the authenticated application shell mounted across section changes, with route content inside it. Keep callback routes separate and retain their existing paths. Use application navigation links for main sections and database selection so opening a link in another tab works naturally; retain the database tab controls' accessible tab interaction while routing their changes.

Route composition and URL parsing belong in `app`, as `docs/architecture.md` requires. The shell receives link destinations and selection. The database page exposes controlled database/tab inputs and navigation callbacks through its public API; it retains collection reads, membership validation, offered-tab rules and mutation acceptance. Pages do not import app routing modules. Shared navigation provides the local path grammar, builder and an identity for deliberate navigation. App passes that identity with controlled database selection so an automatic replacement continues the current availability resolution, while a new navigation can retry previously rejected candidates. Resource fallback policy remains in app and pages.

Alternatives rejected: query parameters under `/` obscure section boundaries without reducing access reconciliation; hash routing adds a second address convention despite existing nested-path support; router replacement adds migration work without a required capability.

### 2. Make the resolved URL authoritative and restrict preferences to unspecified selections

Resolve an explicit workspace against accepted startup workspaces before mounting workspace content. Resolve a database against the successfully loaded complete collection of that workspace, before mounting its detail requests. Search filters do not affect membership. Validate any returned detail's workspace before presenting data or actions to cover a transfer between list and detail reads.

Database navigation no longer lets `useScopedSelection` silently replace an explicit ID with a remembered one. Keep its existing behavior for storages and notifiers. For database entry without an explicit ID, retain desktop remembered/first selection and mobile list-first behavior, then publish any selected desktop detail into the URL with replacement. A viewport change alone must not rewrite a resolved explicit selection. Workspace selection on global pages remains a remembered preference; it does not change the global screen address.

Switching databases carries the currently displayed tab when the new database offers it; otherwise the destination is Backups. Remove the separate preferred-but-hidden tab state. Reload and history use only the addressed tab. This is the intentional behavior break identified in the proposal.

Alternatives rejected: synchronizing independent route and React selection states creates competing authorities; persisting hidden tab preferences makes the same address dependent on navigation history outside browser history; removing preferences for all resource pages expands scope.

### 3. Resolve unavailable destinations in a bounded sequence

Use the following order, with available items in the current complete list order:

| Requested destination | Resolved destination |
| --- | --- |
| Unknown or unsupported database tab | Same database, Backups |
| Database missing from an accessible workspace | First available database in that workspace, requested tab if offered, otherwise Backups |
| No available databases | Addressed workspace database list |
| Workspace missing from accepted accessible workspaces | First accessible workspace's databases destination |
| No accessible workspaces | Root empty-workspace experience |
| Administrative screen without administrator capability | Default databases destination or root empty-workspace experience |
| Wholly unknown application address | Root, then normal default resolution |

Do not preserve a database ID while substituting another workspace. Retain the requested URL during startup, ordinary pending reads and retryable failures. For a detail 403/404 after successful collection validation, clear its details, reconcile that workspace's collection and access, then select a surviving destination. Keep rejected candidates excluded for the current resolution so a stale list cannot repeatedly select them. Exhaustion ends at the list or permitted empty-workspace destination; a failed reconciliation ends in existing recovery feedback. A 401 follows session invalidation rather than resource fallback. Required parallel collection reads must still observe a later access refusal after an earlier transient failure.

Publish only a resolved destination; reject obsolete read-driven fallback decisions after route, workspace or session changes. Mutation completion follows decision 4, including its same-database tab exception. Permission checks use the resolved resource's workspace, never an unrelated selector preference. Update `docs/architecture.md` to distinguish addressed navigation from unaddressed local resource selection and describe the ownership of reconciliation.

Alternatives rejected: an unavailable landing page contradicts the owner's requested automatic recovery; blindly falling back on any exception hides outages; discovering and following a database's actual foreign workspace makes a malformed workspace/database pair ambiguous and risks using the wrong capabilities.

### 4. Separate deliberate navigation from canonical replacement

Push one history entry for a user-selected resolved section, workspace, database or database tab. Resolve an unsupported requested tab before pushing a different database destination so that no intermediate invalid tab entry is created. Replace the current entry for cold-entry normalization, access fallback, selected-database deletion/transfer and selected-workspace deletion. Copy and creation push the new selected resource after their existing completion acceptance checks; creation still waits for the wizard to close.

Preserve the existing exception for a tab change within the continuously selected database: a pending copy can select its copy at the currently displayed offered tab, otherwise Backups; a pending delete/transfer resolves the removed selection with replacement; a rename retains the current tab address. None of these late answers shows feedback after the tab change. Bind mutation selection acceptance to the database/workspace/main-section selection lifetime rather than every tab route revision. Changing database, workspace or main section invalidates the old action's right to select a different resource, even after returning to the source. Accepted list removals still invoke ordinary availability reconciliation if the current resource disappears. Reads and read-driven fallback remain bound to the full current route and session.

Browser Back/Forward read the route and validate it again; they do not push mirrored state changes. Repeated selection of the current destination is a no-op. Mobile Back to list navigates to the list address rather than calling history.back, which might leave the app after direct entry.

Alternatives rejected: replacing every click prevents useful Back navigation; pushing normalization and mutation cleanup creates traps; remounting the shell for every address resets unrelated presentation state and pending lifecycles.

### 5. Keep cold-entry sign-in intent separate from account-owned navigation

When a signed-out user opens a recognized screen address, capture only its validated local path for that sign-in intent. Keep password and two-factor flow on the addressed entry until acceptance. For OAuth, bind the return path to the existing tab-local attempt and consume it with the attempt before exchange; carry the validated target through the successful callback instead of always choosing `/`. Preserve the existing random state, provider, redirect URI, ten-minute expiry and one-exchange behavior. If the initiating intent is abandoned, discard its destination as well.

Do not forward arbitrary `returnTo` input or accept URLs with schemes, authorities, callback targets, query data or fragments. After authentication, resolve access using only the accepted account's startup data. Capture the consumed target before session activation clears authentication metadata, and apply it only if that authentication completion was accepted. Logout, password-change logout, active-session refusal and active-session replacement reset the address to `/` and clear old intent before the signed-out view can treat it as a new deep link. New explicit navigation to a deep link while signed out starts a fresh intent.

Alternatives rejected: retaining every previous account URL across sign-out violates session reset; always returning to `/` breaks shareable links for signed-out recipients; unvalidated return URLs permit external redirects and callback loops.

## Risks / Trade-offs

- Automatic fallback can open a different database than the original link intended. Mitigation: always show that database's existing name and update the address before enabling its actions; never display foreign details while falling back.
- Access can change between collection and detail reads. Mitigation: guard membership and active navigation at both acceptance points, reconcile confirmed refusal and exclude rejected candidates from the current resolution.
- Existing tests assume buttons for main navigation or a root address after entry. Mitigation: update real-screen and browser helpers to use accessible links and assert the resolved address as well as content; do not weaken permission or request-lifecycle assertions.
- A browser Back visit can legitimately hit a deleted resource. Mitigation: revalidate it and replace that entry using the same fallback policy.
- Restored addresses do not restore unsaved drafts. Mitigation: retain current screen-owned draft lifetimes and keep draft persistence explicitly outside this change.

## Migration Plan

No server migration or new dependency is required. Keep `/`, `/auth/callback` and `/storages/google-oauth` reachable; introduce the new routes in the same frontend build as URL-driven selection. Keep local selection preferences only as unspecified-entry defaults and remove duplicate addressed-selection state rather than adding compatibility aliases. Release through the existing image build. Rolling back the frontend restores root navigation; newly introduced deep links then follow the existing unknown-address redirect to `/`.

Verify route behavior through real-screen HTTP-boundary tests, browser UI navigation and built-image end-to-end scenarios. The latter must cover direct nested-path entry/reload, Back/Forward, and inaccessible/mismatched workspace/database fallback with real account permissions. Authentication HTTP-boundary tests cover password/2FA/OAuth return paths, invalid targets and stale completion without contacting external providers.
