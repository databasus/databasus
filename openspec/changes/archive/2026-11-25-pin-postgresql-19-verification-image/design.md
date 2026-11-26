# Design

## Context

See proposal.md, "Why", for the motivation.

- The test-image catalog records the verification images of 12 to 18 by digest (`backend/internal/lib/testimages/images.json:38-86`). Release v3.73.0 published `databasus/verification-postgres:19.0`, whose index digest is `sha256:e9b29c4152d6e690f5f95545bca5976a1231f86c28d077805d403dfc0e6f8ae3`, with `sha256:042c005e51a4ee919b9732b968a9a1500a67f07f1ca4c4d4a19cce7c154ab60d` for amd64 and `sha256:6f8320ff41514b728a2aa154d74302fdbfb3736909abd31ad6dca41a25d8c9c7` for arm64.
- `TestVerifyBackup` runs 12 to 18 on test-image catalog images and 19 in a separate subtest that builds the image from the recipe (`backend/internal/agent/e2e/verify_backup_test.go:23-37`, `:99-103`, `verification_images_test.go:64-90`).
- `scripts/image_pins.py:14` and `:24` check majors 12 to 18 only. CI runs `mise run verify-deps` in the `lint-python` job (`.github/workflows/lint-python.yml:36`), and a release needs that job. A rule that demanded a test-image catalog record for every pinned version would therefore block the commit that raises a version: the image of the new version does not exist until that commit is released.
- The Podman subset's pattern names `runs_without_root_or_capabilities_when_restoring_a_backup` (`mise-tasks/agent/e2e-podman:34`), while the scenario is now called `runs the pinned server version without root or capabilities when restoring a backup` (`backend/internal/agent/e2e/restore_container_test.go:24`). `go test` reports nothing for a pattern that matches no subtest. The run of 2026-11-25 passed with only two subtests.
- The release pushes images with `docker/build-push-action` in `push-image` (`.github/workflows/ci-release.yml:1337`) and in the sixteen `push-verification-image` jobs (`:1458`). The manifest jobs run `docker buildx imagetools create` (`:1402`, `:1408`, `:1532`). The matrices set `fail-fast: false` and no `max-parallel` (`:1316`, `:1427`, `:1492`).

The approach follows the root `AGENTS.md` rules on flaky failures (fix the cause, retry nothing that hides a defect) and on unused code. It also follows the `backend/AGENTS.md` rule that test libraries stay in test files. No backend production code changes.

## Goals / Non-Goals

**Goals:**

- Every pinned major of the agent end-to-end suite restores into the image its release published, whenever that image is recorded.
- A future raise of a pinned version passes CI before its image exists, and its test-image catalog record follows in a later commit.
- A Docker Hub rate limit during a release costs a wait instead of a rerun.

**Non-Goals:**

- Detecting which images a release changed, or skipping unchanged pushes.
- Changing what the agent pulls at run time.

## Decisions

### A test-image catalog record is used when it exists, and the recipe is built when it does not

The test helper that provides an agent's verification image looks up the pinned version of the major with `testimages.GetReference` directly. The current lookup goes through `containers.GetPinnedImage` (`backend/internal/lib/testcontainers/image.go:11`), which fails the test on any error but a missing platform, so the helper cannot use it for this decision. With a record, the helper pulls the digest and tags it into the agent's repository, as today. When the lookup returns `testimages.ErrUnknownImage`, it builds the recipe for the host architecture with `docker buildx build --load`, as the PostgreSQL 19 subtest does today. Every major then runs in one loop, and the PostgreSQL 19 special case and its `isLocallyBuiltImage` field disappear.

Alternatives rejected:

- Delete the local build once 19 has a record. The next raise of a pinned version would need to add it back, because the raised version has no published image until its release.
- Keep a per-scenario flag. It makes the person who raises a version edit the test as well, and nothing tells them to remove the flag later.

The build branch runs only while a pinned version lacks a record. After this change no version lacks one, so the branch is exercised again at the next raise. This is acceptable, because the branch is the same code that ran for 19 in the previous change.

### The dependency check reads the pinned majors and tolerates a missing record

`scripts/image_pins.py` iterates the majors of `postgres_versions.json`:

- The verification recipe's base stage of each major must equal the test-image catalog's `postgres:<major>` record, as it does for 12 to 18 today. The base is an upstream image, so it always exists.
- Every test-image catalog record under `databasus/verification-postgres` must name a pinned version, as today.
- A pinned version without a verification record passes. The agent tests build its image instead.

Alternatives rejected:

- Require a record for every pinned version. That blocks the raise commit in CI, as described in "Context".
- An explicit list of versions that wait for their first publication. It is a second place to edit for every raise, and forgetting to clear it changes nothing that a test notices.

`docs/how-to-upgrade-deps.md` changes accordingly. A raise of a pinned version, such as 18 moving from `18.7` to `18.8`, takes two commits:

1. The raise commit changes the version in `postgres_versions.json`, the recipe's base digest for the major and the test-image catalog's `postgres:<major>` record to the same new base. It also removes the old version's verification record, here `databasus/verification-postgres:18.7`, because every verification record must name a pinned version. `mise run verify-deps` passes, and the agent tests of that major build the image from the recipe.
2. The release of that commit publishes `databasus/verification-postgres:18.8`. A later commit adds its record to the test-image catalog, and the agent tests switch to the published image.

A new major follows the same order without a record to remove.

### The Podman subset checks that each named scenario ran

`mise-tasks/agent/e2e-podman` keeps the three scenario names in one list and builds the `-run` pattern from it. The names become `completes_with_an_encrypted_backup_on_postgres_19`, `fails_after_one_attempt_when_the_backup_is_broken` and `runs_the_pinned_server_version_without_root_or_capabilities_when_restoring_a_backup`. The task runs `go test -json` through `scripts/go_test_report.py`, as `run_reported_go_test` in `scripts/test.sh:238-241` does. The report prints the readable output and writes one `<package>\t<test>\t<result>` line per finished test to a ran-tests file. After the run, the task fails and names every listed scenario that has no line in that file. The log then shows the output of failed and skipped tests only, as in a test plan run, instead of the full `-v` output. A script test runs the task with a stub `go` that prints a recorded `-json` stream and a stub socket. It shows that a missing scenario fails the task, and that a run with all three passes.

Alternatives rejected:

- Fix the pattern only. The next rename would again drop a scenario without a signal.
- Parse the `go test -v` text for `--- PASS` lines. It would be a second parser of Go test output beside `go_test_report.py`.
- A static script test comparing names with `t.Run` strings. The PostgreSQL 19 scenario's name is built in a loop, so it does not appear literally in the source.

The restore container scenario keeps PostgreSQL 18. The subset's requirement asks for the newest major only for the valid backup.

### Release pushes run through a retry on the registry's rate limit

A new script, `scripts/retry-on-registry-rate-limit.sh`, runs its arguments as a command and copies the output to the log. When the command fails and the output contains `429 Too Many Requests` or `toomanyrequests`, it waits and runs the command again. The waits are 60, 120 and 240 seconds, so a command gets four attempts in about seven minutes. Any other failure, or the last attempt's failure, returns the command's exit status at once. A test variable shortens the waits for the script's test.

Both push jobs replace `docker/build-push-action` with a `run` step that calls `docker buildx build` with the same platform, build arguments and `push-by-digest` output through the script. The step writes `--metadata-file` and reads the digest from it with `jq`. This is the same builder the action used, since neither job sets one up. The three `imagetools create` calls go through the script too, because they talk to the same registry. They run in `/tmp/digests`, so they call the script by its path under `$GITHUB_WORKSPACE`, and `merge-image-manifest` gains the checkout it lacked.

Alternatives rejected:

- Keep the action and repeat it in a second step under `if: failure()`. That retries every failure, not only the rate limit, waits a fixed time and doubles the step's configuration.
- `max-parallel` alone. It lowers the burst but does not survive a limit that another project on the same account triggers.
- A third-party retry action. It adds a dependency and a pin for twenty lines of shell.

### At most eight verification pushes run at once

`push-verification-image` sets `max-parallel: 8`. Its sixteen jobs then run in two waves. With the two `push-image` jobs, at most ten pushes run at once. The release of 2026-11-25 failed with eighteen; earlier releases passed with sixteen. A wave takes about as long as one verification build, so a release takes a few minutes longer.

Alternatives rejected:

- Four at once. The retry already covers the rest, and four waves add more time than the margin they buy.
- No limit, retry only. Eighteen concurrent pushes cause the burst in the first place, and each retry wave would repeat it.

## Risks / Trade-offs

- [Docker Hub keeps refusing for longer than seven minutes] → The push fails as before, and a person reruns the failed jobs. The log shows the `429` of every attempt.
- [`docker buildx build` and the action produce different metadata] → The step fails on a missing `containerimage.digest` instead of uploading an empty digest, and the release stops before any manifest is created.
- [A pinned version stays without a test-image catalog record for a long time] → Its agent tests build the image from the recipe and still pass. The procedure in `docs/how-to-upgrade-deps.md` names the follow-up commit.
- [The local build branch rots while every version has a record] → It is a single `docker buildx build` call with the recipe's own server version check. The first raise runs it again.

## Migration Plan

No data or configuration migrates. The next release runs the new push steps.
