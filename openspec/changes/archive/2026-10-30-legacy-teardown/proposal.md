## Why

The domain modules now run through the application composition root, but compatibility bridges still publish their dependencies through process-wide state. Those bridges prevent concurrent test applications and keep obsolete packages, build tags and exceptions in the development tools and architecture checks.

## What Changes

- Remove the remaining bridges, lazy dependency wiring and old configuration, database, logging, encryption, cache and mail accessors after converting their consumers, including console-command tests and test-application file helpers.
- Move reusable stream and test support into the shared library, place directory operations and backup filename handling with their owners, and delete unused utilities.
- Give every test application its own directories and dependencies. Run independent application tests concurrently and prove that overlapping applications cannot affect each other's data, files or shutdown.
- Replace the development seed command's hand-built test application with the real application and generated test client. Use test-owned source containers, then remove the shared-source reset command and its locks from local runs and CI.
- Apply backend lint and architecture rules to every backend package, enable the check against global variables, and remove the temporary layout exclusions. Keep the architecture's intentional rule-specific permissions and documented third-party exceptions.
- **BREAKING for internal callers and development tooling:** delete the old package paths and getters, process-wide test-database entry point, helper build tag and language-server wrapper without compatibility aliases or replacement wrappers. Production HTTP contracts, stored data, operator configuration and agent protocol stay unchanged.
- Remove migration-period exemptions from current documentation, reviewer prompts and active specifications. Update the hook tests that assert those exemptions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backend-code-checks`: enforce the documented checks throughout the backend, reject global dependencies and remove requirements for legacy layout and tagged helpers.
- `backend-test-isolation`: retire process-scoped legacy databases and destructive shared-source resets while preserving per-test database isolation and failure inspection.
- `api-contract`: allow simultaneous real test applications with independent filesystem state and lifecycles.
- `implementation-review`: remove the architecture review exemption for code awaiting migration.
- `ci-pipeline`: run backend selections without the helper build tag and keep architecture checks applicable after legacy backup code disappears.

## Impact

The change follows root `AGENTS.md` and `backend/AGENTS.md`. It touches backend application wiring, test support, utility consumers, the seed command, lint configuration and architecture checks; mise tasks, CI and editor/agent settings; current architecture/testing documents and review prompts. No dependency or database migration is required.

### Out of scope

- New product behavior, HTTP operations, frontend or website changes, and changes to the verification protocol.
- Removal of upgrade handling for persisted master keys or historical database migrations.
- Removing development database sources that `mise run dev` still uses, redesigning the dev seed command, or consolidating command binaries.
- Changing shard counts, test timeout policy, filesystem-suite scheduling, or requiring a full backend or Docker filesystem test run for a task.
- Removing the historical cascade cutoff or the documented third-party dependency exceptions; rewriting archived changes or the historical migration plan.
- Committing, pushing, publishing or deploying the change.
