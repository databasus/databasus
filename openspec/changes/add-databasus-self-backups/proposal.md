## Why

Losing the Databasus server also loses its users, settings, credentials and backup catalog, even when database backup files survive in remote storage. The owner also has no in-app prompt to save the encryption key needed to recover those files.

## What Changes

- Add encrypted backups of the complete internal database, scheduled and started manually, to one or more existing workspace storages. These backups contain instance state and the backup catalog, not the stored database backup files or the encryption key.
- Keep live references to workspace storages. Do not copy their settings, introduce global storages or change who can edit them. Block deletion of a selected storage, including through workspace deletion, until root admin removes it from the Databasus backup destinations.
- Give root admin a Databasus backups section in global settings: destination selection across workspaces, inline workspace/storage creation with automatic selection, shared schedule and retention, history, per-destination outcomes and archive download. Default to hourly backups and 30-day retention.
- Prompt root admin to save the master key on their first authenticated visit, including after upgrading an existing installation. Allow later download after identity confirmation. Show HTTP risk and host-file guidance before any key transfer.
- Show setup reminders only after the instance has a user database or storage. Report failed, partially saved and overdue backups through a persistent floating notice and, when available, the instance SMTP server. Removing the final destination immediately reports that backups have stopped, even if onboarding would otherwise suppress a reminder.
- Explain missing SMTP in the backup section and link directly to the existing mail-server section with scroll and focus. Reuse the system mail service; add no notifier selection.
- Support two recovery paths using an archive and its separately saved master key: startup recovery from mounted files into a fresh instance or one containing only automatically created state, and root-confirmed import of those two files from the computer through the Databasus backups section. Starting Databasus before supplying the files does not prevent startup recovery if no user accounts, user data or user-changed settings exist; adding both files, configuring both file paths and restarting is sufficient. UI import replaces the current installation; it does not depend on its backup history, storages or master key. To reuse a backup listed in history, download it first and import the file.
- Require the exact Databasus release, including its patch version, for both paths. A mismatch refuses recovery and names the required image version; upgrading happens only after recovery on that version. Persist the SHA-256 of a successfully applied startup archive so leaving the files and path settings configured does not repeat recovery or prevent a later ordinary upgrade.
- Start recovered instances in a durable pause that prevents background activity and remote writes/deletions until root admin confirms the old server is stopped. Verify a complete backup/recovery cycle, including subsequent recovery of logical and physical database backups.
- Extend the refusal to open a database migrated by a newer image with an explicit fresh-install alternative for an owner who deliberately wants the older release. Explain the data loss and provide separate cleanup commands for a bind-mounted data directory, a named Docker volume and the anonymous volume created when no mount is supplied. Keep restarting the newer image as the way to preserve the current installation; resetting storage does not downgrade its data.

### Out of scope

- Global storages, copied credentials, extra restrictions on workspace storage editors, or a special workspace type.
- Secondary copies of users' database backups, storage scanning/catalog reconciliation, immutable storage management and a general notification rule engine.
- Recovery directly from a history entry, remote-storage browsing/discovery during recovery, or automatic archive retrieval by the import form. The owner supplies files obtained independently.
- Startup recovery over an instance containing user accounts, user data or user-changed settings, cross-version restore, unattended activation of a recovered instance and monitoring an entirely stopped Databasus process. Replacing such an instance is supported only through explicitly confirmed UI import. Reset guidance does not add reverse migrations or automatically delete any data.

## Capabilities

### New Capabilities

- `databasus-backups`: encrypted internal-database archives, live destinations, scheduling, independent delivery, retention, safe archive access and durable history.
- `databasus-recovery`: exact-version startup recovery and confirmed UI file import, SHA-based startup deduplication, failure safety, paused activation and treatment of recovered runtime state.
- `databasus-backup-ui`: root-only configuration, inline creation, key onboarding, archive/key file import, destination outcomes, SMTP navigation and floating reminders.
- `databasus-backup-alerts`: missing, failed, partial, overdue and stopped protection, system SMTP delivery and bounded repeated alerts.
- `master-key-export`: root-only, explicitly confirmed key download with recent identity proof, HTTP warning and persistent acknowledgment.

### Modified Capabilities

- `frontend-application`: workspace creation inline in Databasus backup configuration preserves the settings page, address and workspace selection; ordinary creation still opens the new workspace's databases destination.
- `frontend-workspace-screens`: inline backup configuration uses a newly created workspace without selecting it application-wide or losing the calling draft; ordinary workspace creation keeps its selection behavior.
- `storages`: selected Databasus backup destinations cannot be deleted; their normal editing and transfer permissions remain unchanged.
- `workspace-deletion`: deletion also refuses a workspace containing a selected Databasus backup destination.
- `internal-database-migrations`: startup recovery validates and imports before the normal migration path, including after ordinary initialization that created only automatic state; UI import requires the same release and schema checks and preserves imported migration history. Neither recovery path upgrades an archive. A newer-schema refusal also explains how to discard the installation and start fresh on an older release for each Docker storage layout.
- `backup-streaming`: encrypted temporary staging is a bounded-memory disk-use exception for internal-database archives delivered to several storages.
- `storage-file-cleanup`: lost-server recovery does not replay copied source-instance cleanup obligations; ordinary restart and upgrade guarantees remain unchanged.

## Impact

- Backend: a new `instancebackups` module, storage reference/deletion integration, bootstrap-admin identity checks, system mail, streamed archive/key uploads, staged recovery, migrations and generated user API. Application composition coordinates exclusive publication and application reinitialization after UI import. The verification-agent API remains unchanged.
- Frontend: root-only global-settings content, reusable workspace/storage creation flows, app-level notices, key download, archive/key file import with validation, replacement confirmation and reconnect/sign-in states, and all interface dictionaries. System SMTP continues to be configured by the operator; the UI links to its existing instructions and test-email action.
- Runtime: private encrypted staging files, a pre-migration recovery entry point, a persistent recovery journal outside the imported snapshot and a durable recovery pause. UI publication waits for application work and connections to stop and reinitializes the application without requiring a Docker restart policy. Ordinary deployment startup remains unchanged when no unfinished recovery or new startup recovery inputs exist.
- Documentation: define the new terms and their code/API names in `docs/glossary.md`, including Root admin and its persisted bootstrap designation, Databasus backup, its destination and copy, Databasus backup health and incident, Databasus recovery, recovery pause and schema identity. Distinguish recovering Databasus's internal state from restoring a user's database. Extend the existing Master key entry instead of introducing a second name or secret. Update architecture, recovery instructions, SMTP guidance, FAQ, security page and README installation advice, including the corresponding six translations. Do not remove the documented manual data-directory backup option before describing the limits of the new archive.
- Agent instructions: add a narrow existing-master-key attachment-export exception beside the secret-response rule in `.agents/skills/api-design/SKILL.md`, with root-only access, recent purpose-bound identity confirmation, short-lived single-use download authorization, no-store and secret-free audit requirements. Ordinary stored-secret response prohibitions remain unchanged.
- This change answers to root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`. It requires no new external service or dependency.
- No existing API, configuration setting or archive format is removed. Recovery of these new archives intentionally requires the exact source release rather than accepting a newer release as discussed in the exploratory notes.
