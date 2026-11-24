## 1. Pinned versions and image selection

- [x] 1.1 Add `backend/internal/agent/container/postgres_versions.json` with 12.22, 13.23, 14.25, 15.20, 16.16, 17.12 and 18.7, embed it, decode and validate it when the container manager is constructed (not at import time), and expose `GetPinnedPostgresVersion(pgMajor)`. `NewManager` gains an error return, and `backend/internal/agent/start/start.go` stops the agent on it. Verify with a unit test that every major from 12 to 18 is present and each version belongs to its major, and that a broken list is rejected by the parser that manager construction calls; verify the startup path with `mise run agent:e2e`.
- [x] 1.2 Resolve the verification image as `<repo>:<version>` and the stock fallback as `postgres:<version>`, with no fallback when the configured repository is `postgres`, leaving TimescaleDB images unchanged. Verify with the updated tests in `backend/internal/agent/container/postgres_test.go` for the default repository, a configured repository, the fallback, the configured official repository and a TimescaleDB job, and with `mise run agent:e2e`.
- [x] 1.3 Verify with a test on the fake engine that an image held locally under `<repo>:18` or `<repo>:18.6` is not used, and the agent asks for `<repo>:18.7`.

- [x] 1.4 Add startup cleanup of superseded verification images: list local tags of the verification repository, remove by tag without force every tag that is not a pinned version, skip everything when the repository is `postgres`, log a warning and continue on any failure, and call it in `backend/internal/agent/start/start.go` after `PurgeContainers`. Verify with fake-engine tests for each scenario of "The agent removes only its own superseded verification images" in `backend/internal/agent/container` (`mise run agent:test`) and with `mise run agent:e2e`.

## 2. Verification image build and release

- [x] 2.1 Pin the `FROM` stages of `backend/cmd/verification-agent/images/postgres/Dockerfile` for 14-18 to the official images of the pinned versions, after checking `postgres --version` in each candidate digest. Update the matching `postgres:<major>` records in `backend/internal/lib/testimages/images.json`. Verify with `mise run verify-deps`.
- [x] 2.2 Add the `PG_SERVER_VERSION` build argument and the build-time check that both the base `PG_VERSION` and the installed server match it. Verify by building majors 12 and 18 locally with the right version (succeeds) and with a wrong one (fails with a message naming both versions).
- [x] 2.3 In `.github/workflows/ci-release.yml`, read the version for each major from `backend/internal/agent/container/postgres_versions.json`, pass it as `PG_SERVER_VERSION` to the check build and to the publishing builds, and add the `:<version>` tag to the published manifest. Verify with `mise run lint-workflows`.

## 3. Tests on pinned images

- [x] 3.1 Rename the `databasus/verification-postgres` catalog records to `<version>` names and pin them to digests whose server is that version, checked with `postgres --version`. Extend `scripts/image_pins.py` to read `backend/internal/agent/container/postgres_versions.json` and require a catalog record under each version name. Add tests to `scripts/dependency_verification_test.py` for a missing version record and for a record whose name disagrees with the list. Verify with `mise run verify-deps` and `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'`.
- [x] 3.2 Make the agent end-to-end setup configure each test agent with a verification repository unique to the test run and tag the pinned catalog digests locally under that repository and their version names before the agent starts, so the startup cleanup of one run never removes tags another worktree's run uses on the shared Docker daemon, update the image list in `backend/internal/agent/e2e/source_test.go`, and assert in the restore test that the restore container uses the agent's pinned tag and that `postgres --version` inside it reports `GetPinnedPostgresVersion` for the backup's major. Verify with `mise run agent:e2e`.

- [x] 3.3 Add an end-to-end test in `backend/internal/agent/e2e` that starts an agent on its run-unique verification image repository holding a stale `:<major>` tag, a superseded version tag and a version tag used by a container, and checks which tags remain. Verify with `mise run agent:e2e`.

- [x] 3.4 Move the custom tablespace of the physical PostgreSQL fixture in `backend/internal/lib/testcontainers/postgres_physical.go` to `/tmp`, because the PostgreSQL 18.7 image no longer makes the tmpfs over `/var/lib/postgresql` writable for the init script. Verify with `go test ./internal/engines/postgresql/physical/ ./internal/lib/testcontainers/` from `backend/`.

## 4. Documentation

- [x] 4.1 Add to `docs/how-to-upgrade-deps.md` the rule for shortening the waiting period: allowed for a release that fixes a security issue or a defect that breaks users' backups, restores or verification; the release comes from an official source; the exception covers only that release; the report and the commit record the reason, the release, its publication date and the shortened period. Verify that the text agrees with the `dependency-maintenance` spec and replaces, rather than repeats, the existing one-line exception.
- [x] 4.2 Document in `docs/how-to-upgrade-deps.md`, under images and verification extensions, that raising a PostgreSQL version of the verification image edits `postgres_versions.json`, the Dockerfile base and the catalog together, and that the build check and `mise run verify-deps` reject a disagreement. Search the repository for other mentions of the verification image tags and bring them in line.
- [x] 4.3 Add `docs/glossary.md` entries for "Verification image" (with its verification image repository) and "Pinned PostgreSQL version", and use these terms in the code names and comments the change adds. Verify by searching the change's code and artifacts for other names of the same concepts.

## 5. Checks

- [x] 5.1 Run `mise run backend:lint`, `mise run python:lint`, `mise run verify-deps`, `mise run agent:test`, `mise run agent:e2e` and `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'`, and fix any failure.
- [x] 5.2 Record in the commit message and the final report the shortened waiting period for 14.25, 15.20, 16.16, 17.12 and 18.7: published 2026-11-12, normal eligibility 2026-11-27, reason CVE-2026-6460 and broken verification for existing agents. Verify by reading the commit message before committing.
- [x] 5.3 Run `mise run test`, `mise run agent:test`, `mise run agent:e2e`, `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'`, `mise run verify-deps` and the linters, and repeat any package that failed after its fix.

## Change verification

```sh
mise run test
```
