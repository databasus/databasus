## Why

Frontend milestone 8 can be accepted only when each risky behavior listed in the analysis has a test that owns it. An audit of that matrix found 8 behaviors with no test and 17 where the test covers only a helper or one branch, plus one case whose setup does not test what its name says. Storage and notifier secrets are among them: NAS, Azure, FTP, rclone, Telegram, Discord, Teams and Mattermost have no test that omitted secrets stay omitted, which is how a refactoring could send or lose a credential unnoticed.

The owner decided to close all of them with tests rather than record reasons for skipping the less risky ones. This is also the last change of the migration, so it runs the final acceptance.

## What Changes

- Add tests for every open item of the behavior matrix, as `design.md` lists them.
- Repair one known defect: logout keeps the selected database, storage and notifier of each workspace and the backup list filters, so the next account opens the previous account's selection and filters, which can hide recent backups. The `frontend-session` spec already forbids this.
- Finish the audit: read the test bodies for the matrix rows checked partly by title, and close whatever else turns up.
- Trace test titles that disappeared during the migration, and record each one's new place or its reason in `coverage.md`.
- Bring `frontend/README.md` in line with the final structure, and search the repository for leftover mentions.
- Run the complete acceptance set and record the suites and their durations.

- Repair a defect the new tests exposed: repeated use of a logical backup's verify action opened several confirmations, and confirming them started several checks.

Beyond these defects, no production behavior changes on purpose. A test that exposes another defect reproduces it first, and the defect is fixed in this change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: the verify action of a logical backup starts one check per confirmation. The main specs did not require this, so it has a delta spec. The other tests pin behavior the specs already require, and the logout fix stays within `frontend-session`.

## Impact

This is the last of three changes of frontend milestone 8, and it depends on `remove-frontend-check-exceptions`. The root `AGENTS.md` and `frontend/AGENTS.md` apply. Tests follow the testing skill.

Affected areas:

- test files under `frontend/src`, and scenarios under `frontend/browser`;
- `frontend/src/shared/auth/session.ts`, `frontend/src/pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx`, and other production code under `frontend/src` only where a test exposes a defect;
- `frontend/README.md`, and any document the final search finds out of date;
- `temp/refactor/frontend-plan.md`, a working document that is not committed.

There is no planned backend, HTTP contract, generated code or dependency change.

### Out of Scope

- A coverage percentage target. The matrix sets the bar.
- Multiplying matrix rows by every language and screen size.
- Adding a real API scenario for databases, because the harness offers no source database. Backend API tests own that contract.
- New browser reference images unless a scenario cannot assert the behavior otherwise.
- The full backend suite, pushing and deploying.
