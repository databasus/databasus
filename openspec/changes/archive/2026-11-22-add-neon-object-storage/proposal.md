## Why

Neon now sells S3-compatible object storage, and Databasus users who already keep their databases on Neon will want to put backups there. Today they can only reach it by filling a generic S3 form by hand, nothing checks that Databasus works with Neon's S3 implementation, and a backup larger than about 10 GiB would fail there because Databasus writes objects of up to about 15.6 GiB while Neon accepts at most 10 GiB per object.

## What Changes

- Every S3 storage writes objects of at most about 4.7 GiB (300 pieces of 16 MiB) instead of about 15.6 GiB, so a large backup stays under Neon's documented 10 GiB limit and under the 5 GiB cap that AWS sets for a single upload and that some sources give for Neon and other S3-compatible services. Backups already written keep reading and deleting as before.
- The storage type list offers "Neon Object Storage". It saves an ordinary S3 storage; the API and the stored settings gain no new type.
- The Neon choice shows a block that reads the credentials block Neon Console gives (`AWS_ENDPOINT_URL_S3`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`) from the clipboard into the S3 fields, followed by an "or enter the details manually" divider and the usual fields. The bucket is entered by hand, because Neon's block does not hold it. A saved S3 storage whose endpoint is at Neon opens with the same Neon block.
- The clipboard import used by the database connection form moves to the shared layer so both forms use one implementation.
- A storage provider test saves, reads and deletes files on a real Neon Object Storage bucket, including a file split into multipart pieces and a cancelled upload. It is skipped locally without its settings and required in the CI `storage-providers` check. The same scenario runs against MinIO on every run.
- The Neon Object Storage credentials reach CI only as repository secrets, exposed only to the step that runs the `storage-providers` tests, and the secret key never reaches test output.
- The website gains `/storages/neon-object-storage` in all 7 languages: a step-by-step guide with search and answer-engine metadata (HowTo and FAQ structured data), linked from the storages page, the docs sidebar, the Neon database guide, the sitemap and `llms.txt`. The README in all 7 languages names Neon Object Storage, without a link, at the end of its cloud storage list next to Cloudflare R2.

## Capabilities

### New Capabilities

- `managed-object-storage-tests`: checks Databasus S3 storage against a real managed object storage, Neon Object Storage, with a container twin, run isolation, cleanup and secret handling.

### Modified Capabilities

- `storages`: S3 objects stay within 5 GiB; the storage form offers Neon Object Storage as an S3 alias with a credentials paste; public materials describe Neon Object Storage setup.
- `ci-pipeline`: the `storage-providers` check requires the Neon Object Storage test and is the only check that receives its secrets.

## Out of scope

- A separate Neon storage type in the API, the database or the stored settings.
- Neon branding (name or logo) on storage cards, storage details, backup settings, database cards, transfer dialogs or instance backup views; they keep showing S3, as databases on Neon keep showing PostgreSQL.
- Presets for other S3-compatible providers such as Cloudflare R2, Backblaze or Wasabi.
- Creating Neon buckets or credentials from Databasus.
- A full backup and restore cycle through the API against Neon Object Storage; the provider-level test covers the S3 behavior Neon can differ in.
- Per-provider object size limits or a setting for them.
- Rewriting existing backups into the new object size.

## Impact

- Backend: the S3 provider's object layout constant, a new test target in the storage providers test package, `.env.example`.
- Frontend: the storage type list and S3 form, an S3-compatible service description for Neon and its endpoint detection in the storages entity, the clipboard import moved from the database connection feature to the shared layer, all 7 interface dictionaries, component and browser tests that list storage types.
- CI: `.github/workflows/ci-release.yml` environment of the Go test step, the secret scoping script test in `scripts/`, five repository secrets.
- Website: a new translated page in 7 locales with screenshots, `app/i18n.ts`, sitemap, sidebar, storages page, Neon database guide, `llms.txt`; the frontend website page table.
- Docs: `docs/testing.md`, `docs/glossary.md` (managed object storage service), `README.md` and the six `assets/readme/README.<locale>.md` copies.
- No API, schema or migration change. No break for existing callers or stored storages.

The change answers to `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`.
