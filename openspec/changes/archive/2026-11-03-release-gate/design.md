## Context

See proposal.md for the motivation. Current state:

- `determine-version` waits for lint and tests only (`.github/workflows/ci-release.yml:1045`). It does not wait for `build-image`, which guards only `push-image` (`ci-release.yml:1146`), so `push-verification-image` (`ci-release.yml:1303`) can tag verification images when `build-image` failed.
- Smoke checks and Trivy run on amd64 only, in `build-image` (`ci-release.yml:891-1018`) and in a near-identical copy in `build-dev-image` (`ci-release.yml:625-751`). The arm64 tool bundle in `assets/tools/arm` is copied into the image by architecture (`Dockerfile:121-125`), so an amd64 check says nothing about it.
- `codeql.yml` holds CodeQL, gitleaks and semgrep as an independent workflow triggered by push to `main`, pull requests and a schedule (`codeql.yml:3-11`). Its concurrency cancels in-progress runs on every ref (`codeql.yml:16-18`).
- The CodeQL analyze step uploads results and does not fail on them. The CodeQL analyze jobs of the run of 2 November 2026 on `7efbe9eb` succeeded with 5 critical and 19 high CodeQL alerts open for that commit. Each alert instance carries its configuration: the API returns `analysis_key` `.github/workflows/codeql.yml:analyze` for the instances checked on `main`.
- gitleaks reads all branches, because the checkout fetches them and the scan has no log restriction. The scheduled runs of 26 October and 2 November failed on `7efbe9eb`, which had passed on 24 October. Reproduced locally: `origin/main` is clean, and `origin/develop` has two findings in commit `37e8dcd3`. Both are the same test constant in `legacy_master_key_test.go:18` and `app_test.go:213`, a fixture ending in `-legacy`.
- `Dockerfile:2`, `:19`, `:49` and `:78` use tags without digests. Trivy installs from apt without a version in three places (`ci-release.yml:121-128`, `:730-737`, `:997-1004`). `npm install -g semver` has no version (`ci-release.yml:1069`).
- `gosec` and `containedctx` are not enabled in `backend/.golangci.yml`, so five `nolint` comments for them do nothing.
- `dependency-review.yml:7-9` grants `pull-requests: write` at the top level. The active ruleset on `main` holds only deletion and non-fast-forward rules, so no check is required for a merge.

Terms, all added to `docs/glossary.md`:

- "Release gate" is the set of checks a commit on `main` must pass before a version is selected.
- "Release analysis" is the CodeQL analysis run as part of the release gate, reported under the key `.github/workflows/ci-release.yml:analyze`.
- "Severity gate" is the `codeql-severity-gate` job, which reads CodeQL alerts by severity.
- "Smoke check" is one start-and-probe check of a running image.
- "Image check" is the whole of `scripts/check_image.sh`: smoke checks plus the Trivy scans.
- "CodeQL alert" is a result of static analysis, which can be dismissed.
- "Secret scan finding" is a gitleaks result, which can only be allowlisted.

Constraints: the Security section of the root `AGENTS.md` (digest pinning, per-job permissions), `backend/AGENTS.md` for the Go comments, `website/AGENTS.md` and `assets/readme/AGENTS.md` for the translated copies. `docs/architecture.md` does not cover CI and does not change.

## Goals / Non-Goals

**Goals:**

- One release gate whose inputs a test can read from the workflow files.
- A release that stops before any tag is written, on either architecture.

**Non-Goals:**

- Testing the exact bytes that get published. `push-image` still rebuilds with the release version, as it does today.
- A manual bypass of the release gate. Rerunning a failed job after a dismissal is the only path, and a fix goes through a commit.

## Decisions

### Scans join the release gate through a reusable workflow

`codeql.yml` gains `workflow_call`, and a new `code-scanning` job in `ci-release.yml` calls it on `main` only and is added to `determine-version`'s `needs`. The caller job grants `contents: read`, `actions: read` and `security-events: write`, because a called workflow cannot exceed its caller. `codeql.yml` drops `push: main` and keeps `pull_request` and `schedule`, so `main` is not scanned twice.

A push to `main` touching only `website/**` does not start `ci-release.yml`, so it loses its push-time scan. It does not release either, and the weekly schedule still covers it.

Rejected:

- Moving the scan jobs into `ci-release.yml`. This loses the pull request runs for `website/**` changes, which `ci-release.yml` ignores, and it needs a separate schedule path.
- Waiting on the other workflow's result from a polling job or `workflow_run`. It needs a polling loop and is a looser link than `needs`.

The concurrency setting becomes `cancel-in-progress: ${{ github.event_name == 'pull_request' }}`. The called workflow shares the group `codeql-refs/heads/main` with the Monday scheduled run, so without this the schedule could cancel a running release scan. A newer push cannot do that, because `ci-release.yml:24-26` already keeps one active run on `main`. A queued `ci-release` run can still be replaced by a newer push. That is existing behavior and harms nothing, because version selection reads `LATEST_TAG..HEAD` and the next release includes those commits.

### Critical CodeQL alerts stop the release, read from the GitHub API

A new `codeql-severity-gate` job in `codeql.yml` runs after `analyze` when the input `is-release-gate` is true. It calls `scripts/check_codeql_alerts.sh`, which uses `gh api --paginate` to list open CodeQL alerts for the ref, filters the raw JSON with `jq`, prints the blocking ones, exits non-zero on critical ones and prints a `::warning::` line for high ones. `gh` exits non-zero on an HTTP error, so a failed read fails the severity gate.

`gh` and `jq` are pinned in `mise.toml` and installed with `install_args`, because nothing in `ci-release.yml` uses `gh` today and the runner image is not known to carry them. `jq` runs as its own step rather than through `gh --jq`, so a test with a stub `gh` exercises the real filter, and the job that runs the script tests installs `jq` too. The job grants itself `contents: read` and `security-events: read` and sets `GH_TOKEN` from the job token, since the top level of `codeql.yml` grants only `contents: read`. The logic lives in a shell script so `shellcheck` reads it and a test can run it against a stub `gh`, the way `scripts/test_sh_test.py` stubs its tools.

The analyze action waits for GitHub to process the upload by default, so the CodeQL alerts are current when the severity gate reads them. Reading the alert list honours dismissals, and a rerun of the failed job picks them up without a new commit.

The threshold is critical only, chosen by the user. It matches the Trivy rule in this repository, where CRITICAL blocks and HIGH is advisory.

Rejected:

- Only requiring the analysis to run. It costs nothing, but CodeQL alerts would still not stop a release and the README claim would stay untrue.
- Failing on the SARIF file. It knows nothing about dismissals, so a false positive would block releases forever or need a suppression in code.
- Critical and high together. That means triaging 24 alerts instead of 5 before the first release, and the user chose the lower threshold.
- Python from the standard library. It needs by hand the pagination and error handling `gh` already gives.

### The severity gate counts only the release analysis

Calling `codeql.yml` from `ci-release.yml` splits `main` into two CodeQL configurations. CodeQL builds the analysis key from the calling workflow's path and the job, so a called run reports as `.github/workflows/ci-release.yml:analyze`, while the schedule stays `.github/workflows/codeql.yml:analyze`. GitHub documents that one alert can show different statuses from different configurations, and that infrequent configurations go out of date.

So a critical or high CodeQL alert counts only if it has an instance with `analysis_key` `.github/workflows/ci-release.yml:analyze` in state `open`. The script reads `/code-scanning/alerts/{number}/instances` for each open critical and high alert. A commit that fixes an alert clears the severity gate at once instead of waiting for Monday. The old instances under `codeql.yml:analyze` do not count either, including the current ones on old paths.

That key for a called workflow is inferred from how CodeQL builds it, not yet observed. A wrong key would match no instance and let the severity gate pass with critical alerts open. So the script first reads `/code-scanning/analyses` for the ref and fails unless the released commit has an analysis under that key for each of the three categories (`/language:go`, `/language:javascript-typescript`, `/language:actions`). A wrong key then stops the release instead of hiding alerts.

Rejected:

- Running the scheduled analysis through the same caller. `ci-release.yml` would need a `schedule` trigger and a guard on each of its jobs.
- Dropping the schedule from CodeQL. The instances of the old configuration would stay open with nothing to update them, so the filter would still be needed, and pull requests would lose their baseline on `main`.
- Keeping `push: main` in `codeql.yml` and polling for its result. This keeps one configuration, but a failed analysis would show up only as a timeout of the polling job.

### The gating secret scan reads only the released history

The same `is-release-gate` input sets `--log-opts=HEAD` on `gitleaks git`. Other runs keep reading all branches.

Rejected: leaving `--all` in the release gate. This is the cause of the current red schedule, and any pushed branch could stop a release of unrelated code.

The two findings go into `.gitleaksignore` with their fingerprints, under the existing heading for test passwords and tokens. The file already holds fingerprints of this kind, and the value is a visible test fixture. Without this the first merge from `develop` stops.

### arm64 runs in `build-image` on `main` only

`build-image` becomes a matrix of amd64 on `namespace-profile-4vcpu-8ram-ubuntu-26-x64` and arm64 on `namespace-profile-2vcpu-4ram-ubuntu-26-arm`, the arm64 profile `push-image` already uses. Both build natively, so arm64 needs no emulation. `determine-version` waits for all matrix entries. Each entry supplies its `platform`, as `push-image` does at `ci-release.yml:1150-1157`, replacing the hardcoded `linux/amd64` in the build step, so an arm64 runner really checks an arm64 image.

The matrix picks its entries with `fromJSON` on `github.ref`: amd64 alone everywhere, both on `refs/heads/main`. The condition is the branch and not the `push` event, because `determine-version` also runs on a `workflow_dispatch` on `main`, and that run must not release without an arm64 check.

Rejected:

- Checking in `push-image` after the push by digest. It tests the exact published bytes, but verification images may already have version tags by then.
- arm64 on every pull request, chosen against by the user. It shows a break before the merge, but it costs a build, smoke check and scan on a 2 vCPU runner per pull request, and work from `develop` first meets the check on `main` either way.

### Image checks live in one script

`scripts/check_image.sh <image>` is the image check: it holds the smoke checks and both Trivy passes, and removes its container on exit. `build-image` and `build-dev-image` call it. This removes the second 85-line copy instead of adding a third. `shellcheck` covers it through `mise run lint-shell`, it can run locally against any image, and `scripts/check_image_test.py` runs it with stub `docker`, `curl` and `trivy`.

Rejected: a composite action. It works, but it cannot run locally, and `shellcheck` does not read the shell inside it.

The dev flow keeps its current coverage: amd64 only.

### Trivy comes from `mise.toml`

`aqua:aquasecurity/trivy` is pinned in `mise.toml`, the way `aqua:pressly/goose` is, and installed through the existing `jdx/mise-action` step with `install_args`, which is how other CI jobs get their tools. The explicit aqua backend verifies the checksum; a bare `trivy` key could resolve through another backend.

Rejected:

- `aquasecurity/setup-trivy` or `trivy-action`. Their tags were replaced with malicious commits in March 2026 (CVE-2026-33634), and they add a second update path.
- A hash-checked `curl` like gitleaks in `codeql.yml`. It would repeat in three jobs and leave developers on a different version.

The pin excludes v0.69.4, the compromised release.

### `semver` stays, pinned

`npm install -g semver@7.8.6`. Rejected: replacing it with shell arithmetic, which adds code to keep a job that holds no secrets free of one pinned dependency.

### Digests come from the multi-architecture index

Each `FROM` keeps its tag and gains `@sha256:` of the index digest, not a single-platform manifest, so both architectures resolve. `node` also moves from `24-alpine` to `24.21.0-alpine`, matching `mise.toml`. Dependabot's existing `docker` entry for `/` is the update path.

Rejected: resolving digests at build time, as `push-verification-image` does for its base. That still floats and does not satisfy the rule.

### Dead `nolint` comments are removed, not revived

Enabling `gosec` is a separate review of its findings across the backend. The reason text stays as a plain comment only where nothing else states it: `encrypting_writer.go:104` keeps its size bound, `testwait_test.go:125` keeps its note, and `unsaved_settings_test.go:367` keeps why the host key is not pinned. In `ftp.go:114` and `post_request.go:169` an adjacent comment or the function's own comment already says it.

### Workflow files get a local linter

`actionlint` is pinned in `mise.toml` and runs through a new `lint-workflows` task, part of `mise run lint`, and a CI job of the same name. A new `.github/actionlint.yaml` declares the `namespace-profile-*` labels under `self-hosted-runner.labels`, because actionlint otherwise reports each of them as an unknown label. The job skips only pull requests from `develop`, so unlike the other lint jobs it also checks pushes to `develop`, which is the path work takes to `main`, and it joins `determine-version`'s `needs`. It runs with `-shellcheck=`, so existing inline shell style does not expand the change.

Rejected: no linter. A mistake in `needs` or in caller permissions would otherwise appear only after the merge to `main`, on the path that publishes releases.

### Tests walk the `needs` graph

`scripts/workflow_jobs.py` cuts jobs out of a workflow file with the same regular expression approach as `scripts/frontend_ci_test.py:12-16`, since the standard library has no YAML parser, and `scripts/release_gate_test.py` reads each job's `needs` from it. For every publishing job it walks `needs` and asserts that the full gate set is reached, and it reads the `codeql.yml` wiring. `scripts/build_inputs_test.py` pins the digests and tool versions, `scripts/workflow_permissions_test.py` the permissions, `scripts/secret_scan_allowlist_test.py` the allowlist fingerprints, and `scripts/workflow_lint_test.py` the lint wiring. `scripts/frontend_ci_test.py:66-71` compares `determine-version`'s `needs` to an exact set, and that set is updated.

The check proposed in the original task, failing a scan on a test branch and watching `determine-version`, cannot work: that job runs only when the ref is `main` (`ci-release.yml:1057`). `act` is rejected too, because CodeQL needs GitHub and the jobs use Namespace runners. A fork has the same problem: without Namespace runners with matching profiles its jobs stay queued.

### Public claims are narrowed to what is true

Three sentences change in the README security section (`README.md:300`, `:302`, `:304`) and the website security page (`website/app/(en)/security/page.tsx:375`, `:386`, `:423`), plus five translations of each. The Dependency Review and Trivy claims in `website/public/llms.txt:64` and the Dependency Review claim at `:252` change too, which makes 13 files. "Trivy image and Dockerfile scans" at `llms.txt:252` and `website/app/(en)/page.tsx:1240` stays, because it remains true:

- Dependency Review "fails the check of any PR" instead of "blocks any PR". This stays true if the owner later makes the check required.
- Trivy scans "the application image on both architectures before a release" instead of "container images on every build", since verification images and arm64 dev images are not scanned.
- "A release only ships if all of it passes" becomes a sentence that names the security scans and states the critical threshold, so the old wording no longer appears.

"Multi-architecture builds on every PR" (`README.md:304`, `website/app/(en)/security/page.tsx:422`) stays by the owner's decision, although pull requests build amd64 only.

Rejected: leaving the three sentences above as they are. They would state a guarantee this change was written to examine and found untrue.

## Risks / Trade-offs

- [A failure in `push-image` or `push-verification-image`, after version selection, can still leave one set of images with the version tag.] → Unchanged from today and out of scope. The tag is overwritten by the next run for the same version, because `release` writes the git tag only at the end.
- [The first release after merging `develop` may stop on critical CodeQL alerts.] → Expected. `develop` is not analyzed, so the number is unknown until the first analysis under the new configuration. See Migration Plan.
- [The alert page keeps showing a fixed CodeQL alert as open until the Monday scheduled analysis, and pull requests compare against a baseline up to a week old.] → Accepted. Neither affects the severity gate, whose output names what blocks. A lower-frequency pull request comparison can only add noise to a pull request annotation.
- [A pinned base image stops picking up fixes by itself, and a new critical CVE in it stops releases until bumped.] → Dependabot proposes the digest with its 7 to 14 day cooldown, and a manual bump is a one-line commit. That Dependabot updates digests on a non-version tag such as `bookworm-slim` is not confirmed locally, so the first weekly run should be watched.
- [A semgrep or CodeQL rule update can stop a release with no code change.] → Accepted as the purpose of a gate. A semgrep failure is fixed by a commit, and a CodeQL alert is fixed or dismissed.
- [A semgrep or GitHub outage delays a release.] → Rerun the failed job. There is no bypass, on purpose.
- [A broken arm64 build shows only after the merge to `main`.] → The release stops before any tag, and the repair is an ordinary commit.
- [The smoke check allows 120 seconds for the app to start, and the arm64 runner has 2 vCPUs.] → If it is too short on the first real run, raise the limit in the script. A faster arm64 profile may not exist on the Namespace account.
- [The published image is still a rebuild of the tested one.] → Unchanged from today. Digest pins narrow the difference, while `apt-get` packages can still differ.
- [Workflow behavior cannot be fully verified locally.] → `actionlint` and the tests cover structure and script logic. The real behavior is observed on the first push to `main`, as below. A failing scan path is not exercised on purpose, because `determine-version` runs only on upstream `main`, and a fork needs matching Namespace runners.

## Migration Plan

Steps for the owner after the merge. They need GitHub access or a push, so they are not tasks:

1. Merge `develop` to `main`. The first analysis under the new configuration runs for the current code.
2. In that run, confirm that `determine-version` waited for `code-scanning` and both `build-image` entries, that `code-scanning` ran `codeql-severity-gate`, and that `gh api 'repos/databasus/databasus/code-scanning/analyses?ref=refs/heads/main'` shows the commit under `.github/workflows/ci-release.yml:analyze`.
3. If `codeql-severity-gate` fails, triage the listed critical CodeQL alerts on GitHub: fix through `develop`, or dismiss with a reason and rerun the failed job. The skipped `determine-version` is then the observed proof of the release gate.
4. If it fails because the release analysis is missing, no alert is listed, and neither a dismissal nor a rerun clears it. Read the real key from the `gh api` call in step 2, and correct it in `scripts/check_codeql_alerts.sh` with a commit.
5. Optionally make `dependency-review` a required check on `main` in the ruleset. The workflow change does not need it.

Rollback: revert the commit. `codeql.yml` then runs independently again, and the pipeline returns to today's behavior.
