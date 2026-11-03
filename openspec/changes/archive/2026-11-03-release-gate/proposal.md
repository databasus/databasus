## Why

A new version can ship while a security scan is red. Secret scanning, rule-based scanning and static analysis (CodeQL) run in a separate workflow that the release does not wait for, and the static analysis never fails on what it reports. The arm64 application image is published without a smoke check or a vulnerability scan, so a broken arm64 build reaches users. Base images and scan tools float, which breaks the project's own security rules and lets a build silently pick up different content.

## What Changes

- The release waits for secret scanning, rule-based scanning and static analysis. A failure in any of them stops version selection and everything after it.
- Open critical CodeQL alerts stop the release. High CodeQL alerts show as a warning. A CodeQL alert the owner dismissed with a reason does not count.
- The secret scan that gates a release reads only the history of the released commit, so an unrelated branch cannot stop a release. The weekly scheduled scan keeps reading all branches.
- The application image gets its smoke checks and vulnerability scan on amd64 in each run outside `develop` and pull requests from it, and also on arm64 in every run on `main`, before anything is published.
- A failed application image check or verification image build stops version selection, so no image gets a version tag because of those failures. A failure in the rebuild after version selection can still leave partial tags, as it can today.
- All four base image references in the root Dockerfile are pinned by digest. The vulnerability scanner and the `semver` tool used for version selection are pinned to exact versions.
- Two test-only secret scan findings from the history of `develop` go into the secret scan's allowlist. Without this the first gated release would stop on them.
- The five dead `nolint` comments for disabled linters are removed.
- `github.com/getkin/kin-openapi` moves from v0.142.0 to v0.144.0. The image check found its critical vulnerability GHSA-r277-6w6q-xmqw, which would stop every release, and the owner approved the update.
- `pull-requests: write` moves from the top level of the dependency review workflow to its job.
- GitHub Actions workflow files gain a local linter, because these workflows cannot be run locally.
- The README and website security sections stop claiming more than the pipeline does: the dependency review fails a pull request's check, and the Trivy scan covers the application image on both architectures.

## Out of scope

- Image signing and SBOM, which are a separate owner decision.
- Triage of the open CodeQL alerts. It is manual work on GitHub (see design.md, Migration Plan).
- Scans, smoke checks and base image pinning for verification images, whose base keeps resolving at build time.
- Smoke checks and scans for the arm64 dev image.
- The rule in version selection that treats any `!` in a commit subject as a breaking change.
- Making the dependency review a required check. That is a repository setting only the owner can change.
- Partial tags from a failed rebuild after version selection.
- Pinning packages installed with `apt-get` in the Dockerfile.
- The wording "multi-architecture builds on every PR" in the README and on the website, which stays as it is by the owner's decision.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: adds requirements for security scans gating a release, CodeQL alerts by severity, the history a gating secret scan reads, image checks on every architecture, version selection waiting for the image checks, pinned build inputs, per-job workflow permissions, and linting of workflow files.

## Impact

- **BREAKING for the release process, not for users or APIs:** a commit on `main` with an open critical CodeQL alert, a secret scan finding or a rule-based scan failure no longer releases. Nothing is kept as a bypass; the old independent scan path on push to `main` is deleted.
- Workflows: `.github/workflows/ci-release.yml`, `.github/workflows/codeql.yml`, `.github/workflows/dependency-review.yml`.
- Build: `Dockerfile`, `mise.toml` (Trivy, `gh`, `jq`, `actionlint`), `.github/actionlint.yaml`, `.gitleaksignore`.
- New scripts and tests: `scripts/check_image.sh`, `scripts/check_image_test.py`, `scripts/check_codeql_alerts.sh`, `scripts/check_codeql_alerts_test.py`, `scripts/release_gate_test.py`, `scripts/build_inputs_test.py`, `scripts/workflow_permissions_test.py`, `scripts/secret_scan_allowlist_test.py`, `scripts/workflow_lint_test.py`, `scripts/workflow_jobs.py`; `scripts/frontend_ci_test.py` is updated.
- The backend test of a database deletion racing a restore start, which failed when the restore ended before the deletion read it, and now holds the restore in progress.
- Backend comments in five Go files, and `backend/go.mod` and `backend/go.sum` for the `kin-openapi` update.
- Documentation, 13 files in all: `README.md`, five `assets/readme/README.<locale>.md`, `website/app/(en)/security/page.tsx`, five `website/app/[lang]/security/content/<locale>.tsx`, and `website/public/llms.txt`. `docs/glossary.md` gains the terms release gate, release analysis, severity gate, image check, smoke check, CodeQL alert and secret scan finding.
- CI cost: one extra arm64 image build, smoke check and scan on every run on `main`.
- This change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`.
