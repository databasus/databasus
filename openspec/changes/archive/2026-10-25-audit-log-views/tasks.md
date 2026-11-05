## 1. Reading events in `lib/audit`

- [x] 1.1 Add `EventListQuery`, `RecordedEvent`, `EventPage` and `Log.ListEvents` with the ordering and the cursor of design.md decision 1, update the comment on `Log`, and add the migration of decision 4. Write the `lib/audit` tests of decision 7. Verify with `mise run backend:test-package ./internal/lib/audit/... ./internal/lib/database/...`.

## 2. The operation

- [x] 2.1 Add `listAuditEvents` and the schemas `AuditEventPage` and `AuditEvent` to `backend/api/openapi.yaml` under the tag `workspaces` as design.md decision 2 describes, with the order and the access rules in the operation's description, run `mise run generate-api`, and serve it from the module with the rules of decision 2 and the name batch of decision 3. Write API tests for every scenario of the `audit-log` delta. Verify with `mise run backend:test-package ./internal/modules/workspaces/...` and `mise run backend:lint`.
- [x] 2.2 Delete the read code of `features/audit_logs` and its three routes with their tests per design.md decision 5, keeping the write bridge. Switch the tests that read events through the old routes, `internal/modules/users/helpers_test.go` and `internal/app/audit_events_cleanup_test.go`, to `listAuditEvents`. Verify with `mise run backend:test-package ./internal/features/audit_logs/... ./internal/modules/users/... ./internal/app/...` and `rg -n 'audit-logs|GetAuditLogsRequest|AuditLogRepository|CountGlobal' backend/internal` printing nothing.
- [x] 2.3 Switch the audit checks of the current release in `backend/e2e/upgrade` to `listAuditEvents`, following `nextCursor` to the last page per design.md decision 7, and leave `previous_release_client_test.go`, which speaks the previous release's API. Verify with `mise run backend:test-upgrade`.

## 3. The frontend

- [x] 3.1 Rewrite `entity/audit-logs/api` on the generated client and delete its hand-written models. Update the three screens of design.md decision 6 to load more by `nextCursor` and show the nested author and workspace, delete `auditLogs.loadedOfTotal` and drop the count from `auditLogs.allLoaded` in all six dictionaries. Verify with `pnpm --dir frontend typecheck` and `pnpm --dir frontend lint`.
- [x] 3.2 Write the screen test of design.md decision 6 for the global view on `mockApi`. Verify with `pnpm --dir frontend test src/features/settings`.

## 4. Documents and checks

- [x] 4.1 Update `temp/refactor/plan.md`: check `5c` and the whole of milestone 5, and cross out the two relaxation rows of design.md decision 5. Verify with `rg -n '^\s*- \[x\] \*\*5c\.' temp/refactor/plan.md` and `rg -n '^- \[x\] \*\*5\. ' temp/refactor/plan.md` showing both lines.
- [x] 4.2 Search the code and the documents for the old paths and fields and bring every mention in line. Verify with `rg -n 'api/v1/audit-logs|audit-logs/global|audit-logs/users|"/audit-logs"|\}/audit-logs|GetAuditLogsResponse|loadedOfTotal' backend/internal backend/cmd backend/e2e frontend/src docs AGENTS.md backend/AGENTS.md frontend/AGENTS.md --glob '!**/previous_release_client_test.go'` printing nothing.
- [x] 4.3 Run `mise run lint` and fix what it reports. Verify that it exits with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/lib/audit/... ./internal/lib/database/... ./internal/modules/workspaces/... ./internal/modules/users/... ./internal/features/audit_logs/... ./internal/features/backups/backups/controllers/... ./internal/features/restores/... ./internal/app/...
pnpm --dir frontend test
```
