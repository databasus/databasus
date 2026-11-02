## Why

Frontend milestone 8 removes what the migration left behind. Its last two changes rework feedback, checks and tests, and a change lands as one commit, so moving code in the same commit would hide those behavior changes among moved lines. This change does the moves and deletions first.

The residue is real. Eight source files have no importer, one of them empty. `src/constants.ts` sits outside every layer, so the dependency check never looks at it, although 14 files import it. Two entities use a `models` segment while the others use `model`. Shared holds a release check and the Cloudflare Turnstile sign-in widget, which each have one consumer.

## What Changes

- Delete the source files, exports and type aliases that nothing imports, and the dictionary keys that only those files read.
- Remove the `export` keyword from declarations that only their own file uses.
- Move `src/constants.ts` to `shared/config`, except `getOAuthRedirectUri`, which moves to `features/oauth-sign-in`.
- Move `useIsNewGitHubVersionAvailable` to `app/model`, and the Turnstile hook and widget to `pages/auth`.
- Rename the `models` segments of `entities/storages` and `entities/notifiers` to `model`.
- Read generated schema names through the `ApiComponents` alias in the entities that import `components` directly.
- **BREAKING (internal imports only)**: the old paths are deleted with no forwarding exports. Screens, addresses, HTTP requests and visible copy stay the same.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change moves and deletes code without changing any behavior, so `.openspec.yaml` sets `skip_specs: true`.

## Impact

This is the first of three changes of frontend milestone 8, followed by `remove-frontend-check-exceptions` and `close-frontend-behavior-matrix`. The root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md#frontend` and the feature-sliced-design skill.

Affected areas:

- `frontend/src/constants.ts`, which is deleted, and its 14 importers;
- the new `frontend/src/shared/config`, and `frontend/src/features/oauth-sign-in`;
- `frontend/src/shared/hooks`, `frontend/src/shared/ui` and `frontend/src/shared/lib`, which lose units;
- `frontend/src/app/model` and `frontend/src/pages/auth`, which receive them;
- `frontend/src/entities/storages`, `frontend/src/entities/notifiers`, `frontend/src/entities/backup-configs`, `frontend/src/entities/disk`, `frontend/src/entities/verification-agents`, `frontend/src/entities/system` and `frontend/src/entities/databases`;
- `frontend/src/pages/databases` and `frontend/src/pages/storages`, which lose dead files;
- `frontend/src/features/database-connection`, `frontend/src/pages/instance-settings` and `frontend/src/shared/theme`, which lose `export` keywords;
- `frontend/src/shared/i18n`: `dictionaries.ts`, `websitePages.ts`, `index.ts`, and in `locales` only keys that lost their last reader.

There is no backend, HTTP contract, generated code or dependency change.

### Out of Scope

- Replacing `alert` and `ToastHelper`, deleting the baseline mechanism and enabling lint rules, which `remove-frontend-check-exceptions` does. `shared/toast` stays until then.
- Rejecting source files outside the layers in `lint:fsd`, which also belongs to that change because it changes a requirement of `frontend-code-checks`.
- New tests beyond the import paths existing tests need, which `close-frontend-behavior-matrix` adds.
- Moving generic shared units that have one consumer today: `FormValidator`, `normalizeNameList`, `formatDuration`, `usePersistentState` and `ClipboardPasteModalComponent` hold no product policy.
- Converting relative imports to the `@/` alias.
