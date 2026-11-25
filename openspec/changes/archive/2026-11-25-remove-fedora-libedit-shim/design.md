# Design

## Context

`scripts/fedora.sh:6-25` defines `apply_libedit_shim`. When `ldconfig -p` lists no `libedit.so.2`, the function links `libedit.so.0` under that name in `~/.local/lib/databasus-compat`, or in `LIBEDIT_SHIM_DIR` when that is set, and prepends the directory to `LD_LIBRARY_PATH`. When `libedit.so.0` is missing too, it fails with a `dnf install` hint.

Five entry points source and call it:

- `scripts/dev.sh:5,47` for `mise run run`
- `scripts/test.sh:5,127` for `mise run test` and `mise run test-full`
- `scripts/doctor.sh:17-21` for `mise run doctor`, which points `LIBEDIT_SHIM_DIR` at a temporary directory and removes it on exit
- `mise-tasks/agent/e2e:7-9`
- `mise-tasks/agent/e2e-podman:32-34`

The client tool bundles check in `scripts/development_environment_check.py` has no alias logic of its own. It inherits `LD_LIBRARY_PATH` from `scripts/doctor.sh`. Once the shim call is gone, the check sees exactly what the system provides, with no change to the Python code.

Four test files stub the function so that copies of the scripts can source it: `scripts/doctor_test.py:19-21,24-29,58-61`, `scripts/test_sh_test.py:178-182,214`, `scripts/test_group_test.py:258` and `.agent-hooks/review_launch_test.py:17`. The stub in `scripts/test_group_test.py` is already dead, because neither task it copies sources the file.

CI does not use the shim. Its jobs install `libedit2` with `apt-get` (`.github/workflows/ci-release.yml:554`).

The change answers to the root `AGENTS.md`: no backward-compatibility shims, documentation consistency and comments only for a *why*. It touches no code governed by a module `AGENTS.md`, and it does not change `docs/architecture.md`.

## Goals / Non-Goals

**Goals:**

- No script, task, test or document of the development tooling refers to the alias step or to `scripts/fedora.sh`.
- `mise run doctor` reports a missing `libedit.so.2` like any other missing library.

**Non-Goals:**

- Changing which system libraries the VM setup installs.
- Changing how `scripts/development_environment_check.py` finds the libraries a program needs.

## Decisions

### Delete the alias step outright

The function and its file are deleted, and every caller loses its two lines.

Rejected alternatives:

- Keeping `apply_libedit_shim` because it does nothing on Ubuntu. A harmless step still costs a source line in five scripts, stubs in four tests and four doc blocks. It also suggests that Fedora is supported, while `mise run doctor` rejects it.
- Moving the alias logic into the Python environment check. It would keep the same dead code in a different place.

### `mise run test` gets no library pre-check of its own

The removed function also stopped `mise run test` early with an install hint when `libedit` was missing. That check is not replaced. `mise run doctor` already checks every bundled program's libraries and names the missing ones, and the VM setup guide installs `libedit2`.

Rejected alternative: an `ldconfig` probe at the start of `scripts/test.sh`. It would repeat one item of the doctor check for one library and drift from the full check.

### Scripts that only test the stubs lose those cases

`scripts/doctor_test.py` keeps its test of the combined exit code and the verdict line. It drops `test_runs_the_environment_check_after_the_libedit_shim_fails` and `test_runs_the_environment_check_with_a_temporary_shim_it_removes_afterwards`, because the behavior they pin no longer exists. The fake `ldconfig` that only served the shim goes as well, and the fake environment check at `scripts/doctor_test.py:19-21` stops computing a shim directory and printing whether it holds `libedit.so.2`. No test replaces them: with the shim code deleted, nothing in `scripts/doctor.sh` changes `LD_LIBRARY_PATH`, and the search in task 1.3 confirms that no caller sources the alias step.

## Risks / Trade-offs

- [A developer runs the tooling on a Fedora machine without the VM] → Tests that start a bundled client fail on the missing `libedit.so.2`. `mise run doctor` names the cause: the system is not Ubuntu, and the client tool bundles check lists the library. The VM setup guide is the documented path.
- [A user's home directory keeps `~/.local/lib/databasus-compat` from earlier runs on Fedora] → Nothing reads the directory any more, and it holds one symlink. It is left alone, because the tooling does not clean files outside the repository that it no longer manages.
