## 1. Setup

- [x] 1.1 Install the frontend dependencies in this worktree without changing the lockfile, and verify with `pnpm --dir frontend install --frozen-lockfile` followed by `pnpm --dir frontend typecheck`

## 2. Sign-in message

- [x] 2.1 In `AuthCallbackPage.test.tsx`, add "explains that sign-in did not finish when the provider refuses consent", asserting `DICTIONARIES.en.app.oauthCallback.notCompleted` and the absence of "Invalid OAuth provider", and make the existing table cases for missing code, missing or wrong state, legacy provider state, expired attempt and missing, malformed or consumed metadata assert the same; see them fail with `pnpm --dir frontend test -t "AuthCallbackPage"`
- [x] 2.2 Add `app.oauthCallback.notCompleted` to all six dictionaries, show it in `AuthCallbackPage.tsx` for failures that are not an `ApiError`, delete `invalidProvider`, `codeNotFound` and `stateMissing`, and verify with `pnpm --dir frontend test -t "AuthCallbackPage"` and `pnpm --dir frontend test -t "uses the same"`

## 3. Selected tab

- [x] 3.1 Add `SidebarComponent.test.tsx` with "marks the selected tab with aria-current" for the desktop and mobile layouts, asserting exactly one `aria-current="page"` element and tab icons with an empty `alt`, and with "moves aria-current to the activated tab", which activates another tab through a small stateful wrapper and asserts that the previous button loses the state; see them fail with `pnpm --dir frontend test -t "aria-current"`
- [x] 3.2 In both `SidebarComponent.tsx` layouts, render each entry as AntD `Button`, `type="primary"` with `selectedIcon` and `aria-current="page"` when selected and `type="text"` otherwise, with `alt=""` on tab icons, replace `findByAltText('Profile')` in `App.test.tsx` with a role query, and verify with `pnpm --dir frontend test -t "aria-current"`, `pnpm --dir frontend test -t "ApplicationScreen"` and `pnpm --dir frontend test src/app/ui/App.test.tsx`
- [x] 3.3 Add a scenario to `frontend/browser/scenarios/appShell.spec.ts` for both themes at 769 and 320 CSS pixels that, with the pointer moved away, asserts `aria-current`, the white icon, at least 4.5:1 between white and the computed background, at least 4.5:1 between the computed label color and the background on the mobile entry, and that these move after activating another section; verify with `pnpm --dir frontend test:browser --project=ui`

## 4. New-version link

- [x] 4.1 In `ApplicationStartup.test.tsx`, which already sets the running version, add "explains the new version link when a newer release exists", asserting the release address, the accessible name, the absence of `↗` and the tooltip on hover, with a second test for the tooltip on keyboard focus, and see them fail with `pnpm --dir frontend test -t "explains the new version link"`
- [x] 4.2 In `ApplicationScreen.tsx`, wrap the `<a>` in `Tooltip`, shown on hover and focus, and replace `↗` inside the link with a `bg-success` dot; verify with `pnpm --dir frontend test -t "explains the new version link"`
- [x] 4.3 Add "hides the new version link when the running version is the latest", "hides the new version link when the release lookup fails" and "hides the new version link in a development build", the last with the version stubbed as `useVersionCheck.test.tsx` does, and verify with `pnpm --dir frontend test -t "hides the new version link"`

## 5. Database search

- [x] 5.1 In `DatabasesComponent.test.tsx`, add "labels the search field" with at least five databases, and see it fail with `pnpm --dir frontend test -t "labels the search field"`
- [x] 5.2 Replace the `<input>` in `DatabasesComponent.tsx` with `Input` and `aria-label`, following `NotifiersComponent.tsx`, remove the `window.alert` checks from `DatabasesComponent.test.tsx`, and verify with `pnpm --dir frontend test -t "DatabasesComponent"`

## 6. Language switch

- [x] 6.1 In `PhysicalRestoreComponent.test.tsx`, add "keeps the restore commands when the language changes", asserting the same token in the commands, one token request and translated titles, and verify with `pnpm --dir frontend test -t "keeps the restore commands when the language changes"`
- [x] 6.2 In `EditVerificationConfigComponent.test.tsx`, add "keeps the draft when the language changes", asserting that the edited value remains, no save request is sent and a form label appears in the selected language, remove the `window.alert` checks from that file, and verify with `pnpm --dir frontend test -t "EditVerificationConfigComponent"`
- [x] 6.3 In `EditPhysicalBackupConfigSection.test.tsx`, add "keeps the draft when the language changes", asserting that the edited value remains, no save request is sent and a form label appears in the selected language, and verify with `pnpm --dir frontend test -t "EditPhysicalBackupConfigSection"`

## 7. Final checks

- [x] 7.1 Format the changed files and verify with `mise run frontend:lint`, `pnpm --dir frontend typecheck` and `pnpm --dir frontend build`

## 8. Browser check stability

- [x] 8.1 Make `enlargeText` wait for CSS transitions, `openWorkspaceScreen` wait for the drawer to return focus, and the SMTP accessibility scan wait for animations, and verify with `pnpm --dir frontend test:browser --project=ui`
- [x] 8.2 Let the storage type field and the workspace selector wrap their value through `WRAPPING_SELECT_CLASS_NAME`, and verify with `pnpm --dir frontend test:browser --project=ui`
- [x] 8.3 Make `TestDeleteDatabaseRacingRestoreStart` restore into a blocked target, and verify with `go test ./internal/modules/backups/ -run TestDeleteDatabaseRacingRestoreStart -count=12` from `backend/`

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
