## Why

Milestone 12d of the backend rewrite plan moves verification into its own module. That move is too large for one review, so it is split in two, as milestone 10c was. This change, the first half, changes only the schema and what the schema forces on other code. The second half, `verification-module`, moves the code onto a schema that is already in place and already checked by the upgrade test.

The four verification tables break the data rules of `docs/architecture.md`:

- **Names.** `backup_verification_configs`, `restore_verifications` and `restore_verification_table_stats` carry no module prefix. Only `verification_agents` does.
- **References to other modules are `CASCADE`.** A config and a verification reference `databases`, and a verification references `backups_logical`, all with `ON DELETE CASCADE` (`backend/migrations/20260520120000_add_restore_verification.sql:69-112`). A reference from one module's table to another's must be `RESTRICT`, so that a missed cleanup fails loudly instead of passing silently.
- **Enum columns have no `CHECK`.** A verification's status and trigger, a config's schedule type and interval type accept any text. The notification choice of a config is a comma-separated string that the code splits on every read (`backend/internal/features/verification/config/model.go:35-63`), and nothing in the schema limits its values.
- **The upgrade test does not cover verification.** A migration of these tables would ship unchecked against data of a real release.

`RESTRICT` to `backups_logical` cannot work on its own. The `backups` module deletes logical backups along four paths: a user's deletion, retention, a storage change and a database deletion. None of them knows that verifications exist, and `backups` cannot import the higher `verification` code to ask. Each path would start failing with a foreign key error on the first backup that has a verification.

## What Changes

- **Table names.** One migration renames:
  - `backup_verification_configs` to `verification_configs`;
  - `restore_verifications` to `verification_runs`;
  - `restore_verification_table_stats` to `verification_run_table_stats`, whose column `restore_verification_id` becomes `run_id`.

  Primary keys, foreign keys and indexes of all four verification tables take the module's naming convention. The unique index on agent tokens becomes `uq_verification_agents_token_hash`. A new index on the backup of a verification serves the deletions of the next bullets.
- **References to `databases` and `backups_logical` become `RESTRICT`.** The references inside the module stay as they are: table statistics cascade with their verification, and a verification's agent is cleared when the agent row goes.
- **`backups` publishes `LogicalBackupsDeleting`.** Every path that deletes logical backups publishes the event inside its transaction, after it has locked the backup rows and before it deletes them. A handler that fails rolls the whole deletion back. The database-deletion and storage-change paths stop deleting by database and delete the backup IDs they locked, so all four paths go through one deletion.
- **The old verification code reacts to the deletions.** Until `verification-module` moves it, the old runs code deletes the verifications of the deleted backups on `LogicalBackupsDeleting`, and the old config code deletes a database's config on `DatabaseDeleting`. A verification that an agent is running when its backup is deleted goes away as it does today, and the agent's next call for it gets `410` as today.
- **`CHECK` constraints** cover a verification's status and trigger, a config's schedule type and interval type, and the notification choice. The notification choice becomes a `TEXT[]` column with a `CHECK` that admits only the two verification notifications, as the notifier categories already do.
- **Rows that a new constraint would reject are repaired first.** An unknown verification status fails the verification with a message that keeps the stored value. An unknown trigger becomes manual. An unknown schedule type or interval type takes the column's default, and a repaired interval gets the time and weekday of the default config where they are empty. Unknown notification values are dropped from the list.
- **The old verification code reads and writes the renamed tables.** The model table names, one raw join, the statistics column and the notification column switch to the new schema. The API's request and response shapes stay as they are.
- **The upgrade test covers verification.** The previous release creates a verification agent, saves a verification config for the logical database and starts a manual verification of its backup. After the upgrade, the current code lists the agent, returns the config and returns the verification.
- **BREAKING** for the schema: three tables and their constraints and indexes are renamed, one column is renamed and one changes its type, the references to other modules change, `CHECK` constraints are added and invalid rows are repaired. A user's own SQL against the old names stops working. The API, the frontend, the agent protocol and every stored verification stay as they are.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-verification`: a new requirement states that every deletion of a logical backup also removes its verifications, and that a refused deletion keeps them.
- `upgrade-test`: the previous release also writes a verification agent, a verification config and a verification, and after the upgrade the current code returns them.

## Out of scope

- Moving the verification code into `modules/verification`, the 11 user operations in `openapi.yaml`, the frontend on the generated client, the scheduler on `background.Runner`, the removal of `features/intervals`, `features/backups/config` and the bridges, and the move of the verification tests. That is `verification-module`.
- Hard deletion of agents. The `deleted_at` column of `verification_agents` stays, because the reaper finds the runs of a removed agent through the agent row (`backend/internal/features/verification/runs/scheduler.go:183-211`).
- The memory unit of the heartbeat and the `max_ram_gb` column, which `agent-contract` changes.
- Refusing to delete a backup while a verification reads it. Today's behavior stays: the verification goes with the backup.
- Changing the order of the handlers of a database deletion, or what a deletion refuses.

## Impact

- **Migrations:** one new migration in `backend/migrations` with its test, which applies the migrations before it to rows with unknown values and checks every repair and every constraint.
- **Backend, changed:**
  - `internal/modules/backups`: the `LogicalBackupsDeleting` event and its subscription in `module.go`, and its publication in `internal/logical` for the four deletion paths;
  - `internal/features/verification`: the models, the join in `runs/repository.go`, the notification column of the config, the handlers of `LogicalBackupsDeleting` and `DatabaseDeleting`, and their subscriptions for the test build;
  - `internal/app/wiring.go`: the two new subscriptions;
  - `internal/features/workspaces/testing/testing.go`: the test workspace removal deletes the verification rows before the databases;
  - `backend/e2e/upgrade`: the verification data before and after the upgrade.
- **Tests:** a scenario per deletion path in a new API test file of `internal/modules/backups`, and the two old tests that checked the cascades are removed.
- **Not affected:** the contract, the frontend, the agent and its protocol, and the old verification routes.
- **Running instances:** the migration renames the tables in place, converts the notification column, changes the references and adds the constraints after the repairs. Agents of the previous release keep working across the upgrade.
- **Documents:**
  - `docs/architecture.md` does not change. The change follows its data rules: the module prefix, `RESTRICT` between modules and `CHECK` on enum columns;
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the 12d line is split into 12d1 and 12d2, and 12d1 is checked off.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
