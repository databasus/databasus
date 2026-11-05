## Why

The move to the new frontend left a few small quality defects. A user who declines GitHub or Google sign-in is told "Invalid OAuth provider", the selected sidebar tab is almost indistinguishable from the others in the light theme, the new-version link lost its explanation, and the database search field has no accessible name. The language switch promised for the physical backup config, the verification config and the restore commands also has no test. Each repair is small, so closing them together costs less than tracking them apart.

## What Changes

- Failures the sign-in callback finds before the code exchange show one translated message: sign-in did not finish, start again. The dictionary keys `invalidProvider`, `codeNotFound` and `stateMissing` are deleted from all six dictionaries.
- The selected sidebar tab regains a solid primary background and its white icon on desktop and in the mobile menu, and it is exposed to assistive technology as the current page. The light theme returns to the v3.64.0 blue, and the dark theme uses the blue the theme resolves, which is slightly darker. Tab icons become decorative images, because each button is already named.
- The new-version link returns to a green dot with a tooltip in place of the `↗` character.
- The database search field becomes the same labelled field the notifier list uses.
- New tests pin the language switch for the physical backup config draft, the verification config draft and the restore commands.
- The storage type field and the workspace selector wrap their value at 320 CSS pixels with enlarged text instead of clipping it or pushing the logo out of the row. The browser check for enlarged text now measures the final layout. It used to measure before AntD finished animating `font-size`, which hid these defects and failed only under machine load.
- `window.alert` checks are removed from the tests this change touches. Production code no longer calls `window.alert`, so these checks would fail only if a call came back.

Nothing here breaks callers, schemas, configs or APIs.

### Out of scope

- Removing `window.alert` checks from test files this change does not otherwise touch. About 188 such lines exist across 19 files, and removing them all does not change behavior.
- The contrast of unselected tab icons in the light theme, which stays at about 1.56:1 against the menu background.
- Showing the new-version link on narrow viewports, where the version block stays hidden as it is now.
- The sign-in start failure message, which reports attempt creation problems separately.
- Backend production code, sign-in rules, navigation addresses and permissions.
- Distinguishing individual sign-in failure causes in the message.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-session`: failures the callback finds before the code exchange tell the user that sign-in did not finish and that they should start again.
- `frontend-application`: the shell identifies the current section visually and to assistive technology, and explains the new-version link.
- `frontend-database-screens`: a language change keeps the physical backup config and verification config drafts and the shown restore commands, and requests no new restore token.

The missing database search label already violates the `frontend-design-system` requirement that every field on the database screens has an accessible name, so that capability needs no delta.

## Impact

- Code: `frontend/src/features/edit-storage/ui/edit/EditStorageComponent.tsx`, `frontend/src/widgets/app-shell/ui/WorkspaceSelectionComponent.tsx`, `frontend/src/shared/ui/wrappingSelectClassName.ts`, `frontend/src/pages/auth-callback`, `frontend/src/widgets/app-shell/ui/SidebarComponent.tsx`, `frontend/src/app/ui/ApplicationScreen.tsx`, `frontend/src/pages/databases/ui/DatabasesComponent.tsx`, six files in `frontend/src/shared/i18n/locales`.
- Tests: screen tests beside the components above, the physical restore, physical backup config and verification config tests, the shell browser scenario, and the browser helpers `enlargeText` and `openWorkspaceScreen`.
- Backend tests: `backend/internal/modules/backups/restore_database_deletion_race_test.go` keeps the restore running through the deletion it races against.
- Rules: the change answers to the root `AGENTS.md`, `frontend/AGENTS.md` and `backend/AGENTS.md`.
