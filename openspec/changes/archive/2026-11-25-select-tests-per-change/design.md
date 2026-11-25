# Design

## Context

See proposal.md for the motivation. The current state that shapes the approach:

- `scripts/test.sh` is one runner with two modes. The routine mode excludes the end-to-end packages (`scripts/test.sh:134`); `--full` keeps them and adds the script, hook, browser and upgrade suites (`scripts/test.sh:135-136`, `:257` onward). Both modes take one machine-wide `flock` (`scripts/test.sh:52`, `:113-115`) and build the image under `databasus-full-test:local` (`scripts/test.sh:59`).
- The runner owns measured limits. The slow packages from `backend/test-slow-packages.txt` run with `-p=8 -parallel=8`, because Go's default started dozens of Patroni clusters at once and the group failed (`scripts/test.sh:224-240`). The slow group gets 30 minutes in the full run and 15 in the routine run (`scripts/test.sh:240-242`).
- The review hook reads the `## Change verification` section of `tasks.md` and accepts three fixed commands (`.agent-hooks/review_gate.py:263-272`, `:319`). It validates the section before every full review (`.agent-hooks/review_gate.py:1270-1279`). A full proposal review starts only when `openspec status` reports `isPlanningComplete` (`.agent-hooks/review_gate.py:912-930`).
- The project uses the package schema `spec-driven` (`openspec/config.yaml:1`), whose artifacts are proposal, specs, design and tasks, with apply requiring tasks.
- The `-run` filter already selects engine versions, because version loops are subtests. Their names differ between packages: `postgres:17` in `internal/engines/postgresql/logical/source_test.go:13-21`, `PostgreSQL 17` in `internal/modules/backups/e2e/postgresql/engine_postgresql_test.go:41-47`.
- Direct runs take no lock: `mise-tasks/agent/e2e` calls `go test` itself.

The change answers to the root `AGENTS.md`: Tests, Documentation consistency, Naming, Code is split into logical units, and the OpenSpec section. It edits the command table of `backend/AGENTS.md` and no backend code. `docs/architecture.md` describes the application, not the development tooling, so it does not change.

## Goals / Non-Goals

**Goals:**

- One runner and one schedule for both commands, so that a plan run behaves like a slice of the full run: same limits, same ordering of browser runs, same cleanup, same summary.
- One parser of the plan's selection, used by the runner and the hook, so the accepted grammar cannot drift between them.
- The change checks itself through the new mechanism before its review ends.

**Non-Goals:**

- A runtime guard that stops an agent from starting `mise run test`. The rule lives in `AGENTS.md` and the reviewers check task verifications.
- Changing the tests to make version filters uniform.

## Decisions

### 1. The test plan is an artifact of a forked schema, after the tasks

`openspec schema fork spec-driven databasus` copies the package schema into `openspec/schemas/databasus/`. The fork adds the `tests` artifact, which generates `tests.md`, requires `tasks` and has its own template and instruction. `apply.requires` becomes `[tasks, tests]`, and `openspec/config.yaml` switches to `schema: databasus`. A comment at the top of `schema.yaml` names the base, `spec-driven` of OpenSpec 1.17.0, and the places that differ: the name and description, the `tests` artifact and `apply.requires`, so an OpenSpec upgrade knows what to compare.

The plan comes after the tasks because the impact map is drawn from what the tasks touch. The instruction tells the agent to read `docs/testing.md#test-plans`, which owns the format and the selection rules, rather than restating them.

Rejected alternatives:

- *Extending `## Change verification` in `tasks.md`.* It needs no fork, but the plan would get no instruction or template of its own and no status in `openspec status`, and a long impact map would sit among the task checkboxes. The user chose the fork.
- *A separate agent that writes the plan.* It would reread the proposal, design, tasks and code that the planning agent already holds. The task reviewer of the proposal review already gives the plan an independent reading.
- *The plan before the tasks.* The impact map would then guess which files the tasks will change.

### 2. The selection holds selectors, and the runner turns them into commands

The selection is the first block fenced as `tests` under `## Selection`. Each line is split like a shell word list (`shlex`), never run through a shell:

| Selector | Meaning | Constraints |
|---|---|---|
| `backend <pattern> [-run <regex>]` | Go packages of the backend module | the pattern starts with `./` and contains no `..`; it is none of `./...`, `./internal/...`, `./internal/modules/...`, `./internal/engines/...`, `./internal/lib/...`; and it cannot reach `./e2e/upgrade`, so `./e2e/...`, `./e2e/upgrade` and anything under it are refused |
| `frontend <path> [-t <name>]` | frontend test files under a path | the path is relative to `frontend/`, starts with `src/` and contains no `..`; it is neither `src` nor a whole layer: `src/pages`, `src/widgets`, `src/features`, `src/entities`, `src/shared` |
| `suite <name>` | a named suite | `browser-ui`, `e2e-browser`, `e2e-unit`, `lifecycle`, `lifecycle-interruption-checks`, `upgrade`, `scripts`, `review-hooks` |
| `none` | nothing to run | the only line of the block |

The refused backend patterns are an explicit list rather than a depth rule. `./internal/...` alone holds 197 of the module's 205 packages, and the four trees under it hold nearly all module and engine tests, while a shallow pattern such as `./cmd/...` holds four packages and stays allowed. `./internal/app/...` and `./internal/agent/...` stay allowed too, because a change to the wiring or to the verification agent can need them whole.

Suite names are the names the run's summary already reports (`scripts/test.sh:265-317`), so the plan and the summary use one word for one suite. The one exception is `lifecycle`, which stands for the crashed-run probes, `lifecycle-crashed-runs`, and the parts of the other probes, `lifecycle-<n>-of-7`, because they are one suite split for scheduling.

The hook checks only the form. A path the change has not created yet is accepted; the runner fails it when it matches nothing.

Rejected alternatives:

- *Shell commands checked by prefix.* The agent would choose `-parallel` and timeouts, and the measured Patroni failure at Go's default limit (`scripts/test.sh:224-236`) would come back in the first plan that names that package. A prefix check also lets arbitrary arguments through.
- *A YAML or JSON file.* A fenced block keeps the selection next to the reasoning that justifies it, and the hook already parses fenced blocks.

### 3. One parser in `scripts/test_plan.py`

`scripts/test_plan.py` parses and validates the selection. It has two entry points: `check <change directory>`, which the hook runs and which prints the reason and the allowed forms on failure, and `list <change directory>`, which prints the selectors as normalized lines for `scripts/test.sh`. The hook calls it as a subprocess, as it already calls `openspec`. Its tests live in `scripts/test_plan_test.py`.

Rejected alternatives:

- *A parser in the hook and another in bash.* Two grammars drift, and a selector one accepts and the other refuses would block a change only at run time.
- *The hook importing the module from `scripts/`.* The hooks run under their own launcher (`.agent-hooks/run.sh`), and a subprocess keeps that boundary.

### 4. The full run is the complete selection of the same schedule

`scripts/test.sh` builds a selection and schedules it. Without arguments the selection is complete: every backend package except the upgrade packages, the whole frontend and every suite. With `--change <name>` it comes from `scripts/test_plan.py list`. Everything after that is shared:

- Each backend selector is expanded with `go list` before anything starts. A selector that matches no package fails at once and names itself, because `go list` only warns and exits with zero in that case. A selector whose `go list` fails, for example on a package that does not load, fails the same way instead of running the packages it did list.
- Backend selectors without `-run` are split by `backend/test-slow-packages.txt` into the slow and the other process, as today. A selector with `-run` runs as its own process, with the slow limits when one of its packages is on the list. When two selectors overlap, the shared tests run once for each, so that each filter is checked on its own. The full run's timeouts apply in both modes, because a plan can hold the same end-to-end packages the full run does.
- Frontend selectors without `-t` share one `pnpm --dir frontend test <paths>`; one with `-t` runs alone.
- Suites keep the full run's order: the crashed-run probes, then the interruption checks, then the browser scenarios and the other lifecycle probes. The image is built once when `upgrade` or `e2e-browser` is selected.
- The shared test PostgreSQL server starts and the test images are pulled only when a backend selector is present. Frontend dependencies are installed only when a frontend selector or `browser-ui` is present. The browser project's Python environment is prepared with `uv sync --locked --project e2e/browser` only when `scripts` is present, because the script tests import its locked dependencies (`docs/testing.md:449-454`).
- Backend processes run with `go test -json`. A new `scripts/go_test_report.py` reads that stream, prints the usual text output, so the console and the log stay readable, and records each test that passed or failed with its package. After the backend processes end, the runner fails every selector whose packages recorded no test at the depth of its `-run` filter: one level without a filter, two for `TestA/sub`, and so on. The record also keeps skipped tests: a selector whose tests at its depth only skipped, such as the Neon tests without their settings, is reported as skipped and does not fail, and the report prints each skip with its reason. Unfiltered selectors share one record, because they run every test of their packages; each filtered selector keeps its own, so tests another selector ran in the same package do not count for it. A selector reports a failure under its own name in the summary. Go's `no tests to run` warning is not enough: it appears only when no top-level test matched, so `-run '/^postgres:17$'` on a package that names its versions `PostgreSQL 17` runs the top-level test, skips every version and passes silently. The warning also appears for one package of a multi-package selector that did run tests in another.
- A frontend selector whose path does not exist fails before anything starts. Vitest matches a path argument as a substring of a file's path, so a directory goes to Vitest with its trailing slash, and `src/features/edit` cannot run `src/features/edit-notifier`. Frontend runs add Vitest's JSON reporter next to the default one, and the runner fails every frontend selector under whose directory, or at whose file, no test passed or failed. A skip does not count here, because Vitest reports a test that `-t` filtered out as skipped.

`none` prints that the plan selects no tests and exits with zero before taking the lock. The log goes to `temp/test-change-<change>-<time>.txt` or `temp/test-<time>.txt`. The old routine branch and `--full` are deleted.

Rejected alternatives:

- *A separate plan runner.* It would duplicate the lock, the cleanup trap, the summary and the browser ordering, which already took several changes to get right.
- *The routine run's 15-minute slow timeout in plan mode.* A plan with the CloudNativePG package beside a browser run hits the same contention the full run does.

### 5. Plan runs share the full run's lock and image tag

`mise run test-change` takes the lock at `scripts/test.sh:52`, so agents queue and each run has the whole machine. Because only lock holders use the run's image tag, the plan run reuses it safely. A direct `go test` or `pnpm test` repeat takes no lock, as today.

The lock and the tag now serve every queued run, not only the full one, so they are renamed: `databasus-full-test.lock` becomes `databasus-test-run.lock`, `full_run_lock` becomes `test_run_lock`, and `databasus-full-test:local` becomes `databasus-test-run:local`. A run of the old code that still holds the old lock file does not block a new run during the switch; with one active change and the agents of this repository, that overlap is short and harmless.

Rejected alternatives:

- *No lock for plans.* Two plans with the Patroni or CloudNativePG packages would exceed the memory budget. The user prefers queuing.
- *A weighted semaphore.* There is no measurement to weight by, and the user accepted the queue.

### 6. The hook requires a plan for every active change it reviews

`parse_change_verification` and `VERIFICATION_FORMS` are deleted. Before a full proposal or implementation review, the hook runs `scripts/test_plan.py check` for every target change. A missing `tests.md` blocks the turn. A change on the project schema cannot reach the proposal review without the plan anyway, because `isPlanningComplete` stays false; the check matters for a change still on `spec-driven`, such as this one. The prompts replace their "Change verification" paragraphs with the plan checks of the delta specs.

### 7. The change checks itself through its own plan

This change is created on `spec-driven` and reviewed by the current hook, whose section rules it satisfies without a section, because it adds no backend or frontend tests. Once the fork and the new hook exist, a task switches the change's `.openspec.yaml` to `schema: databasus` and writes its `tests.md`, selecting `suite scripts` and `suite review-hooks`. The final run is `mise run test-change select-tests-per-change`, and the implementation review then runs on the new hook.

### 8. Engine versions are chosen through `-run`, with names read from the test

The rule in `change-test-plans` is applied through ordinary `-run` filters. `docs/testing.md` tells the author to read the version subtest names in the package, because they differ between packages, and the runner's count of tests at the filter's depth catches a wrong name.

Rejected alternative: *an environment variable such as `DATABASUS_TEST_ENGINE_VERSIONS` read by every version list.* It touches every engine and backup test, adds a second filter beside `-run`, and the backend reads the environment only in `lib/config`.

### 9. Where the rules live

- `docs/testing.md` gets a "Test plans" section with the format, the selector table, the version rule, the end-to-end rules and the run. "Checking work", "Parallelism in local runs" and "Commands" follow the renamed commands.
- `AGENTS.md` Tests says that an agent checks a change with `mise run test-change` and never runs `mise run test`. The command table and the "Several agents, one machine" paragraph follow.
- `openspec/config.yaml` drops the `tasks` rule about the "Change verification" section, gains rules for the `tests` artifact and an apply guidance line to keep the plan matching the diff.
- The testing skill's "Which tests to run" points to the test plan. `docs/glossary.md` defines "Test plan", including its impact map and selection, and "Selector" in its Code section.

## Risks / Trade-offs

- [A plan misses an affected test] → The task reviewer checks the plan against the design and tasks, the tests reviewer checks it against the diff, and CI runs every test before a release. A miss costs a red CI run, not a broken release.
- [A mistyped filter runs nothing and passes] → The runner counts executed tests per selector at the depth of its filter and fails a selector with none.
- [A small plan waits behind a long one] → Accepted. An exception for plans that start no containers can come later.
- [The fork drifts from OpenSpec's `spec-driven`] → The header comment names the base version and the two differences, so an upgrade compares them.
- [Agents still run `mise run test` by habit] → `AGENTS.md`, the reviewers and the hook all reject it in plans and task verifications.
- [Breaking rename for people] → `mise run test-full` disappears with no alias, as the repository rules require. `mise run test` now takes as long as the old full run.

## Migration Plan

All edits land in one commit. No change other than this one is active, and archived changes keep their sections. Rolling back means reverting the commit.
