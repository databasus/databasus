## 1. Backend aggregates on owning features

- [x] 1.1 Add `GetTotalsByDatabaseIDs` and `GetInstallationTotals` to the logical backup repository, with wrappers on `LogicalBackupService`; verify `go vet ./internal/features/backups/...`
- [x] 1.2 Add `GetBackupTotalsByDatabaseIDs` and `GetInstallationBackupTotals` to the core physical backup service; verify `go vet ./internal/features/backups/...`
- [x] 1.3 Add `FindByDatabaseIDs` and `GetBackupConfigsByDatabaseIDs` to both backup config packages; verify `go vet ./internal/features/backups/...`
- [x] 1.4 Add `GetConfigsByDatabaseIDs` to the healthcheck config service and `GetRecentAttemptsByDatabaseIDs` to the healthcheck attempt service; verify `go vet ./internal/features/healthcheck/...`
- [x] 1.5 Add `CountDatabases` to the database service; verify `go vet ./internal/features/databases/...`

## 2. Backend dashboard feature

- [x] 2.1 Create the `dashboard` package with DTOs, service, controller, DI and route registration in `cmd/main.go`; verify `go build ./...`
- [x] 2.2 Move the healthcheck attempt fixture to `healthcheck/attempt/testing.go` and add `EnableHealthcheckForTestDatabase` through the API; verify `go vet ./internal/features/healthcheck/...`
- [ ] 2.3 Write the dashboard controller tests; verify `go test ./internal/features/dashboard/... -count=1`
- [x] 2.4 Regenerate the API docs; verify `make swagger` lists `/dashboard` and `/dashboard/installation`
- [x] 2.5 Run the linter; verify `make lint`

## 3. Frontend

- [x] 3.1 Extract `formatSizeMb` with its test, `HealthStatusBadgeComponent` and `HealthcheckAttemptsStripComponent`, and switch the existing components to them; verify `pnpm test`
- [x] 3.2 Add the `features/dashboard` slice with types, API client and page; verify `pnpm build`
- [x] 3.3 Add the sidebar entry, its icons and the default tab; verify the app opens on the dashboard after sign-in
- [x] 3.4 Add the `dashboard` keys to all six dictionaries; verify `pnpm build` and `pnpm test`
- [x] 3.5 Format and lint; verify `pnpm format` and `pnpm lint`

## 4. Review

- [ ] 4.1 Run the post-implementation compliance review from the root `AGENTS.md` and resolve its findings
