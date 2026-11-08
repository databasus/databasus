## Why

The local seed can leave `test@email.com` as an ordinary member when another account registered first. Developers then cannot exercise administrator screens with the documented dev account, even after restarting `mise run run`.

## What Changes

- After successful dev-account sign-in or registration, make that account the sole bootstrap administrator before seeding its workspace and databases.
- Preserve the previous bootstrap administrator's `ADMIN` role, password and settings; transfer only its bootstrap designation.
- Perform the transfer atomically through a local, development-only operation owned by the users module. Record actual changes in the audit log and leave an already-correct assignment unchanged.
- Coordinate existing user-administration checks and writes with the transfer so an overlapping request cannot demote or deactivate the new bootstrap administrator or retain the previous holder's bootstrap powers.
- Preserve existing health, password, registration and two-factor refusals. A failed authentication never transfers privileges.
- Update the seed documentation and requirements that currently prohibit all changes to existing account roles.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: the dev seed ensures bootstrap-administrator privileges as well as the existing dev items, with explicit development and authentication boundaries.
- `root-admin-account`: recognize development seeding as a narrow exception to the otherwise persistent bootstrap designation and administrator-management restrictions.

## Impact

- Backend: the dev seed, a lightweight composition entry point in `internal/app`, a dedicated users-module service and repository operation exposed through `module.go`, and transaction locking in existing user administration.
- Tests: seed integration scenarios and observable administrator permissions, using real application instances and isolated internal databases.
- Documentation: the seed package documentation and any affected descriptions in the backend README.
- The public HTTP contracts, database schema, production startup and production console commands remain unchanged. The seed additionally needs the checkout's existing `DATABASE_DSN`; no new environment variable is introduced.
- This change follows root `AGENTS.md` and `backend/AGENTS.md`.

### Out of Scope

- Renaming the dev account to `test@example.com` or changing its password.
- Demoting the previous bootstrap administrator to `MEMBER`, resetting credentials, activating users or disabling two-factor authentication.
- Adding a public privilege-transfer endpoint, a general console transfer command or a production seeding path.
- Changing existing workspace, storage, database or backup settings, or editing unrelated frontend changes.
