## Context

See proposal.md - Why. The current state that shapes the approach:

- The healthcheck default is one constant for every database: `GetDefaultConfig` switches the checks on with a 1-minute interval (`backend/internal/modules/healthcheck/internal/configs/config.go:16-24`). `OnDatabaseCreated` saves it without looking at the database (`backend/internal/modules/healthcheck/internal/configs/service.go:121-123`).
- The databases module publishes `Created` inside the creating transaction after it has saved the provider settings (`backend/internal/modules/databases/internal/userdatabase/service.go:185-195`). The event carries only `DatabaseID` (`backend/internal/modules/databases/internal/userdatabase/events.go:12-14`). A copy publishes `Created` and then `Copied`, and the healthcheck handler of `Copied` overwrites the copy's config with the original's (`service.go:125-132`).
- Neon recognition exists once, privately, in the physical engine: `isNeonHost` trims, lowercases, drops a trailing dot and matches `neon.tech` or a `.neon.tech` suffix (`backend/internal/engines/postgresql/physical/neon_source.go:18-22`). The physical engine uses it to refuse replication before dialing.
- Each database type has a provider behind the registry; code outside the family never switches on the type (`backend/internal/modules/databases/internal/providers/provider.go:19-44`, `backend/AGENTS.md` "Engines and providers"). `GetTelemetryFacts` is the existing example of a provider reading a fact from stored settings without decrypting or connecting.
- Creating or editing a database connects to its host to detect the server (`userdatabase/service.go:172`), so an API test can only create a database whose host really answers.
- The healthcheck editor keeps the loaded `HealthcheckConfig` in state and toggles `isHealthcheckEnabled` directly from the switch (`frontend/src/pages/databases/healthcheck/ui/EditHealthcheckConfigComponent.tsx:101-112`). The save strips `databaseId` and sends the rest as the request body (`frontend/src/entities/healthcheck/api/healthcheckConfigApi.ts:6-13`).
- The shared `ConfirmationComponent` has no way to delay its confirm button (`frontend/src/shared/ui/ConfirmationComponent.tsx`). The master key onboarding page has its own 15-second countdown (`frontend/src/pages/master-key-onboarding/ui/MasterKeyOnboardingPage.tsx:29-48`).

Constraints: `docs/architecture.md` "Modules" (healthcheck sits above databases and reaches it only through `module.go`), "Engines and providers" (no type switch outside the family; engines know no tables), and "HTTP API"; `backend/AGENTS.md` "Where code goes", "Engines and providers" and "Handlers, services, repositories"; `frontend/AGENTS.md` "Architecture and skills" and "Design and localization"; the api-design skill for the contract field.

## Goals / Non-Goals

**Goals:**

- One definition of a Neon host, recorded in `docs/glossary.md`, used by the physical refusal, the default and the confirmation flag.
- The server alone decides both the default and whether a confirmation is needed.
- The scenarios in the delta specs are covered by API tests against a real PostgreSQL server reached under a Neon name, and by component tests of the editor.

**Non-Goals:**

- A general notion of "managed provider" or a provider enum. With one provider and no second one planned, a boolean answers the only question asked.
- Server-side enforcement of the confirmation. The dialog is friction for a person, not an access rule; a script that switches the checks on has made its own choice.

## Decisions

### 1. Neon recognition lives in `engines/postgresql/shared`

`isNeonHost` moves from the physical engine to the shared PostgreSQL engine package as an exported `IsNeonHost(host string) bool`, together with its normalization and the existing host cases of its test. The physical refusal calls the shared function.

Alternatives rejected:

- *A second copy in the databases module.* Two definitions drift: a pooler host or a trailing dot handled in one place and not the other would refuse physical backups for a host whose checks stay on.
- *Keeping it in `physical` and exporting it from there.* The logical provider would import the physical engine for a host rule that has nothing to do with replication.

### 2. The databases module answers `IsNeonHost`, through its providers

`databases.Module` gains `IsNeonHost(ctx, databaseID) (bool, error)`; the module, the provider method and the engine function share the phrase, so a search for the rule finds every layer. The database service loads the stored settings and asks the database's provider through a new provider method, `IsNeonHost(settings) (bool, error)`. Both PostgreSQL providers answer with `pgshared.IsNeonHost` on the configured host. MySQL, MariaDB and MongoDB answer `false`. For physical PostgreSQL the answer is a safeguard only: the physical engine refuses a Neon host when its tunnel opens (`backend/internal/engines/postgresql/physical/tunnel.go:28`), and creation and edit open it while detecting, so a physical database never stores a Neon host. The host is not a secret, so nothing is decrypted, and nothing connects.

The method judges the configured host, not the tunnel's address, for the same reason as the physical refusal: the host names the server the user pays for, wherever the tunnel runs.

Alternatives rejected:

- *Put a Neon flag into the `Created` event.* The default would need no extra read, but the confirmation flag must also follow the current host on every read, so the module needs the query anyway, and the event would carry a fact only one subscriber uses.
- *Let the healthcheck module read the connection settings through `GetPostgresqlLogicalConnectionSettings` and the physical counterpart.* That decrypts secrets to read a host, and healthcheck would switch on the database type, which `backend/AGENTS.md` forbids outside the family.
- *Add the fact to `TelemetryFacts`.* Telemetry facts are what the anonymous statistics report. Reusing them would tie a product rule to the telemetry payload.

### 3. The healthcheck module picks the default when the database is created

`OnDatabaseCreated` asks `IsNeonHost` and saves the default config with `IsHealthcheckEnabled` false for a Neon database. Since the handler runs inside the creating transaction after the settings are saved, the read sees the new row. `GetDefaultConfig` keeps its other values, so the two defaults differ in one field. `OnDatabaseCopied` keeps overwriting the copy's config with the original's.

Alternatives rejected:

- *The frontend saves a second config right after creating the database.* Databases created through the API would keep the checks on, and the scheduler could check the database between the two requests.
- *Decide the default in the upgrade migration as well.* The proposal keeps existing configs untouched. The migration that creates missing configs only serves databases from before healthcheck configs existed, and a SQL migration cannot apply the host rule without copying it into SQL.

### 4. `shouldConfirmEnabling` is a read-only field of `HealthcheckConfig`

`HealthcheckConfig` in `backend/api/openapi.yaml` gains a required boolean `shouldConfirmEnabling`, documented as computed from the current host. It is read-only because only the response schema has it: `HealthcheckConfigInput` does not change. The schema carries no `readOnly` keyword, because oapi-codegen turns a required `readOnly` field into an optional pointer, and the response schema is never a request body anyway. The service computes the flag through `IsNeonHost` for both the read and the save answer; it is not stored, because it describes the host, and a host edit must change it at once. The frontend save already builds its body by destructuring; it also drops `shouldConfirmEnabling`, so the body stays a valid `HealthcheckConfigInput`.

Alternatives rejected:

- *A provider field in the database response.* The editor would need the database object it does not load today, and the frontend would decide which provider needs a warning.
- *A stored column.* A host edit would have to update the healthcheck table from the databases module, which sits lower in the module order and cannot call healthcheck.
- *Parsing the host on the frontend with the constants of `PostgresqlConnectionStringParser`.* That is a second definition of a Neon host, in another language, with a separate test suite.

### 5. The editor opens a delayed confirmation when the switch turns on

When `shouldConfirmEnabling` is true and the switch goes from off to on, the editor keeps the switch off and opens the confirmation. Confirming sets `isHealthcheckEnabled` and marks the form unsaved; cancelling or closing changes nothing. The shared `ConfirmationComponent` gains an optional `actionDelaySeconds` prop: while seconds remain, the confirm button is disabled and its label shows the count from a dictionary key that takes the action text and the seconds. The dialog opens with the delay set to 10.

Alternatives rejected:

- *Confirm on save.* After a refusal the user would have to undo a form they had already filled in, and the save path would grow a second branch.
- *A permanent warning under the switch.* Less code, but the user asked for a deliberate pause before enabling, and a static line is easy to read past.
- *A dialog of its own in the healthcheck slice.* It would copy the shared dialog's layout, focus handling and themed buttons for one extra prop.
- *Moving the master key page onto the new prop.* That page is not a dialog and its wait guards a different control; changing it is outside this change.

### 6. The API tests reach a real server under a Neon name

The healthcheck tests build the topology from the exported `testcontainers` calls that `StartPostgresBehindSSHBastion` already composes (`backend/internal/lib/testcontainers/ssh_bastion.go:76-94`): `StartNetwork`, `StartSSHBastionContainerOnNetwork`, and `StartPostgresOnNetwork` whose `NetworkPlacement.Alias` is a Neon name such as `ep-test-123456.us-east-2.aws.neon.tech`. A small helper in the healthcheck test fixture does this. The database settings use that alias as the host and the bastion as the tunnel. The bastion resolves the alias through the network's DNS, so creation and edit detect a real server while the stored host is a Neon host. No new `testcontainers` fixture is added; the healthcheck API tests fail at creation if the alias does not resolve.

Alternatives rejected:

- *A new `testcontainers` variant of `StartPostgresBehindSSHBastion` that takes the alias, with a databases-module test of its own.* The three existing calls already take the alias, and the healthcheck tests prove the name resolves.
- *A unit test of the healthcheck service with a fake databases module.* The testing skill allows a fake only for an external system we cannot run locally or a failure unsafe to cause; the databases module is neither, and the API test also covers the transaction order of decision 3.
- *The real Neon suite under `backups/e2e/postgresql/neon`.* It needs a Neon project and credentials, so it cannot be the routine check of this rule.

## Risks / Trade-offs

- [Existing Neon users keep paying for checks they never chose] -> Accepted by the proposal. Their next attempt to switch the checks off and on again shows the dialog.
- [A Neon custom domain or a private link host is not recognized] -> Accepted; the database behaves as any other. The rule stays a host suffix, which needs no connection.
- [An extra settings read on every healthcheck config read and save] -> One primary-key lookup in the provider table, on a screen-level request; no decryption and no connection.
- [Docker network aliases with dots] -> Docker's embedded DNS resolves them. If a CI runtime ever rejects one, the fixture fails loudly at start rather than letting the test pass on a wrong host.

## Migration Plan

No data migration. The new default applies to databases created after the deployment. Rolling back restores the single default and drops the dialog; databases created meanwhile keep the config they were given.
