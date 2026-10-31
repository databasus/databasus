## Context

See `proposal.md` for motivation. The relevant existing behavior is grounded in:

- `frontend/src/features/storages/ui/edit/storages/EditSFTPStorageComponent.tsx:24` and `frontend/src/features/storages/ui/show/storages/ShowSFTPStorageComponent.tsx:13`: the method is inferred from a write-only private key.
- `backend/internal/modules/storages/internal/httpapi/sftp_settings.go:16`: both secrets are omitted from responses.
- `backend/internal/modules/storages/internal/providers/sftp/provider.go:64`: omitted password and key are independently restored.
- `backend/internal/modules/storages/internal/providers/sftp/sftp_server.go:183`: the connection builds authentication methods from both nonempty secrets.
- `backend/internal/modules/storages/internal/providers/sftp/settings_repository.go:59`: the repository already persists both fields; `backend/internal/lib/crypto/field_encryptor.go:27` preserves empty and already encrypted inputs unchanged.
- `backend/internal/modules/storages/internal/storage/service.go:182` and `backend/internal/modules/storages/internal/storage/connection_test_service.go:115`: update uses a locked merge/validate path, and unsaved tests reuse that resolution.
- `backend/internal/modules/storages/secrets_test.go:334`: existing API coverage explicitly creates and retains both SFTP credentials.

Root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, the API-design skill and `docs/architecture.md` constrain contract-first development, provider-owned policy, encryption, access checks and generated aliases. Frontend design/localization skills apply to the controls and feedback.

## Goals / Non-Goals

**Goals:** one exact selected credential set for create/test/update, truthful nonsecret reads and preservation of existing rows without a migration.

**Non-Goals:** credential policy outside SFTP, stored mode columns and SSH host-key verification. This patch must not imply the current host-key flag is effective.

## Decisions

### 1. Use a required authentication-method set in the existing settings variant

Define **SFTP authentication method** in `docs/glossary.md` as a credential kind selected for one SFTP storage, password or private key; one storage can select either or both. Its canonical names are `sftp.AuthMethod` in the provider, `SftpAuthMethod` in generated code and `authMethods` in API settings, with values `PASSWORD` and `PRIVATE_KEY`. Keep the short definition in the glossary and the selection policy in the storage specification.

Add `SftpAuthMethod` with `PASSWORD` and `PRIVATE_KEY`, and required `authMethods` to `SftpStorageSettings` in `openapi.yaml`. It is an array with one or two unique members. `password` and `privateKey` remain optional, write-only and nonempty when supplied. Keep operations and access marks unchanged; invalid selections use the existing `invalid_request` Problem. The same settings shape serves create, update, unsaved tests and responses.

Carry the selected methods in the provider's settings value during request resolution. On repository load, derive them from nonempty stored credential fields; return a stable password-then-key order. On read/create/update, report the resolved persisted set while omitting secrets. No new persisted mode or presence booleans are needed. Update generated types and all SFTP fixtures/callers rather than deriving missing selections from incoming secret presence.

Rejected alternatives: a single exclusive mode cannot represent existing dual-credential rows; presence booleans plus clear flags introduce redundant states; a persisted enum needs migration and can disagree with the existing credential fields; optional selection would retain ambiguous method switching.

### 2. Resolve selected credentials once, inside the SFTP provider

For each selected method, use a nonempty supplied credential, otherwise retain that same credential from stored settings. Excluded methods resolve to an empty credential. Reject empty/invalid selections, missing selected credentials and supplied credentials outside the selected set. Keep HTTP presence and min-length validation in the contract; do not turn an explicitly supplied empty value into omission.

For create, there is no stored fallback. For update, use the existing transaction and storage-row lock to read, resolve, validate and persist. For unsaved tests, use that same merge/validate policy but persist nothing. A password excluded by a key-only test must be absent from the effective connection settings, including the connection-pool identity. Saved tests continue to load their exact stored credentials. No other provider changes its secret-omission policy.

Keep the Fields rule in `.agents/skills/api-design/SKILL.md` general: omission alone retains a stored secret, and removal requires an explicit request signal. Link to `backend/api/openapi.yaml` for provider-specific semantics rather than duplicating SFTP behavior in agent instructions. Document the selected-credential retention and excluded-credential removal rules in the contract's SFTP settings schema and affected operations. Keep the response-secret rule and its issued-credential exception intact. The delta also modifies both existing storage requirements governing omitted secrets, for update and unsaved tests, preserving workspace access, type and prefix protections.

Rejected alternatives: clearing fields only in React cannot remove secrets restored by the backend; separate test and save resolution can test a fallback that will not be used after saving; a cross-provider credential service adds scope and weakens domain ownership.

### 3. Preserve encrypted values and current connection behavior

Keep the selected omitted credential's existing encrypted bytes. Existing idempotent field encryption preserves those bytes during upsert; replacing a selected credential encrypts the new value, and excluding it persists an empty value under the current repository representation. Derivation treats both NULL and empty fields as absent. No startup rewrite, re-encryption job, schema migration or invented preferred method is required. Preserve the current password-then-key connection ordering when both methods are selected.

Rejected alternatives: normalizing every old row risks losing a working credential and adds upgrade work; choosing a preferred method silently changes existing authentication behavior.

### 4. The UI edits the method set explicitly

Use the returned method set for editor and detail presentation. A new SFTP draft defaults to password authentication. Expose the existing password, private-key and dual-method capability through one labeled selector; render the fields selected by that set. Empty inputs on a saved selected method mean retain its stored credential, shown with translated saved-credential guidance. Removing a method drops any unsaved replacement for it; selecting it again can retain its persisted credential until a successful save has removed it.

Changing methods or credentials invalidates connection approval. Renaming keeps the method set and omits untouched credentials. After save, acknowledge the returned methods and blank only submitted secret input that is unchanged, leaving newer input intact. Keep domain validation in the SFTP settings unit; do not import database SSH models. Use all six dictionaries and real locale providers in screen tests.

Rejected alternatives: continuing to infer methods from secret presence misrepresents reads; automatic removal during render or selection would mutate saved credentials before explicit save.

## Risks / Trade-offs

- Requiring `authMethods` breaks old SFTP request bodies. Regenerate all clients and update every known caller in the same release; return a clear 400 for missing selections and provide no compatibility inference.
- A dual-method test might succeed through an unintended fallback. API tests use local SFTP servers accepting only one credential to prove that an excluded method is never attempted; merely inspecting JSON does not prove it.
- Presence metadata must never become a secret-return path. API tests inspect create/read/list/update responses and encrypted columns, and assert no write or connection on refused requests.
- The current host-key flag is ignored by the connection (`sftp_server.go:207`). Keep that limitation explicit in the proposal and do not present this credential repair as a host-authenticity fix.

## Migration Plan

Regenerate the API, update backend/provider and frontend consumers together, add the glossary definition, reconcile the API-design Fields rule and publish the breaking request-field change. Existing saved rows are readable immediately because metadata is derived. Tests first pin password-only, key-only and dual-credential records, then exercise rename, replace, switch in both directions, refused switch and read-back. A repository reload proves selected credentials remain usable without relying on in-memory settings.

No data rollback is needed. A deliberate user switch removes an excluded credential, so rolling back application code cannot recover that removed secret; the user would need to enter it again. This is explicit save behavior, not upgrade-time deletion.
