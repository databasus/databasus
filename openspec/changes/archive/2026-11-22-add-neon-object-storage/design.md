## Context

See proposal.md for the motivation. The current state that shapes the approach:

- The S3 provider writes a file that exceeds one 16 MiB piece as numbered objects of up to `defaultMaxPartsPerObject = 1000` pieces each (`backend/internal/modules/storages/internal/providers/s3/bucket.go:48`, `:56`), about 15.6 GiB per object, described by a manifest. Reading follows the manifest's list of objects and deleting lists the objects by their name pattern (`bucket.go`, `openFile` and `derivedObjectKeys`), so neither depends on the constant.
- Neon Object Storage documents S3 support for every request the provider sends (multipart start, part upload, complete, abort and listing, batch delete, ranged reads, object metadata, bucket location), SigV4 only, path-style addressing only, and a maximum object of 10 GiB for multipart uploads; a third-party guide says 5 GiB. The provider's default bucket lookup (`minio.BucketLookupAuto`, `bucket.go`, `getClientParams`) picks path style for a non-AWS endpoint. A live `ListBuckets` with the provided credential answered, and the branch holds the bucket `databasus-bucket`.
- No test reaches a real cloud S3 service. The storage provider package runs generic scenarios over container targets (`backend/internal/modules/storages/internal/providers/provider_test.go:256` `getProviderTargets`, `:361` `buildS3Storage`) and runs alone in the CI `storage-providers` group (`.github/workflows/ci-release.yml:820-826`). `scripts/test.sh:124-127` sources `.env` for local runs.
- The Neon database test sets the pattern this change follows: environment gating with a pure decision function (`backend/internal/modules/backups/e2e/postgresql/neon/neon_connection_test.go:36-95`), a marker with a run-unique part and creation time plus a one-hour sweep (`test_run_test.go:19-66`, `:134-196`), a redactor (`secret_redactor_test.go:11-40`), a step-scoped secret (`ci-release.yml:802-803`) and a Python test that pins that scoping (`scripts/neon_secret_test.py`).
- The storage form picks the type from a select built from `Object.values(StorageType)` (`frontend/src/features/edit-storage/ui/edit/EditStorageComponent.tsx:98-101`, `:145-153`); changing it resets the settings through `createDefaultStorageSettings(type)`. A saved storage shows its type as text with the type icon (`:143`, `:156-160`). The S3 form links only to the Cloudflare R2 guide (`frontend/src/features/edit-storage/ui/edit/storages/EditS3StorageComponent.tsx:49-61`).
- The Neon database alias keeps the provider in UI state while the backend type stays PostgreSQL (`frontend/src/pages/databases/create-wizard/ui/CreateDatabaseComponent.tsx:94`), describes providers in a table (`frontend/src/entities/databases/model/postgresql/ManagedPostgresqlProvider.ts:25-52`), detects Neon from the host suffix (`getManagedPostgresqlProviderFromHost.ts:4-17`) and renders the paste block with its divider (`frontend/src/features/database-connection/ui/ManagedConnectionStringPaste.tsx:58-62`). The clipboard flow lives in `frontend/src/features/database-connection/model/useConnectionStringImport.ts` and hardcodes `databases.edit.*` messages (`:38`, `:53`, `:55`, `:69`).

The approach is constrained by `docs/architecture.md` "Backup data flow" (one piece in flight per upload, memory independent of database size), by `frontend/AGENTS.md` (Feature-Sliced Design layers, `BrowserClipboard` and `ClipboardPasteModalComponent` for every clipboard read, all seven dictionaries), by `website/AGENTS.md` (the translated page convention, the sync rule, `TRANSLATED_PATHS` and `websitePages.ts`), by `assets/readme/AGENTS.md`, and by the root `AGENTS.md` rules on secrets and tests.

## Goals / Non-Goals

**Goals:**

- Keep every S3 object within 5 GiB without changing how existing backups are read or deleted.
- Verify against a real Neon bucket every S3 request the provider makes, including multipart abort after cancellation.
- Give the Neon choice its own entry and paste flow while storing a plain S3 storage.

**Non-Goals:**

- Measuring Neon's real object size limit; the test writes tens of MiB, not gigabytes.
- A catalog of S3-compatible services beyond the one Neon entry the form needs.

## Decisions

### 1. Lower `defaultMaxPartsPerObject` to 300 for every S3 storage

300 pieces of 16 MiB give 4800 MiB (about 4.69 GiB), under both the 5 GiB and the 10 GiB figure. The constant's comment states the new reason: the smallest common object cap of S3-compatible services, AWS's single-PUT limit, with Neon as an example.

Alternatives rejected:

- A per-provider limit chosen from the endpoint. It adds detection to the backend for one number, and an unknown provider would still get the risky default.
- A setting in the storage form. Users cannot know their provider's cap better than a safe default does, and the cost of a smaller object is only a few more requests per backup.
- 320 pieces, exactly 5 GiB. A provider that treats the limit as exclusive would refuse the last object.

The cost: a 100 GiB backup becomes about 22 objects instead of about 7, with the same number of piece uploads. Memory does not change, because one piece is still in flight. Existing backups keep their larger objects and read and delete through the manifest and the name pattern.

### 2. A provider-level test target in the storage providers package, not a backup cycle through the API

The test is a new scenario in `backend/internal/modules/storages/internal/providers`: connection test, a small file, a file of a few pieces, digest comparison, delete, then a cancelled multi-piece save, each followed by a listing of objects and incomplete multipart uploads under the run's prefix. The same function runs on the MinIO target, so the scenario is checked on every `mise run test`.

With the production layout, a file of 20 MiB exercises multipart (two pieces in one object, the second one partial), and the cancelled save stops after its first piece was uploaded. The sizes stay small because a run uploads them over the developer's own connection. The roll-over to a second object is Databasus logic that the in-package MinIO tests already cover with a small layout (`s3/provider_test.go:722-738`), so the real-provider test does not need access to the unexported layout.

Alternatives rejected:

- A backup and restore cycle through the API, like the Neon database test. It adds a PostgreSQL source and a restore target and still sends Neon the same S3 requests, so it costs minutes and containers without covering anything more of Neon.
- Placing the test in package `s3`. That package runs in the `other` shards, so the secret would have to reach shards whose membership changes, which breaks the step-and-group scoping the CI rule requires.
- Adding Neon to `getProviderTargets` so every generic scenario runs on it. Those scenarios include credential and failure cases that would send many more requests to a shared bucket on every run and depend on Neon for behavior that is not Neon-specific.

### 3. Five settings variables, all required together

`TEST_NEON_S3_ENDPOINT`, `TEST_NEON_S3_REGION`, `TEST_NEON_S3_BUCKET`, `TEST_NEON_S3_ACCESS_KEY`, `TEST_NEON_S3_SECRET_KEY`, plus `TEST_IS_NEON_S3_REQUIRED`. A pure decision function, tested as a table like `decideNeonRun`, returns skip, run or a failure naming the missing variables. Partial settings fail, so a typo in one name cannot switch the test off.

All five are repository secrets. The endpoint and bucket identify the Neon project, so masking them in CI logs is acceptable; local output still names them for diagnosis.

Alternatives rejected:

- One URL-shaped secret such as `https://key:secret@host/bucket?region=…`. It invents a format that nobody copies from Neon and that has to be escaped by hand when a key holds special characters.
- Endpoint, region and bucket as GitHub repository variables. That splits one setting group across two mechanisms and two scoping checks for no security gain.

### 4. Run isolation by key prefix with a one-hour sweep

Each run sets the storage prefix to `databasus_e2e_<unix seconds>_<random hex>/`. Cleanup is registered before the first write, runs on a context detached from cancellation with its own deadline, removes every object and aborts every incomplete upload under the prefix, and fails naming each key it could not remove. Before its work, a run lists top-level marked prefixes, removes those older than one hour and only logs a failure there, because another run may be sweeping the same prefix. This mirrors `test_run_test.go`.

Alternative rejected: a bucket per run. Creating buckets needs broader credential scopes, and Neon's free plan caps storage per project rather than buckets, so it buys nothing.

### 5. A redactor for both keys

Every message that can carry a client error goes through a redactor built from the access key and the secret key, including their URL-escaped forms. A MinIO-twin case with a deliberately wrong secret proves that the failure names the endpoint and bucket but contains neither key, reusing the recording reporter pattern from the Neon database test.

Alternative rejected: relying on GitHub masking. It hides only exact secret values in CI and does nothing for local runs or files.

### 6. CI scoping and its pinned test

The Go test step sets each of the five variables to `matrix.group == 'storage-providers' && secrets.<NAME> || ''` and `TEST_IS_NEON_S3_REQUIRED` to `'true'` for that group only. `scripts/neon_secret_test.py` becomes a table over secret groups (the Neon database secret for `logical`, the five object storage secrets for `storage-providers`), keeping its existing assertions for each. The `storage-providers` step already runs `go test` on the isolated package with a 40-minute timeout, which covers the few added requests.

Alternative rejected: a second Python test file copying the first. Two copies of the same workflow parsing drift apart.

### 7. Neon is a UI choice over the S3 type

The storage entity gains an `S3CompatibleService` table with one entry, Neon (name, logo, examples, guide page, paste description), and `getS3CompatibleServiceFromEndpoint`, which matches an endpoint host that is `neon.tech` or ends in `.neon.tech`, next to the storage types. The name follows the glossary's "managed object storage service" and stays clear of "provider", which the glossary reserves for one implementation of a storage type such as the backend S3 provider. The type select builds its options from the storage types plus the S3-compatible services; the selected value is either a storage type or a service. Choosing Neon keeps type S3 and the current S3 values; switching between S3 and Neon does not call `createDefaultStorageSettings`. For a saved storage, the service is derived from the saved endpoint when the form opens, and the type text and icon stay S3.

The S3 form receives the S3-compatible service. With Neon, it renders the paste block and divider, Neon examples and the Neon guide link instead of the Cloudflare R2 link.

Alternatives rejected:

- A new backend storage type. It needs a migration, an API change and a second set of S3 code paths for a storage that is S3.
- Storing a provider field in the S3 settings. It adds an API and schema change while the endpoint already identifies Neon.
- Provider tiles as in the database wizard. The storage form is reused in five places as a compact select; tiles would redesign all of them for one option.

### 8. The clipboard import moves to the shared layer

`useConnectionStringImport` moves to `frontend/src/shared/lib/` and becomes `useClipboardTextImport`, a hook whose `parse` takes the pasted `text` and which takes the result handler and its notification texts (empty clipboard, read failure, parsed, parsed with a warning). The database connection feature passes its current `databases.edit.*` texts, so its behavior and tests do not change. The storage feature passes `storages.edit.s3.neonPaste.*` texts. The paste block's markup (title, hint, example, button, divider) moves to `frontend/src/shared/ui/` as `ClipboardPasteBlockComponent`, which takes those texts; `ManagedConnectionStringPaste` becomes a thin use of it. The old name would describe a connection string, which a Neon credentials block is not.

Alternatives rejected:

- Importing the hook from `features/database-connection` into `features/edit-storage`. Feature-Sliced Design forbids imports between features.
- Copying the hook and block into the storage feature. The two copies would diverge at the next fix of clipboard handling.

### 9. A dotenv-shaped parser in the storages entity

`parseNeonObjectStorageCredentials(text)` reads lines `NAME=value` with optional `export `, single or double quotes and `#` comments, and ignores unknown names. It returns an error when no `AWS_ENDPOINT_URL_S3` with a host at `.neon.tech` is present. A key counts as missing when it is absent, empty, or holds whitespace, angle brackets or an ellipsis, which is how placeholders look; the result then lists the missing keys, and the form shows the warning. The parser does not check Neon's current key prefixes (`nak_`, `nsk_`): if Neon changes them, a prefix check would refuse valid keys. The parser never logs and returns no message that holds a value.

Alternatives rejected:

- Accepting any non-empty key value. A placeholder from the Console would be saved as a credential and fail only at the connection test, with an S3 error that does not say what to do.
- Parsing only the exact four lines in the exact order. Users edit and reorder `.env` blocks, and Neon may add lines.

### 10. The guide page follows the Neon database guide

The page lives at `website/app/(en)/storages/neon-object-storage/page.tsx` with six `website/app/[lang]/storages/neon-object-storage/content/<locale>.tsx` copies and a dispatcher, and is added to `TRANSLATED_PATHS`, the sitemap, the sidebar, the storages page, `llms.txt` and `frontend/src/shared/i18n/websitePages.ts`. Its structure follows `/faq/neon`, the most recent guide: numbered steps with screenshots, then a short "Questions" section whose answers also go into `FAQPage` structured data next to the `HowTo` data. The English title targets "Neon Object Storage backup"; each locale targets its own search phrasing per `website/AGENTS.md`. Neon Console screenshots come from `temp/photos`, which the user fills before apply; if it is still empty, the agent captures them in the user's signed-in browser through the browser tools. Databasus screenshots are taken from the running app.

The term "managed object storage service" goes into `docs/glossary.md` next to "Managed PostgreSQL service", because the capability, the frontend table and the guide use it.

Alternatives rejected:

- Copying the Cloudflare R2 page's shape. It has no FAQ, no structured answers and no restore context, which is what answer engines quote.
- Placing the guide under `/faq/`. Storage guides live under `/storages/`, and the user asked for that path.

## Risks / Trade-offs

- [Neon Object Storage is in beta and may change limits or behavior] → The CI test fails on the first change it affects, and the guide states the beta status.
- [The real test depends on Neon availability and can block a release] → This is the intended gate, as with the Neon database test. A Neon outage is visible in the check's failure message, which names the endpoint.
- [Neon may refuse `Content-MD5` or a checksum header that MinIO accepts] → The test sends the same requests a backup sends, so such a refusal fails the test before users meet it. The fix would then belong to the S3 provider, not to the test.
- [A shared bucket fills with abandoned objects from killed runs] → The one-hour sweep and the per-run cleanup; the free plan's 5 GB per project is far above the few tens of MiB a run writes.
- [Neon's soft delete may keep deleted objects billable] → The test checks only what listing shows. The guide states that deletion follows Neon's own retention rules.
- [More objects per large backup on every S3 provider] → About three times more multipart starts and completions, against thousands of piece uploads per such backup; negligible.
- [The exact placeholder Neon Console shows before a credential is generated is unknown] → The parser treats empty and placeholder-shaped values as missing; when the Console screenshots show the real placeholder, a parser test pins it.
- [Endpoint detection by `.neon.tech` may also match a non-storage Neon host] → Such a host is never a valid S3 endpoint, so the only effect is that a misconfigured storage shows the Neon paste block.
- [Credentials pasted in the chat that set up this change] → The CI and `.env` use a credential the user can revoke and reissue in Neon Console without code changes.

## Migration Plan

No data migration. Existing S3 backups keep their objects. Rollback restores the constant; files written with smaller objects remain readable by the old code, because reading follows the manifest.

The user's Neon credentials go into the local `.env` (untracked) and into the five repository secrets before the CI check is required; the change sets `TEST_IS_NEON_S3_REQUIRED` only in CI.
