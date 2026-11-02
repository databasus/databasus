# Verification list and agent screen refactoring coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-02, before this change: 170 files, 1969 tests passed.

## Ownership map

Paths are relative to `frontend/src/pages`.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `databases/ui/verification/VerificationsComponent.test.tsx` | Same file | Following `nextCursor`, the after-backup trigger, failing polls and a failed refresh after a cancel |
| `instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx` | Same file | The token revealed once after a creation, the reported memory and no late completion after a session replacement |

## Tests changed beyond their imports

- `databases/ui/verification/VerificationsComponent.test.tsx`: `shows no alert and refreshes again when the refresh after a successful cancel fails` presses the named cancel button instead of the icon image, and `shows no alert, keeps the rows and requests again when polls fail` also asserts that no message appears.
- `instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx`: `suppresses old agent completion after session replacement` is now `suppresses old agent completion after the screen unmounts for a new session`. Its poll cases are gone, as design decision 4 explains, and its delete cases stay.

New scenarios: the new groups of both files above, `databases/ui/verification/VerificationDetailDrawer.test.tsx` and `frontend/browser/scenarios/verificationScreens.spec.ts`.

## Guards

These new tests already held on the unchanged code, once the old agent screen had named its icon buttons, so they pin behavior this change must keep rather than a defect:

- `instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx`: `sends no request after an unmount and ignores an answer that arrives after it`, `keeps the agent and reports once when the deletion is refused`, `keeps the entered name and reports once when the creation is refused`, `keeps the dialog, reports once and reveals no token when the rotation is refused`, both cases of `removes the token from $origin from the page and browser storage on close, and later install commands show the placeholder`, and the three tests of `its icon actions`.

The new tests of the verification list could not separate guards in this way, because the old list's actions were not buttons, so all of them failed on the unchanged code. Removing the session guards in `cancelVerification`, the drawer's request revision, the reload after a deletion or `forceRender` on the token dialog each made tests fail.

## Findings during the work

- `VerificationTableStat` has no `id`, while the details table used `rowKey="id"`, so every row had the same missing key. The key is now the schema and the table name, and `gives each table its own row key when two schemas hold a table of the same name` pins it.
- AntD's colored `Tag` presets failed the browser contrast check in the details drawer, with 2.95 for `error` and 3.57 for `processing` in the light theme, and 3.88 and 3.25 in the dark one. The drawer now shows the status through `VerificationStatusLabel`, as the list does, the agent table shows its status as text in a semantic role, and the trigger tag is neutral, so its word alone tells a manual verification from an automatic one.
- rc-dialog keeps a closing modal's last content until its animation ends. The token dialog stays mounted so its close can return the focus, so it sets `forceRender`, which removes the token at once.
- In the dark theme, the focused sortable header of the details table has a contrast of 4.27. This is AntD's focus style, the drawer's accessibility check runs before the focus reaches it, and it stays as it is.
- `CopyableCodeBlock` and `VerificationTriggerTag` are units beyond those the tasks list, because the token dialog and `InstallCommands` share the first and the list, the cards and the drawer share the second. `BackupActionButton` is now `RowActionButton`, because the verification actions use it too.

## Final

Final `pnpm --dir frontend test` on 2026-11-02: 171 files, 2003 tests passed. `pnpm --dir frontend test:browser --project=ui`: 107 passed.
