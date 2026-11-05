## 1. Map coverage

- [x] 1.1 Run `pnpm --dir frontend test` and record the file and test counts in a new `coverage.md` of this change, with a note that no test case is removed. Verification: `coverage.md` states both numbers.

## 2. Delete dead code

- [x] 2.1 Delete the files and exports in Decision 1 of `design.md`. Delete a dictionary key from all six dictionaries only when `rg` finds no other reader of it or its parent group. Verification: `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend test`.
- [x] 2.2 Remove `export` from the 17 declarations in Decision 2 of `design.md`, and delete the four re-export lines from `entities/databases/index.ts`, `shared/i18n/index.ts` and `shared/theme/index.ts`. Verification: `pnpm --dir frontend typecheck`.

## 3. Move units to their owners

- [x] 3.1 Move `src/constants.ts` to `shared/config/deploymentConfig.ts`, add `shared/config/index.ts` and point its importers at it, with no forwarding file. Move `getOAuthRedirectUri` to `features/oauth-sign-in/model/getOAuthRedirectUri.ts` and leave it out of the slice's public API. Verification: `pnpm --dir frontend test src/shared/api src/features/oauth-sign-in src/pages/auth`, `pnpm --dir frontend build`, and `rg -n "/constants'" frontend/src` finds nothing.
- [x] 3.2 Move `useIsNewGitHubVersionAvailable` to `app/model` and the Turnstile hook and widget to `pages/auth`, with no forwarding exports. Verification: `pnpm --dir frontend test src/app src/pages/auth`, `mise run frontend:lint`, and `rg -n "useCloudflareTurnstile|CloudflareTurnstileWidget|useIsNewGitHubVersionAvailable" frontend/src/shared` finds nothing.
- [x] 3.3 Rename the `models` segments of `entities/storages` and `entities/notifiers` to `model` with `git mv`. Verification: `pnpm --dir frontend test src/entities src/features/edit-storage src/features/edit-notifier`, and `find frontend/src -type d -name models` finds nothing.
- [x] 3.4 Read `ApiComponents` in `entities/backup-configs/model/StorageSwitch.ts` and `entities/disk/model/DiskUsage.ts`, and use the `StorageSwitch` alias in `entities/backup-configs/api/backupConfigApi.ts`. Verification: `pnpm --dir frontend typecheck`, and `rg -n "schema\.gen" frontend/src/entities` finds nothing.

## 4. Check and hand over

- [x] 4.1 Update `frontend/README.md` and `docs/` if they name a moved path. Verification: `rg -n "constants\.ts|entities/(storages|notifiers)/models|shared/hooks/useCloudflareTurnstile" frontend/README.md frontend/AGENTS.md docs .agents/skills --glob '!**/upstream/**'` finds nothing.
- [x] 4.2 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build`, `pnpm --dir frontend test` and `pnpm --dir frontend test:browser --project=ui`. Complete `coverage.md` with any test that changed beyond its imports. Verification: all four pass, and the test count equals the one recorded in 1.1.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
```
