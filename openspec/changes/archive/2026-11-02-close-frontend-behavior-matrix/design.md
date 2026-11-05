## Context

See `proposal.md` for the motivation. Paths are under `frontend`. The matrix is in `temp/refactor/frontend-analysis.md`, and the audit read test bodies for its auth, shell and general refusal rows and for every item below. It relied partly on titles and search for the profile, storage, connections, composition, backup config, logical and verification rows. The physical and i18n rows were read only for the items below, so their other behaviors, such as restore commands, polling cleanup, dictionaries and themes, have no confirmed test body yet. More gaps may appear in all of these.

- `src/shared/lib/test/mockApi.ts` provides `network`, `malformed`, `pending` and `delayed`, and fails on an undeclared request. `buildProblem` sets `detail` to the error code, so a test of private server text must send its own `detail`.
- `playwright.config.ts:23-25` fixes `locale: 'en-US'` and `reducedMotion: 'reduce'` for every scenario.
- `src/pages/instance-settings/lib/verificationAgentCommands.ts` builds the agent download and launch commands. `VerificationAgentsComponent.test.tsx:460` opens the install dialog and clicks its copy buttons without asserting the commands.
- `backend/api/openapi.yaml` marks these fields `writeOnly`: `password` for NAS and FTP, `connectionString` and `accountKey` for Azure, `configContent` for rclone, `botToken` and `proxyUrl` for Telegram, `channelWebhookUrl` for Discord, `powerAutomateUrl` for Teams, and `webhookUrl` and `botToken` for Mattermost.
- Offline has no separate behavior in the code: nothing reads `navigator.onLine`. A lost network reaches the monitoring screen as a failed reading.
- CI already repeats the local checks: `lint-frontend` runs `frontend:lint` and the build, and `test-frontend`, `test-frontend-browser-ui` and `test-frontend-browser-e2e` run the rest and feed the release. Every action is pinned by SHA.

The approach is bound by the testing skill, `docs/testing.md` and the root `AGENTS.md` rule that defects exposed by checks are fixed at once, without weakening assertions.

## Goals / Non-Goals

**Goals:**

- Every matrix item has a test exercising the user's path, not only a helper.
- A failure of any of these tests means a real regression.

**Non-Goals:**

- Rewriting existing tests that already own their item.

## Decisions

### 1. Each open item gets a test at the lowest layer that proves it

Screen tests own behavior with HTTP responses. Browser scenarios own what happy-dom cannot show: real fonts, the browser's language and region, chart geometry and media queries.

| # | Item | Where the test goes | What it proves |
| --- | --- | --- | --- |
| 1 | Replayed OAuth callback | `src/pages/auth-callback/ui/AuthCallbackPage.test.tsx` | Opening the same URL after a completed exchange sends no second exchange |
| 2 | Sign-in refusal | `src/pages/auth/ui/SignInComponent.test.tsx`, `browser/scenarios/auth.spec.ts` | A refused sign-in keeps the input, shows the reason and does not reload; the browser scenario asserts the values it claims |
| 3 | Logout cleanup | `src/app/ui/ApplicationSession.test.tsx`, `browser/e2e/signIn.spec.ts` | The Logout button clears the token, the user, `selected_workspace_id`, `databasus_oauth_sign_in_attempt`, the scoped selection keys and both backup filter keys (Decision 6), and keeps language, theme and the version-reload cooldown |
| 4 | SMTP not configured | `src/pages/instance-settings/ui/MailServer.test.tsx` | The test action follows `isEmailConfigured` |
| 5 | Storage variants | `src/features/edit-storage/ui/edit/EditStorageComponent.test.tsx` | NAS, both Azure methods, FTP and rclone send the entered settings; a rename omits each secret; a replacement sends it |
| 6 | Notifier variants | `src/features/edit-notifier/ui/edit/EditNotifierComponent.test.tsx` | Telegram, Discord, Teams and Mattermost do the same with their secrets |
| 7 | Refused transfer | `src/pages/storages/ui/StorageTransferDialogComponent.test.tsx`, a new `src/pages/notifiers/ui/NotifierTransferDialogComponent.test.tsx` | A refusal keeps the dialog and resource in place and shows the translated reason |
| 8 | Refused database delete | `src/pages/databases/ui/DatabasesComponent.test.tsx` | The database stays selected and listed, and the reason appears |
| 9 | Encryption save | `src/pages/databases/ui/backup-config/EditLogicalBackupConfigSection.test.tsx` and its physical counterpart | Changing encryption on the screen sends the new value |
| 10 | Trigger save | The same two files | Changing notification triggers in the settings section sends them |
| 11 | Dirty and reset | The same two files, `src/pages/databases/ui/DatabaseConfigComponent.test.tsx` | Returning to the saved values disables Save in the sections; in the settings screen, Cancel discards the draft and the view shows saved values |
| 12 | Logical viewer actions | `src/pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx` | A viewer gets no cancel, delete or verify, and keeps start, restore and download, as the contract allows |
| 13 | Logical repeated click | The same file | A second click on start or delete sends one request |
| 14 | Logical status actions | The same file | Each backup status offers exactly its actions |
| 15 | FULL and INCREMENTAL badges | `browser/scenarios/physicalBackups.spec.ts` | The existing list scenario also asserts the incremental label, beside its `FULL` and `WAL` assertions |
| 16 | Physical terminal states | `src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx` | `CHAIN_BROKEN` and `CANCELED` show their status and allowed actions |
| 17 | Visibility | The monitoring screen test in `src/pages/databases/ui/wal-stream-monitoring` | Hiding the tab stops requests, and showing it resumes them |
| 18 | Lost network | The same screen test | A network rejection keeps the last values, and the next success updates them |
| 19 | Chart geometry and series | `browser/scenarios/wal.spec.ts` | The chart draws the fixture's series and labelled axes at positive size |
| 20 | Reduced motion | `browser/scenarios/wal.spec.ts`, switching with `page.emulateMedia` | The preference and its change reach the real screen, and transitions stop only with `reduce` |
| 21 | Install commands | `src/pages/instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx` | The shown launch command holds host, agent ID, token placeholder and limits |
| 22 | Agent downloads | The same file | The shown URL carries the chosen architecture, and the switch updates it |
| 23 | Browser language and region | A new `browser/scenarios/browserRegion.spec.ts` | `zh-TW` opens the Chinese dictionary, and `en-GB` and `en-US` differ in clock and date order under one language |
| 24 | CJK fallback | The same file | Chinese text renders with a CJK font reported by Chromium, with no external font request |
| 25 | Malformed proxy body | A screen test using `mockApi.malformed` in `src/app/ui/WorkspaceResources.test.tsx` | The screen shows safe translated feedback with a retry |
| 26 | Unknown code and private text | The same file, with its own `detail` | The screen shows the generic message, and the server's text does not appear |

Alternatives rejected:

- Browser scenarios for every item. They are slower and cannot control a late or malformed response as precisely as the screen harness.
- Testing helpers alone for items such as region or font fallback. That is what left them partial, because a helper passing does not show that the screen uses it.

### 2. "Offline" is tested as the failure the product actually sees

Item 18 sends a network rejection during a reading and then a success. It closes the matrix's offline item without adding `navigator.onLine` handling, which would be a new feature.

Alternative rejected: a new offline state on the screen. The plan adds no product behavior, and a user gains nothing over the last values staying visible.

### 3. Provider tests are one table per editor

Each row holds a provider, its settings and its secret fields, and the test asserts the body of the outgoing request. A new provider then adds a row. The existing S3, SFTP, Google Drive, email, Slack and webhook tests stay as they are, because they cover provider-specific rules.

Alternative rejected: one test per provider, which would repeat the same steps ten times.

### 4. Removed tests are explained here, not in archived changes

`coverage.md` of this change records the outcome of the title audit, because an archived change describes its moment. The known case: `is false while the database is still being created` left `hasStoredSshTunnelSecretsForAuthType.test.ts` in `407b31e1`, because that commit removed the function's `databaseId` parameter, while `2026-11-01-refactor-database-connection-form/coverage.md` says "No test was dropped". The roles now express this through draft types, and the record states that.

### 5. Acceptance records the run

`coverage.md` ends with the commands, test counts and durations, which is what the plan requires. Commands that need a push or CI stay out of the tasks. The CI configuration is read, not run.

### 6. Session transitions clear scoped selections and backup filters

`src/shared/auth/session.ts:47-52` removes the token, the user, `selected_workspace_id` and the OAuth attempt, while `selectedDatabaseId_<workspace>`, `selectedStorageId_<workspace>` and `selectedNotifierId_<workspace>` stay. These keys are scoped by workspace, not by account. When two accounts share a workspace, `useScopedSelection.ts:36` accepts what the previous account selected. The `frontend-session` requirement says the previous account's selection does not appear in the new session, and `docs/architecture.md` retains only language, theme and version-reload cooldown, so this is a defect. The same holds for `logicalBackupListFilter` (`LogicalBackupsComponent.tsx:51`) and `physicalBackupsFilters` (`PhysicalBackupsComponent.tsx:58`), which `publishSession` also leaves in place: the next account opens the backup lists narrowed by the previous account's statuses, types and dates, with the filter panel closed, so recent backups can seem missing.

A test first reproduces each case: account A selects a database and filters a backup list, logs out, and account B, which lists the same database, signs in. Then `publishSession` removes every key matching `selected<Resource>Id_` and the two filter keys, beside the removals it already does. Matching the shape covers a future selection type too. The GitHub release cache stays, because it holds no account's choice. The same user also starts without a remembered selection or filter after signing back in, as already happens for the workspace.

Alternative rejected: scoping the keys by user as well as workspace. The same user would keep a selection and filters, but the session rule would gain a new retained item, which needs a spec and architecture change for a small convenience.

### 7. One verify confirmation at a time

A test of item 45 showed that two clicks on a logical backup's verify action opened two confirmations, and that confirming both sent two `POST /verifications`. `LogicalBackupsComponent` now holds a ref while a confirmation is open, and releases it when the confirmation is declined or its request settles. The main specs did not require this, so the change carries a delta spec for `frontend-database-screens`.

Alternative rejected: guarding only `startVerification`. It would stop the second request and still stack a second dialog over the first.

## Risks / Trade-offs

- [A new test exposes a defect] → It is reproduced, fixed in this change, and the affected artifact is updated, including a delta spec if the fix gives a user behavior the main specs do not already require.
- [The finished audit adds more items than the 26] → They go into the same table in `coverage.md` and are closed here, because the owner chose to close the whole matrix.
- [Font assertions depend on the machine] → The scenario runs in the digest-pinned browser image that supplies the CJK font, and asserts through Chromium's reported font rather than a screenshot.
- [Region scenarios conflict with the fixed locale] → They override `locale` with `test.use` in their own file, so other scenarios keep theirs.
