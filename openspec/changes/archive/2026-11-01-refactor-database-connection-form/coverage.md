# Database connection form refactoring coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-01, before this change: 133 files, 1305 tests passed.

## Ownership map

Paths are relative to `frontend/src`.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `features/database-connection/ui/EditDatabaseSpecificDataComponent.test.tsx` | `features/database-connection/ui/DatabaseConnectionForm.test.tsx` | No password is sent in a test of a saved database whose password stays empty; a failed physical test shows its classified reason and the server error |
| `features/database-connection/ui/EditDatabaseSpecificDataComponent.test.tsx` | `pages/databases/ui/DatabaseConfigComponent.test.tsx`, `entities/databases/api/databaseApi.test.ts` | No password is sent when a saved database is saved with it left empty |
| `features/database-connection/ui/EditMySqlSpecificDataComponent.test.tsx` | `features/database-connection/ui/EditMySqlCompatibleConnectionComponent.test.tsx` | The table exclusion label that has no data-only counterpart stays |
| `features/database-connection/ui/EditPostgreSqlLogicalSpecificDataComponent.test.tsx` | `features/database-connection/ui/EditPostgresqlLogicalConnectionComponent.test.tsx` | Advanced settings open with both exclusion fields when only table data is excluded, and stay collapsed when no advanced value is set |
| `features/restores/ui/RestoresComponent.test.tsx` | `pages/databases/ui/RestoreTargetComposition.test.tsx` for every scenario that needs the real form; a new feature file covers what the feature does on its own | Restore target entered without source settings, refusals keep input, a corrected retry, one start, an empty password keeps the start disabled, no request names the source database |
| `pages/databases/ui/CreateDatabaseComponent.test.tsx` | Unchanged | The wizard advances with the submitted connection settings, keeps input when going back, starts over when the type changes at Continue, and creates once |
| `pages/databases/ui/DatabaseConfigComponent.test.tsx` | Unchanged | Limited user hint after a save, none for a viewer, late configuration answers ignored |
| `pages/databases/ui/DatabaseComponent.test.tsx` | Unchanged | The connection section keeps unsaved edits across a language change and closes on cancel without a new request |
| `pages/databases/ui/DatabasesComponent.test.tsx` | Unchanged | A saved connection keeps the settings tab and the list, and a refused save keeps the section open with the entered values |

Owner note for `pages/databases/ui/RestoreTargetComposition.test.tsx`, "shows nothing when a start answers after the dialog closed": closing the real dialog unmounts `RestoresComponent` and the form, so a late answer has nothing left to change on the screen, and the test passes with or without the `isActive()` checks. It pins that the dialog stays closed and no alert appears. The check in `startRestore` is pinned by "shows nothing when a start answers after the session changed", where the form stays mounted. The check on a refused submission in `DatabaseConnectionForm` is pinned by "an answer after the session changed > shows nothing when the save was refused" in `pages/databases/ui/DatabaseConfigComponent.test.tsx`.

## Tests that failed before the change

Run on the unchanged forms on 2026-11-01. Each failed for the reason its name describes.

| Level | Failing test | Failure on the old form |
| --- | --- | --- |
| Wizard, each of five variants | keeps the connection input after going back from the connection step | The host came back empty |
| Wizard | does not offer Continue when the password changes during a successful test | Continue appeared after the late answer |
| Wizard | keeps the chosen SSL mode and certificates after going back and entering localhost | The mode became `Disable` |
| Wizard | shows a typed certificate as editable text after going back | The typed value was not shown |
| Wizard | keeps the input when another engine is chosen and the original one is chosen again | The host came back empty |
| Wizard | keeps the physical type and input when the highlighted PostgreSQL engine is clicked | The host came back empty |
| Wizard | shows a test that cannot reach the server inline and keeps every value | The reason went to a browser dialog |
| Settings | does not offer Save when the password changes during a successful test | Save appeared after the late answer |
| Settings | shows no failure of a test whose settings changed while it was pending | A browser dialog showed the late failure |
| Settings | reports the saved connection when the limited user suggestion is closed | The page never received the saved database |
| Settings | asks for a new test when the user goes back from the limited user suggestion | The page never received the saved database |
| Settings | closes the section with the saved values and the reason when the check after a save fails | The section stayed open |
| Settings | sends no certificate after SSL is turned off and on and shows no mask | The mask and Replace stayed |
| Settings | removes the failure of a physical test when the SSH host changes | The failure stayed |
| Settings | shows a test that cannot reach the server inline | The reason went to a browser dialog |
| Settings | keeps the section open with every value and the reason when a save is refused | The reason went to a browser dialog |
| Settings | sends one update when Save is clicked twice while pending | The fields stayed enabled during the save |
| Settings | ignores a save answer after the session changed | The check ran and unexpected requests followed |
| Restore, MySQL, MariaDB, MongoDB | offers no exclusion setting a restore ignores | The exclusion fields were offered |
| Restore | shows a typed certificate as typed after reopening a target | The certificate appeared masked as stored |

## Result

Final `pnpm --dir frontend test`: 140 files, 1496 tests passed, against the baseline of 133 files and 1305 tests. No test was dropped: every scenario of the three deleted test files and of the moved restore file has its owner above.
