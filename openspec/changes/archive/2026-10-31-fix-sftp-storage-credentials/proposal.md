## Why

Saved SFTP credentials are correctly hidden by the API, but the interface uses the missing private key to guess the authentication method. Switching the visible method also leaves the old credential active because the backend restores every omitted secret, so the displayed choice and actual connection can disagree.

## What Changes

- Return and accept an explicit SFTP authentication-method set while keeping passwords and private keys write-only. Derive the returned set from stored credentials without adding a persisted mode or rewriting encrypted values.
- Preserve each selected credential when omitted, replace it when supplied, and remove a credential only when the submitted method set explicitly excludes it. Reject a selected method without a supplied or retained credential.
- Use identical credential resolution for unsaved connection tests and saves, with tests changing no stored values.
- Display password-only, key-only and existing dual-credential settings truthfully. Preserve both stored credentials until the user explicitly changes the selected set.
- **BREAKING**: require `authMethods` in SFTP create, update and unsaved-test settings, and reject explicitly empty credentials instead of treating them as omission. Delete secret-based UI inference and implicit method selection; do not retain a missing-field compatibility path. Existing script callers must send their intended method set and omit unchanged secrets. Saved rows require no migration.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storages`: explicit SFTP method metadata, credential selection and replacement, consistent test/save behavior and preservation of existing dual-credential rows.

## Impact

Second milestone-4 change. It can be implemented independently of the Google Drive repair; both precede `refactor-workspace-resource-screens`. Root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md` apply, with contract-first API design and the testing, frontend design and localization skills.

Affected areas are `backend/api/openapi.yaml`, generated server/client/frontend types, SFTP settings conversion and provider resolution, storage API tests, SFTP editor/detail views, fixtures and all six interface dictionaries. Define SFTP authentication method and its canonical code/API names in `docs/glossary.md`. Keep the Fields rule in `.agents/skills/api-design/SKILL.md` general: omission alone retains a secret, while removal requires an explicit request signal. Link to the API contract for provider-specific retention and removal semantics, and document SFTP `authMethods` behavior there. Update affected storage-contract documentation in the same change. No database migration, new endpoint or dependency is needed.

### Out of Scope

- Database SSH authentication policy or a common credential model for every provider.
- Changing SFTP transport behavior, connection pooling or provider ordering when both methods are selected.
- Host-key trust management. The current connection unconditionally ignores the host key despite the exposed flag; this remains a separately identified defect, not a guarantee repaired by credential selection.
- The Google Drive flow, page migration and reusable editor extraction owned by the other two changes.
