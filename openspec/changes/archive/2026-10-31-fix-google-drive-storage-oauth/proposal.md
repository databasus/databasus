## Why

Google Drive authorization currently puts the entered client secret into the redirect URL and accepts an unbound callback. The provider is planned for removal, so this change fixes that exposure and callback acceptance in the frontend without building a new server-side OAuth subsystem.

## What Changes

- Keep the storage draft and entered credentials in the initiating editor's memory. Send only a cryptographically random, short-lived, single-use state through Google.
- Turn the existing callback page into a same-origin relay of the authorization code or refusal to the initiating editor. Continue the existing browser-to-Google code exchange there, then use the existing storage test and save operations.
- Preserve the originating form and its storage picker. Authorization does not create or update a storage until the user explicitly tests and saves it.
- Reject unsolicited, expired, replayed and obsolete callbacks; preserve existing stored credentials on cancellation or failed reconnection. Reconnection requires entering the client secret again because the API does not return it.
- **BREAKING**: remove JSON-state callbacks and `StorageOauthDto` without a compatibility parser. An authorization started before the update, or interrupted by reloading the initiating tab, must be restarted. Keep `/storages/google-oauth` and stored credentials unchanged.

## Capabilities

### New Capabilities

- `google-drive-storage-oauth`: bounded browser authorization, credential containment, explicit persistence and recoverable failure for Google Drive storages.

### Modified Capabilities

None. Existing storage API secret-omission and persistence requirements remain in force.

## Impact

First of three milestone-4 changes, followed by `fix-sftp-storage-credentials` and `refactor-workspace-resource-screens`. Implementation follows root `AGENTS.md` and `frontend/AGENTS.md`, including the design, localization, testing and FSD skills.

Affected areas are the Google Drive storage form, its editor lifecycle, `pages/storage-oauth`, the storage public API, six interface dictionaries, screen tests and browser UI scenarios. Update `docs/architecture.md` to describe the bounded frontend interaction and `docs/glossary.md` to define a storage authorization attempt separately from an OAuth sign-in attempt. No backend code, migrations, HTTP contract, generated API files, new package or stored-credential migration is needed.

### Out of Scope

- Removing Google Drive itself, changing scopes or replacing the user's Google OAuth client configuration.
- Server-owned attempts, token exchange endpoints, persistent draft recovery, a general OAuth framework or a second popup transport.
- Changing sign-in OAuth, extracting the milestone's reusable editors or refactoring other provider forms.
- Claiming that a browser-held client secret is confidential from the user, extensions or injected same-origin code. This is a limited frontend repair, not a confidential-client redesign.
