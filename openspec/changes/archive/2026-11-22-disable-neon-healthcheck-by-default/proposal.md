## Why

Neon suspends a project's compute after a few minutes without connections and does not bill compute while it sleeps. Databasus checks every new database once a minute by default, and each check opens a connection, so a Neon database added to Databasus never sleeps and its compute is billed around the clock. The user pays for this without having asked for it, and nothing on the screen warns them.

## What Changes

- A new database whose host is a Neon host gets its healthcheck switched off by default. Its other healthcheck values stay the usual defaults, so switching the checks on later gives a check every minute, as for any database.
- The healthcheck config read and save answers carry a new read-only field, `shouldConfirmEnabling`, which is `true` for a database whose current host is a Neon host. The server decides which databases need the warning, so the frontend does not parse hosts.
- When a user switches the healthcheck of such a database from off to on, the editor shows a confirmation dialog. It explains that Neon stops billing compute while the database sleeps and that a check every minute keeps it awake, so compute is billed all the time. The confirm button stays locked for 10 seconds. Cancelling leaves the switch off.
- Neon host recognition moves to one place in the PostgreSQL engine and serves both the existing refusal of physical backups and the new default.

The new response field is additive. Nothing breaks for existing callers.

## Out of scope

- Supabase and any other managed PostgreSQL provider. A provider is added only with a concrete scenario showing that the checks raise its bill.
- Existing databases. Their stored healthcheck configs stay as they are, including Neon databases that already have checks switched on. No migration changes them.
- A change of host after creation. Editing a database to point at Neon does not switch its checks off; the confirmation dialog still appears the next time someone switches them on.
- Copies. A copied database keeps inheriting the original's healthcheck config, as it does now.
- Neon reached through a custom domain or another host name that does not end in `neon.tech`.
- Warnings for other settings that wake Neon, such as the backup schedule or the check interval.
- The default interval and other default values for databases outside Neon.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `database-healthcheck`: the default config of a new Neon database has checks switched off, and the config answer tells whether switching the checks on needs a confirmation.
- `frontend-database-screens`: switching the healthcheck on for a database that needs confirmation shows a dialog with a confirm button locked for 10 seconds.

## Impact

- Backend: the PostgreSQL engine (`internal/engines/postgresql`), the databases module, which exposes whether a database is hosted on Neon, and the healthcheck module, which picks the default and fills the new field.
- API: `backend/api/openapi.yaml` gains `shouldConfirmEnabling` in `HealthcheckConfig`; the generated server, test client and frontend types are regenerated.
- Frontend: the healthcheck editor, the shared confirmation dialog and all seven interface dictionaries.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
