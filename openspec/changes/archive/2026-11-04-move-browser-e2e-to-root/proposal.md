## Why

No test walks the path a new user depends on: become the instance admin, add a database, back it up and restore it. Each step is tested somewhere, but the browser suite against a real backend stops at creating a storage, and the current code inside the Docker image never takes a logical backup in any test. A break between the interface and the real backup tools, or one that exists only in the image, can therefore reach a release.

The current browser end-to-end suite also lives inside the frontend and backend trees and builds its application from backend internals, with a test gateway in place of the real way the frontend is served. It tests an assembly that no user runs.

## What Changes

- Add an independent browser end-to-end suite in the root `e2e/` directory. It imports nothing from the backend or the frontend and talks to the application only through its address, HTTP, browser storage and the screen.
- `mise run e2e:browser`, a hidden task that `mise run test-full` and CI call, starts everything itself. It always tests the application image, running it through a production-like compose file. The image is the one named by `DATABASUS_IMAGE` or is built from the checkout, and each run starts from empty data.
- Add the critical path scenario: register the first instance admin through the interface, sign in, add a PostgreSQL database through the wizard, wait for its first logical backup, restore it into an empty database and compare the rows.
- Move the existing sign-in and workspace resource scenarios to the new suite.
- Remove everything a run started after every outcome, and before a run remove what crashed runs left behind. Other runs stay untouched.
- CI runs the suite on the commit's candidate image, in a new check that replaces the old one and still gates releases.
- **BREAKING** for developers and agents: `pnpm --dir frontend test:browser --project=e2e` is deleted, and `mise run e2e:browser` replaces it, including in the commands the review hook accepts and in `openspec/config.yaml`.
- **BREAKING** for developers and agents: `mise run test` no longer runs the end-to-end packages of the verification agent and of backups, or the upgrade test. The new `mise run test-full` runs them, everything `mise run test` runs, frontend browser UI, script and review hook tests, and the browser suite with its unit, lifecycle and interruption checks. Podman and filesystem tests remain separate because they need additional host configuration. CI keeps its existing jobs; review hook tests and lifecycle interruption checks remain local. An agent never runs `mise run test-full` unless the user directly asks for that run: at the end of a change it runs `mise run test` and, by hand, the specific long or end-to-end tests the change touches.
- **BREAKING** for CI configuration: the `browser-e2e` test group, the `test-frontend-browser-e2e` check and its retained `status.json` artifact are deleted, and `test-e2e-browser` replaces the check. This browser path loses its race detector run, because the candidate image is not built with it.
- **BREAKING** for local commands: rename `dev` to `run` and `full` to `test-full` without aliases. Keep eight user commands at the top of `mise.toml`, hidden CI/CD tasks next, and dependencies last. Remove redundant wrappers and make `pnpm lint` the complete frontend check, with eight frontend scripts.
- One wait in a backend test stops after two minutes instead of at the package's deadline, and the package timeout of local runs goes from 30 to 15 minutes, so a hung test costs minutes. This came up while checking this change, and the rules on waiting change with it.
- Delete with no shim: `frontend/browser/e2e`, `backend/internal/app/browsertests` with its gateway, `frontend/tooling/runBrowserE2E.ts`, `runBrowserE2EInside.ts`, `runBrowserE2E.test.ts`, `removeBrowserContainer.ts` with its test, `frontend/vitest.browser-e2e.config.ts` and the `e2e` project of `frontend/playwright.config.ts`.

### Out of scope

- Moving the upgrade test out of `backend/e2e/upgrade`.
- Moving or renaming `e2e/docker-storage`, or changing `scripts/check_image.sh`.
- The API-level suites `backend/internal/modules/backups/e2e` and `backend/internal/agent/e2e`. They rely on fakes and a movable clock, so they cannot treat the application as a black box.
- Testing the published image byte for byte. Publishing rebuilds it with the version chosen after the checks, so the suite tests a candidate of the same commit and Dockerfile.
- arm64: the suite runs on the amd64 candidate.
- Changing the scenarios of the fixture-based browser suite, `pnpm --dir frontend test:browser --project=ui`; `test-full` runs that suite unchanged.
- Scenarios for other engines and physical backups.
- A local run without an image build. Every local run without `DATABASUS_IMAGE` builds the `Dockerfile`, which keeps one way of running the suite.
- A configurable server port. The suite does not need one, because the application is reached only over the run's compose network.
- A Python linter or formatter. The repository's Python files have none, and adding one is a separate decision for all of them.

## Capabilities

### New Capabilities

- `browser-e2e`: the independent browser suite on the application image, the critical path, isolation, cleanup and secrets in its output.

### Modified Capabilities

- `ci-pipeline`: the `browser-e2e` group goes away, the release-gating browser check runs on the commit's candidate image, and it no longer runs with the race detector.
- `frontend-screen-testing`: the real-backend browser suite stops being part of the frontend's testing.
- `proposal-review`: the task reviewer requires `mise run e2e:browser` in the change verification of a change to the suite's scenarios, a task's own command for other end-to-end tests, and no verification that runs `mise run test-full`.
- `implementation-review`: the change verification block accepts `mise run e2e:browser` instead of the removed browser project and rejects `mise run test-full`, and a change to the suite's scenarios must name it.
- `local-development-workflow`: `mise run test` no longer has browser packages to exclude, the suite's package manager joins the pinned tools, the user commands and hidden CI/CD tasks are separated, `pnpm lint` runs every frontend check, and `mise run test-full` covers the ordinary Docker suites while `mise run test` drops the end-to-end packages and the upgrade test.

## Impact

- This change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
- New: `e2e/browser/`, the user task `test-full`, hidden tasks `e2e:browser`, `e2e:unit` and `e2e:lifecycle`, and pinned `uv` in `mise.toml`.
- New dependencies, limited to `e2e/browser`: Python's `playwright` and `pytest`, locked with hashes, and one digest-pinned Playwright image.
- Repository checks: `.github/workflows/ci-release.yml`, `.agent-hooks/review_gate.py` and its review prompts, `scripts/test.sh`, `mise-tasks/backend/test-group-packages`, `mise-tasks/backend/test-upgrade` and their script tests, `openspec/config.yaml`.
- Documents: `AGENTS.md`, `backend/AGENTS.md`, `frontend/README.md`, `.agents/skills/testing/SKILL.md`, `docs/testing.md`, `docs/architecture.md`, `docs/glossary.md`, `docs/machine-setup.md`.
- A local e2e run needs Docker and pays for an image build, which the layer cache only shortens. CI's new check depends on the candidate image job, so it starts later than the old check did.
