## 1. Protect existing installations before migration

- [x] 1.1 Add the read-only storages upgrade check, expose it through `module.go` and call it from shared application initialization before migrations. Cover unused storages, another workspace, missing provider settings, the legacy and current layouts, fresh databases, restarts after removal and console initialization. Assert the exact 3.67.0 recovery guidance, unchanged records and migration history, and credential-free diagnostics. Verify with `mise run test`.
- [x] 1.2 Create the transactional removal migration with goose. Recheck for Google Drive parent rows, refuse before changing anything, and otherwise drop the obsolete settings table without cascading or deleting parent records or files. Provide the empty-schema Down definition. Add migration tests for refusal, unchanged encrypted values and history, successful removal, fresh installation and older dumps; limit historical Drive-preservation fixtures to the earlier migration they exercise while keeping full-upgrade coverage for supported secrets. Verify this migration boundary with `go -C backend test -count=1 ./migrations` and `mise run backend:lint`. Run the full suite only after the provider, dependent tests and frontend consumers are removed in sections 2 and 3, as required by Change verification.

## 2. Remove the API and backend provider

- [x] 2.1 Remove the Google Drive settings schema and discriminator from `backend/api/openapi.yaml`, run `mise run generate-api`, and delete the backend provider, registration, type, settings conversion, provider-only fixtures and tests. Remove the built-in provider's imports and run `go mod tidy`, retaining `google.golang.org/api` as an indirect rclone dependency and preserving account OAuth dependencies. Keep only the retirement check and historical migration support. Verify contract generation, architecture and compilation with `mise run backend:lint`.
- [x] 2.2 Add API rejection coverage for authenticated creation, update and unsaved connection testing with the removed discriminator; assert `400 invalid_request`, unchanged records and no provider calls. Preserve existing permissions, supported-provider secret and proxy coverage, and account Google sign-in tests. Use raw requests for the removed contract variant. Verify the backend surface with `go -C backend test -count=1 ./internal/modules/storages/... ./internal/modules/users/...`; run the full backend and frontend suite after section 3 through Change verification.

## 3. Remove storage authorization and frontend presentation

- [x] 3.1 Delete the Google Drive editor and detail presentation, storage callback route and page, authorization hook, attempt, relay and their unused exports and helpers. Remove provider selection, defaults, validation branches, logo and website-link entries. Remove obsolete copy from every interface dictionary together and update affected screen and App tests. Cover absent options in storage management and backup configuration, ordinary unknown-route handling for a late storage callback with no code exchange or mutation, and preserved account OAuth. Verify with `pnpm --dir frontend test`, `mise run frontend:lint` and `pnpm --dir frontend build`.
- [x] 3.2 Remove the retired authorization browser scenarios and their exclusive references, and update shared provider fixtures and storage resource scenarios for the remaining types, labels, fields and layout. Preserve creation, editing and usage checks for supported storages. Verify with `pnpm --dir frontend test:browser --project=ui`.

## 4. Remove public documentation in every language

- [x] 4.1 Delete the Google Drive setup page, localized dispatcher and six content copies, old redirect and screenshots. Update navigation, translated routes, sitemap, `llms.txt`, structured data, metadata, remaining links and provider counts. Remove mentions throughout affected English pages and their six translations. Verify with `pnpm --dir website lint`, `pnpm --dir website build`, and `pnpm --dir frontend test src/shared/i18n/websitePages.test.ts`; audit the static output to confirm retired pages are absent and remaining localized routes are emitted.
- [x] 4.2 Remove Drive mentions from the root README and all six translations, delete the obsolete authorization document, remove its architecture section and glossary entry, and update the glossary's storage examples. Keep current documentation, including `docs/architecture.md` and `docs/testing.md`, free of explanations about historical Google Drive support, its removal or upgrade recovery. Retain the pre-migration refusal, recovery diagnostic and boundary between refusal rollback and a completed one-way upgrade in runtime behavior, tests and behavioral specifications. Remove the dead documentation link and brand-label example from the root and website `AGENTS.md` files without adding conventions. Verify with `mise run lint` and a repository-wide `rg -n -i 'google[ _-]?drive|gdrive|google-oauth|StorageAuthorizationAttempt|storageAuthorizationRelay'` audit; remaining matches must be limited to historical migrations and focused tests, retirement checks and tests, and OpenSpec history or retirement requirements.

## 5. Prove upgrade refusal and recovery with the released image

- [x] 5.1 Update the general previous-release fixture to seed only currently supported storage types while preserving all remaining storage, notifier, database, backup, restore and agent checks. Verify with `mise run backend:test-upgrade`.
- [x] 5.2 Add the regression anchored to the actual 3.67.0 image and pin its resolved release identity. Through its API create an account, workspace, unused Google Drive storage with synthetic credentials and supported storage. Stop the old instance, snapshot persisted records and history read-only, observe candidate refusal and non-zero exit, and compare unchanged snapshots without exposing credentials. Restart 3.67.0 on the same volume, sign in, check records, delete the unused Drive storage through its API, and retry the candidate successfully. Confirm preserved supported records and no obsolete settings table; retain unique resources, labels and unconditional cleanup. Verify with `mise run backend:test-upgrade`.

## 6. Complete cross-area verification

- [x] 6.1 Run relevant formatters for the changed implementation files, then `mise run lint`, `pnpm --dir frontend build` and `pnpm --dir website build`. Execute the Change verification commands below and the upgrade command from section 5 against the final candidate. Audit source and static output for remaining public Drive claims and dead links. Resolve blocking automatic implementation review findings before marking the change complete; verify the artifacts with `openspec validate remove-google-drive --strict`.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
