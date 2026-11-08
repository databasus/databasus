## Why

Application sections and database tabs reset after reload, and browser history cannot restore previous selections. Database selection is already remembered locally, but it cannot be shared through a link. Navigation needs explicit addresses with predictable recovery when the requested screen or resource is unavailable.

## What Changes

- Give application sections, workspaces, databases and database tabs canonical addresses that survive reload and support browser Back and Forward.
- Open the workspace named by an accessible link, regardless of the locally remembered workspace, and open database details directly on mobile as well as desktop.
- Automatically select an available destination when a requested section, workspace, database or database tab is unavailable; replace the address with the destination actually shown.
- Preserve a cold deep link through password, two-factor and OAuth sign-in, while explicit logout and account-ending transitions clear the previous account's navigation intent.
- Make the URL authoritative for addressed selections, retaining existing local preferences only for entry points that do not specify a selection.
- **BREAKING behavior:** remove the latent database-tab preference that can revive an unavailable tab after switching databases. An unavailable tab becomes Backups in both the interface and URL; browser history remains the way to restore an earlier screen. No HTTP operations, persisted server schemas or existing callback addresses change.

## Capabilities

### New Capabilities

- `frontend-navigation`: Canonical screen addresses, history, direct entry, automatic fallback and navigation access isolation.

### Modified Capabilities

- `frontend-application`: Extend existing navigation to addressed sections and reconcile unavailable explicit workspace selections with canonical fallback destinations.
- `frontend-database-screens`: Replace the remembered unavailable-tab preference with canonical Backups fallback; distinguish unavailable-database fallback from retryable load failures, and reconcile copy, transfer and deletion with the current URL while preserving accepted completion after same-database tab changes.
- `frontend-session`: Preserve a new deep-link sign-in intent without carrying a previous account's selections into another session.
- `frontend-workspace-screens`: Give explicit database links precedence over remembered selection and the mobile list-first default while retaining storage and notifier selection behavior.

## Impact

The frontend app routing, startup workspace selection, shell navigation and database page selection contracts change. Authentication entry and callback handling gain a validated local return destination. Existing React Router and domain HTTP operations are sufficient; no new production dependency or backend operation is planned. Browser tests cover the built application and direct nested-path reloads.

Implementation answers to root `AGENTS.md` and `frontend/AGENTS.md`. Update `docs/architecture.md` for URL ownership and `docs/testing.md` if navigation coverage or test instructions change. Browser end-to-end work also follows any instructions found next to the affected suite.

### Out of scope

- Loading page code on demand, bundle optimization or router replacement.
- Links to an individual storage, notifier, backup, verification, restore or audit record.
- URL persistence for form drafts, wizard steps, dialogs, filters, pagination or scroll position.
- Restoring unsaved drafts after reload or adding navigation confirmation dialogs.
- New permissions, access-request flows, HTTP operations, server migrations or deployment changes.
