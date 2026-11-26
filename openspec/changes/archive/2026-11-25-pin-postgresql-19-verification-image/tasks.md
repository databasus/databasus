# Tasks

## 1. Catalog and dependency check

- [x] 1.1 Add `databasus/verification-postgres:19.0` to the test-image catalog `backend/internal/lib/testimages/images.json` with the index digest and both platform digests from design.md, and verify that `go -C backend test -count=1 ./internal/lib/testimages/...` passes
- [x] 1.2 Make `scripts/image_pins.py` take the majors from `postgres_versions.json`: check each major's recipe base against the test-image catalog's `postgres:<major>` record, keep the check that every verification record names a pinned version, and let a pinned version without a record pass. Extend `scripts/dependency_verification_test.py` with a base mismatch for 19 and a pinned version without a record, and verify that `python3 -m unittest discover -s scripts -p 'dependency_verification_test.py'` and `mise run verify-deps` pass
- [x] 1.3 Describe in `docs/how-to-upgrade-deps.md`, under "Images and verification extensions", the two commits of a raised or new pinned version from design.md. The raise commit changes the pinned version, the recipe's base digest and the test-image catalog's `postgres:<major>` record together, and removes the old version's `databasus/verification-postgres` record. A commit after the release that publishes the new image adds its record, and the agent tests build the image from the recipe until then. Correct the sentence about which mismatches `mise run verify-deps` catches, and verify that `rg -n "has no record for a pinned version" docs` finds nothing

## 2. Agent end-to-end tests

- [x] 2.1 Make the agent's verification image helper look up the pinned version with `testimages.GetReference` directly, use the test-image catalog record when one exists and build the recipe with `docker buildx build --load` when the lookup returns `ErrUnknownImage`. Run every major from 12 to 19 in one `TestVerifyBackup` loop and remove `isLocallyBuiltImage`, then verify that `go -C backend test -count=1 -run 'TestVerifyBackup/completes_with_an_encrypted_backup_on_postgres_(12|19)$' ./internal/agent/e2e/` passes and `mise run backend:lint` reports nothing
- [x] 2.2 Rebuild the Podman subset in `mise-tasks/agent/e2e-podman` from one list of three scenario names, with the valid backup on PostgreSQL 19 and the restore container scenario under its current name, run `go test -json` through `scripts/go_test_report.py` into a ran-tests file, and make the task fail and name each listed scenario without a line in that file. Add a script test that runs the task with a stub `go` printing a recorded `-json` stream and a stub socket, for a run missing one scenario and a run with all three, and verify that `python3 -m unittest discover -s scripts -p '*_test.py'` passes and `mise run lint-shell` reports nothing

## 3. Release pushes

- [x] 3.1 Add `scripts/retry-on-registry-rate-limit.sh`, which retries a command after 60, 120 and 240 seconds when its failed output names a registry rate limit, and returns any other failure at once. Add a script test with shortened waits for a success after a rate limit, a rate limit on every attempt and an unrelated failure, and verify that the script tests pass and `mise run lint-shell` reports nothing
- [x] 3.2 Replace `docker/build-push-action` in `push-image` and `push-verification-image` with `docker buildx build` through the retry script, reading the digest from `--metadata-file` and failing on a missing digest. Run the three `imagetools create` calls through the script, and set `max-parallel: 8` on `push-verification-image`. Add workflow test assertions that `push-verification-image` sets `max-parallel: 8`, that both push steps and the three `imagetools create` calls in `push-image`, `merge-image-manifest`, `push-verification-image` and `merge-verification-image-manifest` run through `scripts/retry-on-registry-rate-limit.sh`, and that a push step fails on a missing `containerimage.digest`. Update the existing workflow tests that read these jobs, and verify that `python3 -m unittest discover -s scripts -p '*_test.py'` and `mise run lint-workflows` pass

## 4. Integration

- [x] 4.1 Run `mise run lint` and `mise run verify-deps` over the whole change and verify that both pass
