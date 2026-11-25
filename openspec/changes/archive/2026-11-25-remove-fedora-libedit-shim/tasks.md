# Tasks

## 1. Remove the alias step from the scripts and their tests

- [x] 1.1 Remove the `source` line and the `apply_libedit_shim` call from `scripts/dev.sh`, `scripts/test.sh`, `mise-tasks/agent/e2e` and `mise-tasks/agent/e2e-podman`, together with their `# shellcheck source=` comments, and verify with `shellcheck scripts/dev.sh scripts/test.sh mise-tasks/agent/e2e mise-tasks/agent/e2e-podman`
- [x] 1.2 Remove the temporary alias directory, the `source` line and the call from `scripts/doctor.sh`, and rewrite its header comment so it no longer mentions the shim; verify with `shellcheck scripts/doctor.sh`
- [x] 1.3 Delete `scripts/fedora.sh` and `scripts/fedora_test.py`, and verify that `rg -n 'fedora\.sh|apply_libedit_shim|LIBEDIT_SHIM_DIR' --hidden -g '!.git' -g '!openspec/changes/archive'` finds nothing outside this change, which confirms that `mise run test` and `mise run doctor` create no alias directory and leave `LD_LIBRARY_PATH` as the caller set it
- [x] 1.4 Remove the stubs from `scripts/doctor_test.py`, `scripts/test_sh_test.py`, `scripts/test_group_test.py` and `.agent-hooks/review_launch_test.py`, delete the two shim tests and the fake `ldconfig` from `scripts/doctor_test.py` and trim its fake environment check to print only its report line, and verify with `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'` and `mise exec -- python3 -m unittest discover -s .agent-hooks -p '*_test.py'`
- [x] 1.5 Verify that the client tool bundles check passes without the alias step by running `mise run doctor`, which exits with code 0 and prints `Client tool bundles: passed`

## 2. Update the documentation

- [x] 2.1 Remove the `source scripts/fedora.sh` and `apply_libedit_shim` lines from the four command blocks of `docs/testing.md`, the paragraph about the shim under Commands and the references to the shim around them, and verify that `rg -n -i 'libedit|shim|fedora' docs/testing.md` finds nothing
- [x] 2.2 Remove the sentence about the Fedora library name and the run and test commands from `assets/tools/README.md`, keeping the note that `libedit.so.2` reaches the image through `postgresql-common`, and verify that `rg -n -i 'fedora|shim' assets/tools/README.md` finds nothing
- [x] 2.3 Run `mise run lint` and verify that it passes

## Workflow follow-up

- At archive, change the Purpose line of `openspec/specs/local-development-workflow/spec.md` from "a local Fedora host" to "the Ubuntu development VM". Delta specs cannot change a Purpose.
