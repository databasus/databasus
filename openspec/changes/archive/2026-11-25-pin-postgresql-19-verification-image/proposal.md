# Proposal

## Why

Release v3.73.0 published `databasus/verification-postgres:19.0`. Until then, the agent end-to-end tests built the PostgreSQL 19 image from the recipe, because no published image existed to pin. Now the tests can restore into the published image by its digest, as they do for 12 to 18. The same release exposed two problems in CI:

- Docker Hub answered `429 Too Many Requests` when the eighteen image pushes of a release ran at once. Six pushes failed, and the release went out only after a manual rerun.
- The Podman subset of the agent tests ran two scenarios instead of the three its spec names. The restore container scenario was renamed, and the subset's `-run` pattern no longer matches it.

## What Changes

- The test-image catalog gains the published `databasus/verification-postgres:19.0`, pinned by digest.
- The agent end-to-end tests take a version's verification image from the test-image catalog when it has a record, and build it from the recipe when it has none. The PostgreSQL 19 case stops being a special case, and a future pinned version needs no special case either.
- The Podman subset restores its valid backup on PostgreSQL 19, the newest supported major, as the subset's requirement already says. Its pattern matches the restore container scenario again. The task fails and names each scenario of the subset that reported no result, and a script test covers that behavior.
- `scripts/image_pins.py` takes the majors from `postgres_versions.json` instead of a fixed range of 12 to 18. Each major's verification base must match the test-image catalog's base record. A test-image catalog record of a verification image must still name a pinned version. A pinned version may lack a record until its first release publishes the image.
- Release image pushes run at most eight at a time. A push or manifest step that Docker Hub refuses with `429` waits and tries again, up to three more times. Any other failure stops the step at once, as before.
- `docs/how-to-upgrade-deps.md` describes the order for a raised or new pinned version: the first release publishes the image, and a later commit pins it in the test-image catalog.

Nothing is **BREAKING**. The local-build helper of the PostgreSQL 19 case becomes the fallback for every version without a test-image catalog record, so no code becomes unused.

## Out of scope

- Raising any pinned PostgreSQL version or adding a major.
- The CloudNativePG failover guarantee for PostgreSQL 19, which waits for CloudNativePG's PostgreSQL 19 images.
- Retrying any registry error other than `429`, or retrying test jobs.
- The High CodeQL alert `go/disabled-certificate-check` in `backend/internal/engines/mysqlfamily/driver_connection.go`, which is unrelated and older than this change.
- Moving other agent scenarios from PostgreSQL 18 to 19.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: release image pushes are limited in concurrency and wait out a registry rate limit instead of failing the release.

## Impact

- Test-image catalog: `backend/internal/lib/testimages/images.json`.
- Agent end-to-end tests: `backend/internal/agent/e2e/verification_images_test.go`, `verify_backup_test.go`.
- Podman subset: `mise-tasks/agent/e2e-podman`, with a new script test.
- Dependency check: `scripts/image_pins.py` and its test in `scripts/dependency_verification_test.py`.
- Release workflow: `.github/workflows/ci-release.yml`, jobs `push-image`, `merge-image-manifest`, `push-verification-image` and `merge-verification-image-manifest`. A new script retries a command on a registry rate limit, with its own test.
- Documentation: `docs/how-to-upgrade-deps.md`.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.
