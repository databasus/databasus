## Context

See `proposal.md` for the motivation. The state that matters:

- v3.64.0 could store a nameless row. Its form added `{ key: '', value: '' }` and its validation checked only the URL. Its `applyHeaders` skipped a row with an empty key when sending.
- The migration copies the headers JSON as it is, at `backend/migrations/20261025180000_move_notifiers_into_module.sql:357-362`. No release tag contains this migration yet; the newest tag is v3.64.0.
- The send path sets every stored header without a name check, at `backend/internal/modules/notifiers/internal/providers/webhook/provider.go:212`. `net/http` then refuses the request, because `validateHeaders` rejects an empty field name.
- A save fails on the name check at `provider.go:62`, because the read returns the nameless row to the form and the form sends it back.
- The check at `provider.go:62` refuses only an empty name. A name like `X Team` is saved, and every send then fails on the same `net/http` check. v3.64.0 behaved the same way.
- The form checks only the URL, at `frontend/src/entities/notifiers/model/webhook/validateWebhookNotifier.ts:3`. The save and test buttons depend on that result through `isNotifierSettingsFilled`, at `frontend/src/features/edit-notifier/ui/edit/EditNotifierComponent.tsx:84`.
- Every write already goes through that check: `notifier/service.go:122` on create, `notifier/service.go:339` on change, and `notifier/test_send_service.go:104` on a test send of unsaved settings. Once the old rows are gone, the application does not try to store a nameless row again.

The approach is constrained by "Migrations" in `docs/architecture.md`, by "No unrequested backward compatibility" in the root `AGENTS.md`, by "Migrations" in `backend/AGENTS.md`, and by "Forms" and "Design and localization" in `frontend/AGENTS.md`. `docs/architecture.md:280` allows a contract break when the same change updates the frontend.

## Goals / Non-Goals

**Goals:**

- Remove nameless rows from stored data and keep them out through validation, so the read and send paths need no special case.
- Refuse every header name the HTTP client would refuse, before it is saved.

**Non-Goals:**

- Repairing stored invalid names. `proposal.md` lists this as out of scope.

## Decisions

### Repair the rows in the migration

The migration removes nameless rows from the JSON. The root `AGENTS.md` forbids fallbacks for an old data shape, and this migration already repairs a webhook method it cannot send, at line 79.

Rejected alternatives:

- **Filter in `toSettings`** (`settings_repository.go:133`). It fixes both sending and saving, but it stays in the code forever as a fallback for an old data shape, which the root `AGENTS.md` forbids. The fallback for encrypted URLs in that function exists because SQL cannot decrypt them. SQL can filter these rows.
- **Skip the row in `buildRequest`.** Delivery comes back, but the save still answers 400 until the user deletes the row.

### Edit the existing migration

The repair goes into `20261025180000_move_notifiers_into_module.sql`, right after the headers column becomes `JSONB NOT NULL`. The migration is unreleased, and commit `11fea906` already corrected it in the same way.

Rejected alternative: a new migration. It would also reach databases that already applied the existing one, but those exist only among unreleased builds. It adds a second migration for one fix, and it would depend on the table that the earlier migration creates and converts.

### Keep empty names out through validation, without a schema constraint

After the repair, every write path goes through `Validate`, so no new nameless row can be stored. The project owner decided against a constraint here.

Rejected alternative: a `CHECK` on `notifiers_webhook` that refuses a header element with an empty `key`. "Invariants live in the schema" in `docs/architecture.md:190` favors it, but `docs/architecture.md:196` names webhook headers as JSON the system treats as opaque, so a `CHECK` on its contents would not follow that rule. It would also guard JSON contents that one validated write path produces, and it would add a statement to the `Down` section for a state the application already refuses.

### Filter with a JSON path expression

The repair uses `jsonb_path_query_array` with a filter on a non-empty `key`. The function keeps the array order, which matters because a later header with the same name replaces an earlier one when a request is built. It copies each kept element whole, so encrypted values stay byte for byte, as `docs/architecture.md:208` requires. The internal database is PostgreSQL 17 (`Dockerfile:109`), and the function exists since PostgreSQL 12.

Rejected alternative: `jsonb_array_elements` with `jsonb_agg`. It needs `WITH ORDINALITY` to keep the order and `COALESCE` for an empty result, and it gains nothing here.

### Pin the behavior with a migration test and one read test

New cases in `TestNotifiersMigration` write rows through the old schema, apply the migration and assert the stored JSON. Existing API tests in the notifiers module already show that a webhook with named headers or with none delivers and saves, so the migration test closes the chain. No existing test reads a webhook without headers, so one new case in `TestGetNotifier` covers that read.

Rejected alternatives:

- **API tests that write a nameless row through SQL.** After the repair no real path reaches that state, so they would test a state the system cannot be in, and they would fail unless a read fallback existed.
- **A nameless row in the upgrade test.** It would exercise the same transformation through a much slower suite.

### Validate the name with an RFC 9110 token rule in the provider

`Validate` in `provider.go` replaces its empty-name check with a regular expression for an RFC 9110 token. Every write path already calls it, so one change covers create, change and test send, and the existing `NewInvalidSettingError` gives the field name in the `400`.

Rejected alternatives:

- **`httpguts.ValidHeaderFieldName`.** It is the exact check `net/http` runs, but `golang.org/x/net` is only an indirect dependency, and making it direct for one line of validation changes `go.mod` for little gain.
- **Trimming spaces in the backend.** It would silently change what an API caller sent, and the name stored would differ from the one in the request. The form trims instead, where the user sees the result.

### Describe the rule in the contract without `pattern`

The `description` of `WebhookHeader.name` in `backend/api/openapi.yaml` states the rule, and the `400` is the existing validation response.

Rejected alternative: `pattern`. The same schema describes responses, so it would promise something the database does not guarantee for a stored invalid name, and the response check in `contracttest` would fail on such a read.

### Check the name in the form as well

`validateWebhookNotifier` gains the same rule, so the buttons disable through the existing path. The form shows one dictionary message under a row with an empty or invalid name. The row reuses the presentation of the email form (`Form.Item` with `validateStatus` and `help`, plus `aria-invalid`), at `frontend/src/features/edit-notifier/ui/edit/notifiers/EditEmailNotifierComponent.tsx:58-72`. It departs from that form in one way: the email form shows an error only after a field is touched (`EditEmailNotifierComponent.tsx:40-54`), and this row has no such gate. A newly added empty row shows the error at once, because a disabled button with no visible reason would be worse.

Rejected alternatives:

- **Leaving the check to the backend.** The user would learn about the name only from a failed save.
- **Dropping empty rows on save.** It would hide a mistake instead of showing it, and the form would send something other than what it displays.

### Trim the name in the form when the field loses focus

The name input trims its value on blur and changes the draft only when the trimmed name differs from the current one, so the draft holds the trimmed name. `changeSettings` in `frontend/src/features/edit-notifier/model/useNotifierDraft.ts:71` always withdraws test approval and cancels a running test, and `EditNotifierComponent.tsx:227` then swaps Save for the test button. An unconditional update would do that when the user only clicks into a name field and leaves it, even though no setting changed. Save and test read the draft, and the dirty state compares it with what the backend returned, so no separate cleaning of the request is needed. The validation ignores spaces at the edges, so a name being typed does not show an error or disable the buttons while the field has focus. Clicking a button moves focus first, so the trimmed name is what gets sent for any name the user edited. A stored row the user never focused stays as it was.

Rejected alternatives:

- **Trimming on every keystroke.** A space typed in the middle would vanish before the next letter, so `X Team` would silently become `XTeam` instead of showing an error.
- **Trimming only in the request.** The draft and the sent settings would differ, and the draft would look unsaved after a successful save.

## Risks / Trade-offs

- [A database that already applied the unreleased migration keeps its nameless rows, so its webhook still fails] → It affects only development databases and unreleased builds. The user can delete the row in the form.
- [A stored name like `X Team` keeps failing on send, and any change of that notifier gets a 400 until the name is fixed] → We assume no such rows exist, because no delivery could work with them. If one exists, the form shows the cause at the row.
- [A stored name with spaces at its edges, such as ` X-Team`, has two bad outcomes. If the user leaves that field, the form trims it, the name no longer matches the stored one, and the value is lost unless the user enters it again. If the user never focuses it, the row shows no error, and the backend answers 400 with only the general error in the form] → Both fall under the same assumption that no such names exist, since no delivery could work with them.
- [A script that sends an invalid name now gets a 400] → The webhook it saved could never deliver, so the script learns about an existing failure earlier.
- [The rule exists twice, in Go and in TypeScript] → The rule comes from the HTTP standard and does not change. A test on each side pins it.
- [The value in a removed row cannot be restored by the `Down` migration] → No release ever sent that value, and `docs/architecture.md:209` makes an upgrade one-way.

## Migration Plan

The repair runs as part of the existing migration at the first start of the new release, inside that migration's transaction. A failure leaves no partial changes, and the next start applies it again.
