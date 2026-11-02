## Why

The webhook form of v3.64.0 could save a header row with no name, and that release skipped such a row when it sent a request. The current code sends the row as a header with an empty name, so the request fails. After an upgrade the user stops getting notifications, backup failure alerts included, and sees the cause only in the notifier's last send error and the log. Saving the notifier also fails until the user deletes the row. The migration that causes this has not been released, so the regression can be closed before any release ships it.

## What Changes

- The notifiers migration removes every webhook header row with an empty name. Named headers keep their order and their stored values byte for byte.
- A webhook that had such a row delivers and saves after the upgrade without any action from the user.
- The value in a removed row is deleted with it. No release ever sent that value.
- The backend refuses a webhook header name that is not a valid HTTP header name, such as `X Team` or a single space, with a 400 on create, change and test send. A valid name is an RFC 9110 token. Today such a name is saved, and every send through the webhook then fails.
- The webhook form trims spaces at the start and the end of a header name when the user leaves the field. It shows an error at a row with an empty or invalid name, and keeps save and test send unavailable until the row is fixed or removed.
- **BREAKING**: a script that sends an invalid header name through the API now gets a 400 instead of a saved notifier. Such a webhook could never deliver, so no working delivery is lost. No shim is kept for the old acceptance.

**Out of scope:**

- Repairing invalid non-empty header names already stored in the database. We assume nobody saved such a name, because no delivery could work with it.
- Trimming spaces in the backend. An API caller that sends a name with a space gets a 400.
- A database constraint on header names. The application validation covers every write.
- Tolerating nameless rows when settings are read or sent.
- Repairing databases that already applied the unreleased migration, such as development databases.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `notifiers`: the upgrade requirement gains the removal of nameless webhook header rows, which is the one exception to keeping every stored value. New requirements refuse invalid webhook header names in the API and in the form.

## Impact

- `backend/migrations/20261025180000_move_notifiers_into_module.sql` and its test `backend/migrations/notifiers_migration_test.go`.
- The webhook provider's validation, `backend/internal/modules/notifiers/secrets_test.go` and `backend/internal/modules/notifiers/test_send_test.go`.
- The description of the header name in `backend/api/openapi.yaml`, with the generated code.
- The webhook form, its validation, `EditNotifierComponent.test.tsx` and the six frontend dictionaries.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
