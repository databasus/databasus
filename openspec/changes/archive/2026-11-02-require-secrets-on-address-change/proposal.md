## Why

A saved storage, notifier or database reuses its stored secret when a request leaves the secret out, whatever address the request names. A user who may not see that secret can therefore point the resource at their own server and receive it. A viewer can do this today through the test of unsaved notifier settings, and a member can do it through a save or a test in all three families.

## What Changes

- **BREAKING** When a request changes the address of a saved storage, notifier or database and leaves out a secret that the address uses, the request gets `400` with `code` `invalid_request` and a field error naming a secret it still needs. Nothing is saved or sent. This holds for a save, a test of unsaved settings and the limited-user check and creation, for every workspace role. A script that changes a host without sending the secret again must now send it. Nothing is kept as a shim: the unconditional fallback to the stored secret is deleted.
- **BREAKING** A viewer gets `403` with `code` `permission_denied` from `POST /notifier-tests`. Testing unsaved notifier settings needs the right to change the workspace's content, as it already does for storages and databases. A viewer keeps the test of a saved notifier.
- A stored secret that the request leaves out and the settings no longer use is removed when the address changes, so a later request cannot bring it back at the new address.
- Only the fields of the new term "resource address" count: host, port, endpoint, URL, the Azure account name, the Telegram Bot API URL and proxy, the SSH tunnel's address and, for MongoDB, the SRV flag. They are compared exactly, without normalization.
- While an address in an edit form differs from the saved one, the form requires each secret the settings use and says the secret must be entered again. When the address returns to the saved value, the form stops requiring it.
- The API contract and the `api-design` skill state the new rule for omitted secrets.

### Out of scope

- Fields other than addresses. Changing a login, a TLS mode, certificate verification, an S3 bucket or region, or a Telegram chat keeps the stored secret. By the user's decision, a member can still weaken the transport of a secret at the same address.
- Protecting the backups themselves. A member can still point a storage at their own server with their own keys, and a viewer can still download backups.
- The restore path. A restore target takes its secrets only from the request and never reads stored ones.
- Tests of saved resources. They use only stored settings, so they stay as they are, a viewer included.
- A name resolving differently over time. The rule compares what was typed, so control over the DNS of a saved host is not covered.
- Existing stored data. No migration is needed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `secret-handling`: adds the rule that a stored secret is used only at the address it was saved for.
- `storages`: changing and testing a storage follow that rule.
- `notifiers`: changing and testing a notifier follow that rule, testing unsaved settings needs the right to change content, and the form asks for the secret again.
- `databases`: changing and testing a database and the limited-user operations follow that rule.
- `workspace-access`: a viewer no longer tests unsaved notifier settings.
- `frontend-database-connection`: the connection form asks for the secret again after an address change.

## Impact

- **API:** the contract text of `updateNotifier`, `testNotifierSettings`, `updateStorage`, `testStorageSettings`, `updateDatabase`, `testDatabaseSettings`, `checkLimitedUser` and `createLimitedUser`, and of the fields that describe omitted secrets. The request and response shapes do not change, and no new error code appears. The behavior change breaks scripts as described above.
- **Backend:** the secret merge of every provider with a secret bound to an address in `backend/internal/modules/storages`, `notifiers` and `databases`, and the access check of `TestSettings` in `notifiers`.
- **Frontend:** `features/edit-storage`, `features/edit-notifier`, `features/database-connection`, the notifier validators in `entities/notifiers`, and all six interface dictionaries.
- **Documents:** `.agents/skills/api-design/SKILL.md` and `docs/glossary.md`.
- **Rules this change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
