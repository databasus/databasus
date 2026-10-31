## 1. Reproduce the defects

- [x] 1.1 Add screen tests to `frontend/src/features/restores/ui/RestoresComponent.test.tsx` for a PostgreSQL source with an SSH tunnel, a client certificate, a non-default port and a database name: the target form opens with the engine's default port, an empty host, user, password and database name, a disabled tunnel and no stored certificate or SSH credentials; an empty password leaves the start button disabled and sends no start request; the flow sends no `/database-tests` request, no request carries the source database's ID, and the start request holds only entered values and defaults. Run `pnpm --dir frontend test src/features/restores` and confirm the new tests fail for these reasons on the unfixed code.

## 2. Start the target from a new database

- [x] 2.1 Add `createRestoreTargetDraft` in `frontend/src/features/restores/ui/createRestoreTargetDraft.ts`, next to `toRestoreTarget.ts`: an empty ID, the source's workspace and the default settings of the backup's engine. Use it as the dialog's initial target and delete `createInitialEditingDatabase` with its comment. Verify with `pnpm --dir frontend test src/features/restores`, where the form-content test from 1.1 now passes.

## 3. Start the restore without a separate test

- [x] 3.1 Forward `isRestoreMode` from `EditDatabaseSpecificDataComponent` to the MySQL, MariaDB and MongoDB forms. In restore mode, make each of the four logical forms render the submit button directly, enabled when its required fields are filled and without calling the settings test. Verify with `pnpm --dir frontend test src/features/restores src/features/databases/ui/edit`, where the empty-password and no-test-request tests from 1.1 pass and the existing form tests still pass.
- [x] 3.2 Let `onSaved` return a promise. In the `isSaveToApi={false}` path, make each logical form hold its saving state until the callback settles and ignore a submission while one is pending. Make the dispatcher stop replacing its draft before it hands such a submission over, and return the start promise from the restore dialog. Add screen tests: two clicks during a pending start send exactly one start request; a refused start shows the server's reason, keeps every entered value and adds no entry to the restore list; a corrected second start succeeds and closes the form; for the viewer scenario, a start that answers `202` is reached without any `/database-tests` request or request carrying the source database's ID. Add a screen test of `CreateDatabaseComponent` in `frontend/src/features/databases/ui/` that submits the settings step and checks that the wizard advances to the limited-user step with the submitted settings, because the dispatcher change also reaches the wizard. Verify with `pnpm --dir frontend test src/features/restores src/features/databases`.
- [x] 3.3 Cover the MySQL, MariaDB and MongoDB targets in `RestoresComponent.test.tsx` with one table test that enters a target per engine and checks the start request. Update the existing MySQL expectations that relied on the source's database name and on the test step. Verify with `pnpm --dir frontend test src/features/restores`.

## 4. Checks

- [x] 4.1 Search the frontend and `docs/` for comments and documentation that describe the old target prefill or the test step in restore mode, and bring them in line. Then format only the changed frontend files with the project formatter, run `mise run frontend:lint` and `pnpm --dir frontend build`, and confirm both pass without new FSD baseline entries.

## Change verification

```sh
pnpm --dir frontend test src/features/restores src/features/databases
```
