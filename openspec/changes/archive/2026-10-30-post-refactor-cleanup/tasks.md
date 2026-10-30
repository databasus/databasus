## 1. Remove the unused middleware

- [x] 1.1 Confirm runtime references with gopls, remove GetSessionMiddleware from the users module interface, forwarding implementation and access adapter, and rewrite the stale access comments. Preserve every contract access mark and GetAccessMiddleware. Verify with `mise run backend:test-package ./internal/modules/users ./internal/app/...` and `mise run backend:lint`.

## 2. Reconcile comments and completion records

- [x] 2.1 Confirm that users/internal/signin/session.go has no deleted-helper rationale; remove stale helper and verification comments in databases/module.go, backups/module.go and backups/internal/logical/logical_backup.go. Search migrated modules for equivalent transitional claims, including database deletion cascades and workspace test routes. Preserve methods with current runtime callers. Verify with `mise run backend:test-package ./internal/modules/users ./internal/modules/users/internal/signin ./internal/modules/databases ./internal/modules/backups ./internal/modules/backups/internal/logical ./internal/modules/databases/internal/userdatabase ./internal/modules/workspaces` and `mise run backend:lint`.
- [x] 2.2 Check archived legacy-teardown tasks and mark only milestones 13 and 13c complete in temp/refactor/plan.md. Verify with `python3 -c 'from pathlib import Path; plan = Path("temp/refactor/plan.md").read_text(); tasks = Path("openspec/changes/archive/2026-10-30-legacy-teardown/tasks.md").read_text(); assert "- [ ]" not in tasks; assert "- [x] **13." in plan; assert "- [x] **13c." in plan'`.
- [x] 2.3 Qualify the architecture's production test-dependency statement with the exact existing rclone exceptions and their source of truth, keeping repository test support out of both binaries. Do not modify the allowlist or provider registry. Verify with `mise run backend:test-package ./tools/archlint` and `mise run backend:lint`.

## 3. Complete verification

- [x] 3.1 Run `mise run backend:format`, `mise run backend:lint` and all Change verification commands; search live documents for contradictory helper/middleware/dependency claims, excluding archives and historical analysis. Resolve mandatory review findings and leave the change ready to commit.

## Change verification

```sh
mise run backend:test-package ./internal/modules/users ./internal/modules/users/internal/signin ./internal/modules/databases ./internal/modules/backups ./internal/modules/backups/internal/logical ./internal/app/... ./tools/archlint ./internal/modules/users/internal/module ./internal/modules/users/internal/httpapi ./internal/modules/databases/internal/userdatabase ./internal/modules/workspaces
```
