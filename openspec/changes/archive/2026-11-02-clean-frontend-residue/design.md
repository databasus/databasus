## Context

See `proposal.md` for the motivation. Paths are under `frontend/src`. The facts come from commit `864ae947`.

- `constants.ts` holds the runtime configuration, the application version, the server address and the OAuth redirect URI. `shared/api/apiClient.ts:3`, `shared/hooks/useCloudflareTurnstile.ts:3`, `shared/hooks/useIsNewGitHubVersionAvailable.ts:3` and `shared/ui/CloudflareTurnstileWidget.tsx:3` import it upward out of the layers. `lint/checkProjectDependencies.ts` skips any path whose first part is not a layer, so these imports are never checked.
- `shared/hooks/useIsNewGitHubVersionAvailable.ts:6` names this product's GitHub repository, and `app/ui/ApplicationScreen.tsx:68` is its only consumer.
- `useCloudflareTurnstile` and `CloudflareTurnstileWidget` are used only by `pages/auth/ui`.
- `entities/storages/index.ts:2` and `entities/notifiers/index.ts:2` read from `./models/`. Other slices use `model`.
- `entities/backup-configs/model/StorageSwitch.ts:3`, `entities/backup-configs/api/backupConfigApi.ts:56,68` and `entities/disk/model/DiskUsage.ts:3` read `components` from the generated file. Other entities read `ApiComponents` from `shared/api`, whose `shared/api/index.ts:4` provides that name.

The approach is bound by the root `AGENTS.md` (logical units, no unrequested backward compatibility) and `frontend/AGENTS.md`. It follows "Start in the page" and "Keep state near its lifecycle" in `docs/architecture.md`. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- Every TypeScript module under `src` except `main.tsx` and `vite-env.d.ts` lives in a layer. `index.css` stays at the root as the global style sheet.
- Nothing is left that no consumer reads.
- A reviewer can read this diff as moves and deletions only.

**Non-Goals:**

- Any change of behavior, styles or copy.

## Decisions

### 1. What gets deleted

These files have no importer. The empty file is a leftover of the move to `pages/storages`.

| Unit | Path |
| --- | --- |
| Empty file | `pages/storages/ui/StorageMigrationDialogComponent.tsx` |
| Unused view | `pages/databases/ui/show/ShowDatabaseBaseInfoComponent.tsx` |
| Unused generated aliases | `pages/databases/model/physical-backups/GetPhysicalBackupsResponse.ts`, `pages/databases/model/physical-backups/PhysicalRestoreTokenResponse.ts`, `pages/databases/model/verification/GetVerificationsResponse.ts`, `pages/databases/model/wal-stream-monitoring/WalStreamCurrent.ts`, `pages/databases/model/wal-stream-monitoring/WalStreamStreamer.ts` |
| Unused label map | `pages/databases/model/wal-stream-monitoring/WalStreamKeepingUpStateLabelKeys.ts` |
| Unused helper | `shared/lib/StringUtils.ts` and its export |
| Unused exports | `isSupportedLocale` in `shared/i18n/dictionaries.ts`, `CreatedAgentResponse` and `RotateTokenResponse` in `entities/verification-agents`, `VersionResponse` and its `ApiComponents` import in `entities/system/index.ts`, `LocalStorageSettings` in `entities/storages/models/ProviderStorageSettings.ts` and its re-export in `entities/storages/index.ts` |

A dictionary key goes only when a search finds no other reader. Keys that another file still reads stay, for example `common.fields.name`.

Alternative rejected: adding a tool such as knip to find this. The scan was a one-time need of this milestone, and `noUnusedLocals` keeps file-level dead code out once `export` is removed. A new dependency for one pass costs more than it saves.

### 2. Declarations that only their own file uses lose `export`

These 17 declarations have a reader inside their file and none outside it, apart from a re-export for four of them, so `export` falsely announces a public unit: `DatabaseListFilter`; `PhysicalConnectionErrorStep`; `PhysicalConnectionErrorContent`; `clearPostgresqlCertificates`; `LogicalBackupPageRequest`; `ScriptCommandParams`; `DockerScriptCommandParams`; `ManualStepsParams`; `RestoreStep`; `WalStreamChartColors`; `easeOutCubic`; `WalStreamLiveDependencies`; `WalStreamRefreshDependencies`; `AgentStatus`; `WebsitePageId`; `PagedListFailedPage`; `ThemeContextValue`.

Four also lose their re-export lines, which nothing reads: `PhysicalConnectionErrorStep` and `PhysicalConnectionErrorContent` in `entities/databases/index.ts`, `WebsitePageId` in `shared/i18n/index.ts` and `ThemeContextValue` in `shared/theme/index.ts`.

Exports that only tests read stay, because the tests are real consumers.

Alternative rejected: leaving them. `noUnusedLocals` cannot see an unused export, so a declaration that later loses its last reader inside the file would stay silently dead.

### 3. `constants.ts` becomes `shared/config/deploymentConfig.ts`

The file moves with an `index.ts` as its entry point, and every importer points at `@/shared/config` or the matching relative path. The name covers what it holds: values that depend on how this copy of the application was built and deployed, namely the runtime configuration, the version and the server address.

`getOAuthRedirectUri` does not move there. It names the sign-in callback route, reads no deployment value, and only `features/oauth-sign-in/model/OAuthSignInAttempt.ts` calls it, so it becomes the private unit `features/oauth-sign-in/model/getOAuthRedirectUri.ts`. This follows "Authentication attempts have bounded ownership" in `docs/architecture.md`.

Alternatives rejected:

- Splitting the configuration values by consumer (OAuth client IDs to `features/oauth-sign-in`, the Turnstile key to `pages/auth`). `window.__RUNTIME_CONFIG__` is one document-level contract with the server, and splitting would repeat its reading in several slices.
- `shared/lib`. It holds generic helpers, while this is deployment configuration with its own reason to change.

### 4. Product units leave shared, generic units stay

The test is whether a unit holds product policy, per the milestone's "no business logic in shared".

| Unit | From | To |
| --- | --- | --- |
| `useIsNewGitHubVersionAvailable` | `shared/hooks` | `app/model`, beside `useVersionCheck` |
| `useCloudflareTurnstile` | `shared/hooks` | `pages/auth/model` |
| `CloudflareTurnstileWidget` | `shared/ui` | `pages/auth/ui` |

`shared/hooks/useCloudflareTurnstile.ts` is imported by file path today; both Turnstile units lose their shared entry points with no forwarding export.

Alternative rejected: moving every shared unit with one consumer. `FormValidator`, `normalizeNameList` and the others are generic, so moving them would create churn and a second move when a new consumer appears.

### 5. `models` becomes `model`, and aliases read `ApiComponents`

`git mv` keeps file history. Public entry points hide the segment, so only the two slices' own files change. `backupConfigApi.ts` reads the existing `StorageSwitch` alias instead of repeating the schema name. `shared/api/requireData.ts` keeps `components`, because it lives beside the generated file in the slice that owns it.

## Risks / Trade-offs

- [An unused-looking export is read through a dynamic path] → The scan matched names across all source files. Task 2.1 runs the build and the full frontend suite after the deletions, and task 4.2 runs them again at the end.
- [A dictionary key looks unused but is built at runtime] → A key goes only after a search for its parent group and its full name finds no reader. Task 2.1 runs lint and the full frontend suite afterwards.
- [Moving `constants.ts` changes module evaluation order in tests] → It reads `globalThis.window` when imported, and no test mocks its path. The full suite covers this.
