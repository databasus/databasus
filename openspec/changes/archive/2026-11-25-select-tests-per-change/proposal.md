# Proposal

## Why

An agent checks every finished change with `mise run test`, which runs all 128 backend test packages outside the end-to-end ones plus every frontend test. One run takes 5 to 10 minutes, and a change usually needs three: after implementation, after review and after the fixes. The command holds a machine-wide lock, so a second agent on the same machine waits for the first one's whole run before its own starts. Most of that time goes to tests the change cannot affect.

The change `speed-up-local-backend-tests` (archived on 2026-11-02) deleted the narrow test commands because agents picked packages by judgment and missed things. This change brings selection back in a different form. The selection is no longer an ad-hoc choice made at the end. It is a written plan with the reasoning behind every included and excluded area. The proposal review checks the plan against the design, and the implementation review checks it against the diff. CI still runs every test before a release.

## What Changes

- Every OpenSpec change gets a test plan, `tests.md`, as a new artifact of a project-local OpenSpec schema forked from `spec-driven`. The plan holds:
  - an impact map: what the change touches and which tests that can break, with a reason for each;
  - deliberate exclusions with their reasons, such as engine versions whose code path the change does not reach;
  - a selection of tests written as selectors, never as shell commands.
- Selectors name what to run: a backend package with an optional test filter, a frontend path with an optional test name, or a named suite such as the browser end-to-end scenarios or the upgrade test. The runner decides how to run them: flags, parallel limits, timeouts, the image build and the order of browser runs. A selector that runs no test fails, so a mistyped filter cannot pass.
- A plan selects engine versions where the behavior under test differs, and otherwise the oldest and the newest supported version.
- A plan never selects everything. Selecting the whole backend module, one of its broad trees such as `./internal/...`, or the whole frontend source tree or one of its layers is rejected. A change to documentation alone selects nothing and says why.
- The agent edits the plan during implementation when it finds an affected area the plan missed or a selected one the change does not affect.
- New everyday command `mise run test-change <change>` runs the change's selection on the schedule of `mise run test`: each suite starts as soon as that schedule allows, and the run ends with one summary and a log in `temp/`. The command takes the same machine-wide lock as `mise run test`, so agents on one machine queue, and each run uses the whole machine.
- **BREAKING** `mise run test-full` is renamed to `mise run test`. The current `mise run test`, which runs the backend packages without the end-to-end ones and the frontend tests, is deleted together with the `--full` flag of the test runner. Nothing keeps the old names.
- **BREAKING** Only a person starts `mise run test`. An agent never runs it, even after a broad change, and a test plan or a task's verification cannot name it.
- **BREAKING** The `## Change verification` section of `tasks.md` is deleted. The review hooks check the format of `tests.md` instead and refuse a change without one. Tasks keep their own targeted verification commands for use during implementation.
- The task reviewer of the proposal review checks the plan against the design and the tasks. The tests reviewer of the implementation review checks it against the diff.
- Repeating one failing test with the Go test command or the frontend runner stays available and takes no lock.

The change answers to the root `AGENTS.md`. It edits `backend/AGENTS.md`, whose command table names the test commands, but changes no backend code, so no backend code rule applies to it. It changes no frontend or website code.

## Capabilities

### New Capabilities

- `change-test-plans`: the test plan every OpenSpec change carries, the selector format, the version rule, how the plan changes during implementation and how `mise run test-change` runs it.

### Modified Capabilities

- `local-development-workflow`: the everyday command list changes; `mise run test-full` becomes `mise run test`; the routine run and its separate frontend requirement are removed; the lock, interruption, slow-package and timing requirements name the new commands; an agent no longer runs the whole test set.
- `proposal-review`: the format check before the reviewers checks `tests.md` instead of the "Change verification" section, and the task reviewer checks the plan.
- `implementation-review`: the hook validates `tests.md` instead of the "Change verification" section, and the tests reviewer checks the plan against the diff.
- `managed-postgresql-backup-tests`: the Neon backup test belongs to the logical end-to-end tests that the renamed `mise run test` runs and a test plan can select; `mise run test` and `mise run test-change` provide its setting.
- `managed-object-storage-tests`: the renamed `mise run test` and the new `mise run test-change` provide the Neon Object Storage settings, in place of `mise run test-full`.

## Out of scope

- CI. Its groups, shards and the race detector stay as they are, and CI still runs every test.
- Choosing tests automatically from the import graph. A measurement showed it is too coarse here: 31 of 128 test packages depend on the notifiers module and 37 on the MySQL engine, because module tests build the whole application. It also cannot see migrations, the API contract, client tool bundles or the frontend. The planning agent may use it as a hint.
- An exception from the lock for plans that start no containers, such as a frontend-only plan. It can be added later if waiting becomes a problem.
- Changing the tests themselves, their isolation or their parallel limits.
- The Podman subset and the filesystem suite, which keep their separate commands and are not selectable.
- Changes already archived. Their `tasks.md` files keep the old section.

## Impact

- `openspec/schemas/` gains the forked schema with the `tests` artifact and its template. `openspec/config.yaml` selects it, gets rules for the new artifact and loses the rule about the "Change verification" section.
- `scripts/test.sh` loses its routine mode and gains the plan mode; `mise.toml` renames the tasks; the script tests in `scripts/test_sh_test.py`, `scripts/development_commands_test.py` and `scripts/frontend_ci_test.py` follow.
- `.agent-hooks/review_gate.py`, `proposal-prompt.md`, `implementation-prompt.md` and their tests switch to `tests.md`.
- Documentation: `AGENTS.md`, `backend/AGENTS.md`, `backend/README.md`, `docs/testing.md`, `docs/how-to-setup-vm.md` and the testing skill.
- Workflow: an agent that checked its work with `mise run test` now writes and runs a plan. A person who ran `mise run test-full` now runs `mise run test`.
