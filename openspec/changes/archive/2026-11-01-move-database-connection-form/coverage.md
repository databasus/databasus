# Database connection form move coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-01, before the move: 133 files, 1305 tests passed.

## Ownership map

Paths are relative to `frontend/src`. The move changes only file paths, import paths and the name of the PostgreSQL parser, so every test keeps its scenarios and assertions.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `features/databases/ui/edit/EditDatabaseSpecificDataComponent.test.tsx` | `features/database-connection/ui/EditDatabaseSpecificDataComponent.test.tsx` | No password is sent when a saved database is saved with it left empty; a failed connection test shows the reason and the server error |
| `features/databases/ui/edit/EditMySqlSpecificDataComponent.test.tsx` | `features/database-connection/ui/EditMySqlSpecificDataComponent.test.tsx` | The table exclusion label that has no data-only counterpart stays |
| `features/databases/ui/edit/EditPostgreSqlLogicalSpecificDataComponent.test.tsx` | `features/database-connection/ui/EditPostgreSqlLogicalSpecificDataComponent.test.tsx` | Advanced settings open with both exclusion fields when only table data is excluded, and stay collapsed when no advanced value is set |
| `entities/databases/model/postgresql/ConnectionStringParser.test.ts` | `entities/databases/model/postgresql/PostgresqlConnectionStringParser.test.ts` | Every URI, JDBC and libpq parsing case, under the new class name |
| `features/restores/ui/RestoresComponent.test.tsx` | Unchanged; the form now comes from the `features/database-connection` public API | Restore target entered without source settings, refusals keep input, one start |
| `pages/databases/ui/CreateDatabaseComponent.test.tsx` | Unchanged; the base-info and backup type forms are now page files and the connection form comes from the new public API | The wizard advances with the submitted connection settings, keeps input when going back, and starts over when the engine or PostgreSQL backup type changes |
| `pages/databases/ui/DatabaseConfigComponent.test.tsx` | Unchanged; the connection form comes from the new public API | Limited user hint after a save, and none for a viewer |
| `pages/databases/ui/DatabaseComponent.test.tsx` | Unchanged | The connection section keeps unsaved edits across a language change and closes on cancel without a new request |
| `pages/databases/ui/DatabasesComponent.test.tsx` | Unchanged | A saved connection keeps the settings tab and the list, and a refused save keeps the section open with the entered values |

## Result

Final `pnpm --dir frontend test`: 133 files, 1305 tests passed, the same counts as the baseline. No test was dropped.
