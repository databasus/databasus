## 1. One definition of a Neon host

- [x] 1.1 Move `isNeonHost` from `backend/internal/engines/postgresql/physical/neon_source.go` to `backend/internal/engines/postgresql/shared` as `IsNeonHost`, with a test of the existing host cases plus `neon.tech.example.com` and an empty host, and make the physical refusal call it; verify with `go -C backend test -count=1 ./internal/engines/postgresql/shared/... ./internal/engines/postgresql/physical/...`
- [x] 1.2 Add `IsNeonHost(settings)` to the database provider interface, implement it in both PostgreSQL providers by calling `pgshared.IsNeonHost` on the configured database host and as `false` in MySQL, MariaDB and MongoDB, and expose `IsNeonHost(ctx, databaseID)` in `backend/internal/modules/databases/module.go`; verify with `go -C backend build ./...` and `mise run backend:lint`

## 2. Healthcheck default and confirmation flag

- [x] 2.1 Add a helper to the healthcheck test fixture that starts a network, an SSH bastion and PostgreSQL with a Neon network alias such as `ep-test-123456.us-east-2.aws.neon.tech` through the exported `StartNetwork`, `StartSSHBastionContainerOnNetwork` and `StartPostgresOnNetwork`, and creates a logical database at that alias through the bastion. Make `OnDatabaseCreated` in the healthcheck module save the default with checks switched off when `IsNeonHost` answers true, and add API tests for a new Neon database, a Neon pooler host, a host that only contains the Neon name, no attempt added for a new Neon database, and a copy of a Neon database whose checks were switched on; verify with `go -C backend test -count=1 -run 'TestHealthcheckConfig' ./internal/modules/healthcheck/...`
- [x] 2.2 Add the required read-only boolean `shouldConfirmEnabling` to `HealthcheckConfig` in `backend/api/openapi.yaml`, run `mise run generate-api`, and fill it from `IsNeonHost` in the read and save answers; add API tests for the flag on a Neon and another database, a save that switches the checks on for a Neon database followed by a due-check search that adds an attempt for it, and a database whose host is edited to Neon; verify with `go -C backend test -count=1 ./internal/modules/healthcheck/...` and `mise run backend:lint`

## 3. Delayed confirmation in the editor

- [x] 3.1 Add an optional `actionDelaySeconds` prop to `frontend/src/shared/ui/ConfirmationComponent.tsx` that disables the confirm button and shows the seconds left in its label, with the label key in all seven dictionaries under `frontend/src/shared/i18n/locales/`, and extend `ConfirmationComponent.test.tsx` with fake timers for the locked and unlocked button; verify with `pnpm --dir frontend test src/shared/ui/ConfirmationComponent.test.tsx`
- [x] 3.2 Make `EditHealthcheckConfigComponent` open the confirmation with a 10-second delay when `shouldConfirmEnabling` is true and the switch goes from off to on, with the explanation of Neon compute billing in all seven dictionaries; drop `shouldConfirmEnabling` from the save body in `healthcheckConfigApi.ts`; extend `EditHealthcheckConfigComponent.test.tsx` for the countdown, confirming after the wait, cancelling, closing the dialog with its close control and with Escape (the switch stays off and the save button keeps its earlier state), switching off without a dialog, a database without the flag, and a save body without the flag; verify with `pnpm --dir frontend test src/pages/databases/healthcheck` and `mise run frontend:lint`
- [x] 3.3 Check that the frontend builds with the regenerated types; verify with `pnpm --dir frontend build` and `pnpm --dir frontend typecheck`

## 4. Documentation

- [x] 4.1 Add a "Neon host" entry to `docs/glossary.md`: the host rule that decides the refusal of physical backups and the healthcheck default and confirmation, judged on the configured database host rather than the SSH tunnel's address; verify with `rg -n 'Neon host' docs/glossary.md`

## Change verification

```sh
mise run test
```
