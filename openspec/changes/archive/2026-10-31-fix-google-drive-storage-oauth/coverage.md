## Retained guarantees

| Surface | Coverage | Guarantee |
| --- | --- | --- |
| Storage editor | `EditStorageComponent.test.tsx` | Saved provider type stays fixed; rename and explicit test/save omit stored secrets; S3 storage classes remain intact. |
| Application route | `App.test.tsx` | `/storages/google-oauth` remains reachable independently of sign-in and rejects old draft-bearing callbacks. |
| Storage API | Existing backend storages API tests | Secret omission and retaining omitted credentials remain unchanged; no backend or API contract change. |
| Config consumers | Existing backup config screen tests and new browser scenario | Authorization updates the nested editor without saving or changing the surrounding config or storage selection. |

## Reproduction before repair

Ran `pnpm --dir frontend test src/features/storages/ui/edit/EditStorageComponent.test.tsx src/app/ui/App.test.tsx` on the unchanged implementation: 11 passed, two intended failures. The authorization state contained the serialized draft and synthetic client secret instead of a random identifier. The legacy callback retained its credential-bearing query instead of clearing and rejecting it.

## Authorization and callback coverage

| Surface | Coverage | Guarantee |
| --- | --- | --- |
| Attempt lifecycle | `GoogleDriveAuthorization.test.tsx` | Independent editors, random replacement state, single-use acceptance, exact ten-minute expiry, future clock, settings/workspace/session replacement, unmount and StrictMode setup replay. Late exchange success and controlled network rejection cannot publish tokens or feedback or cancel a newer attempt. Tests await the observed fetch settlement before checking the current attempt, then authorize that attempt successfully. |
| Exchange and persistence | `GoogleDriveAuthorization.test.tsx` | Fixed Google token endpoint, captured form body, no application Authorization header, required access/refresh tokens, refusal/network/provider failures, unchanged saved settings and retained prior draft tokens. Rename omits secrets; failed explicit save retains the authorized draft. A connection test for old credentials cannot authorize saving tokens installed by a later OAuth exchange. Old test success and network failure are ignored after settings, workspace, storage or session replacement and unmount; they cannot finish a newer pending test. |
| Callback relay | `StorageOAuthPage.test.tsx` and `App.test.tsx` | Old JSON, malformed/duplicate parameters and mixed code/error callbacks are rejected without requests. Query clearing precedes relay; code/refusal is captured across setup replay, acknowledgment checks origin/state, and a missing editor ends at five seconds. |
| Localization | `GoogleDriveAuthorization.test.tsx` and existing dictionary tests | Input and active attempts survive locale/theme changes. A visible refusal translates in place. All six dictionaries retain complete keys and placeholders. |
| Cross-window interaction | `browser/scenarios/googleDriveStorage.spec.ts` | Actual backup-settings form, changed unsaved cron, retained original storage and workspace, nested Drive editor, removed opener, acknowledgment/close, duplicate delivery and explicit test/save. Canceling the authorized editor preserves the original selection. |
| Browser recovery and containment | `browser/scenarios/googleDriveStorage.spec.ts` | Refusal, missing refresh token, canceled/replaced settings, blocked popup and orphan callback. Navigation, both windows' relayed messages and browser storage contain no credentials or draft. New controls pass scoped accessibility and 320px containment checks. |
| HTTP boundary | `browser/HttpFixture.ts` and `tooling/verifyBrowserProbes.ts` | Finite synthetic Google routes are installed before popup creation. Context-wide tracking catches the first unexpected popup request and popup errors. Form-encoded token bodies are parsed explicitly; existing caught HTTP/XHR and reference-isolation probes remain enforced. |

Retained API coverage includes `backend/internal/modules/storages/secrets_test.go` for hidden/encrypted Google Drive credentials and preserving omitted secrets, plus `update_test.go`. Retained config coverage includes `EditLogicalBackupConfigComponent.test.tsx`, `EditPhysicalBackupConfigComponent.test.tsx`, `DatabaseConfigComponent.test.tsx` and `entities/backups/config/api/backupConfigApi.test.ts`. These existing frontend tests ran in the complete frontend suite; unchanged backend tests were not rerun for this frontend-only change.

## Verification and limits

- `pnpm --dir frontend test`: 123 files, 1083 tests passed.
- `pnpm --dir frontend test:browser --project=ui`: 38 browser scenarios passed, including nine Google Drive scenarios and the boundary failure probes.
- `mise run frontend:lint`: formatting, source lint, FSD, styles, TypeScript and generated API freshness passed.
- `pnpm --dir frontend build`: passed; existing runtime-config and large-chunk warnings remain.
- `openspec validate fix-google-drive-storage-oauth --strict`: passed.

Google responses and credentials are synthetic. Browser tests establish local orchestration, containment and recovery, not live Google client configuration or CORS acceptance. The browser still holds entered secrets and returned tokens in memory. Reloading the initiating editor requires a fresh attempt; stored credentials and the backend contract are unchanged.

## Additional regression checks

The overlapping connection-test/OAuth regression failed on the reviewed implementation because the old success exposed Save for untested tokens. The repaired editor invalidates the submitted test revision when settings or context change and checks it on both success and failure.

Temporarily removing only the OAuth catch-path current-attempt guard made five controlled late-rejection cases fail with stale alerts after cancellation, replacement and settings/workspace/session changes. The guard was restored. Unmount is covered separately for absent feedback and released channels.
