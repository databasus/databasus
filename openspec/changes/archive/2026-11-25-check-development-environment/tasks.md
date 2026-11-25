# Tasks

## 1. Check module and its report

- [x] 1.1 Create `scripts/development_environment_check.py` with `MachineProbe`, `PrerequisiteCheckResult`, the report printer and the exit code rule; verify with `scripts/development_environment_check_test.py` that a fake probe with one failed required check prints every check and exits with 1, and that only warnings exit with 0.
- [x] 1.2 Add the bridge netfilter and inotify checks; verify with unit tests for a missing bridge sysctl, a bridge sysctl set to `0` whose failed line says the setting is off, the kernel default of 128 instances, whose failed line reports the current and the required value, exactly the required limits and higher limits.
- [x] 1.3 Add the Docker and Docker Compose checks with a 10-second timeout; verify with unit tests for a rootful daemon, a rootless daemon whose failed line says the tests need rootful Docker, a missing `docker` command, a non-zero `docker info` exit, a timeout after which every other check is still reported, and a missing Compose plugin whose failed line names it.
- [x] 1.4 Add the client tool bundle check over `assets/tools/<arch>/`; verify with unit tests that a `not found` line names the program and the library, that an unknown architecture fails with its name, and that clean `ldd` output passes.
- [x] 1.5 Add the pinned tools check; verify with unit tests that empty `mise ls --current --missing` output passes, that non-empty output fails with a line naming each missing tool and pointing to the locked tool installation, that a task tool pin missing from `mise ls --installed --json` fails with a line naming the tool and its task, and that a failing `mise` command fails.
- [x] 1.6 Add the required ext4, CPUs and Memory checks and the recommended Ubuntu and Commit hook checks; verify with unit tests that btrfs, 2 CPUs and a `MemTotal` of 15.6 GiB fail with lines that name the ext4 requirement and the minimums of 4 CPUs and 32 GiB, that an ext4 root with exactly 4 CPUs passes, that a `MemTotal` of exactly 30 GiB passes and 29.9 GiB fails, that Fedora warns that the tooling targets Ubuntu, and that a missing hook warns with the command that installs it.
- [x] 1.7 Verify that the module only reads state: a unit test with a recording probe shows that every command it runs is one of the read-only commands in the design table, and `mise run python:lint` passes for the new files.

## 2. Wire the check into `mise run doctor`

- [x] 2.1 Add `scripts/doctor.sh`, which runs the hook check with the forwarded `--client`, applies the libedit shim from `scripts/fedora.sh` in a temporary directory it removes on exit, without stopping on the shim's failure, runs the development environment check and exits with the hook tool check's code when that part failed and otherwise with the development environment check's code. Verify with `scripts/doctor_test.py`, which stubs both parts and covers: pass and pass; hook failure; environment failure; both failures, each printing both reports; a hook usage error whose code 2 is kept; a shim failure, after which the environment part still runs; and, with a fake `libedit.so.0`, that the environment part sees the temporary shim directory on `LD_LIBRARY_PATH` and that neither that directory nor anything in the home directory remains afterwards.
- [x] 2.2 Point the `doctor` task in `mise.toml` at `scripts/doctor.sh` and update its description; verify that `scripts/development_commands_test.py` still passes and that `mise run lint-shell` is clean.
- [x] 2.3 Run `mise run doctor` on this machine and verify that it prints the hook tools first, then one line for each of the eleven checks in the design table, and that its exit code is 1 exactly when a required line failed.

## 3. Documentation

- [x] 3.1 Update the `doctor` row of the command table and "Environment check before a change" in `AGENTS.md` to cover the development environment; verify by searching the repository for every mention of `mise run doctor` and bringing each in line.
- [x] 3.2 In `docs/how-to-setup-vm.md`, state 4 logical CPUs, 32 GiB of memory and an ext4 root as hard minimums in the CPU, Memory and Disk requirements; add the minimum host of 6 logical CPUs next to the 75% rule and point smaller hosts to a remote Ubuntu server; keep the sentence that a guest with fewer CPUs than the reference machine only runs the suites more slowly, bounded by the new minimum; and add `mise run doctor` as the first row of "Checking the VM". Verify by searching `docs/` for CPU counts, memory sizes and `ext4` and bringing each in line, and with a unit test that collects every hint anchor of the check and finds each among the guide's headings.
- [x] 3.3 Add glossary entries to `docs/glossary.md` for the hook tool check, the development environment check, and required and recommended prerequisites; verify that the specs, the design and the guide use these terms and no synonyms.

## 4. Integration

- [x] 4.1 Run `mise run python:lint`, `mise run lint-shell`, `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'` and `mise exec -- python3 -m unittest discover -s .agent-hooks -p '*_test.py'`, and verify that all pass.

## 5. Every prerequisite is required

- [x] 5.1 Make the Ubuntu and Commit hook checks fail instead of warn, and remove `Severity`, `Outcome.WARNING` and the severity branch of the result builder; verify with unit tests that Fedora fails with a line saying the tooling targets Ubuntu, that a missing hook fails with `hk install --mise`, and that the check exits with 0 only when every line passed. Replace the required and recommended prerequisite entries of `docs/glossary.md` with a definition of the prerequisite in the development environment check entry, update the "Checking the VM" row of `docs/how-to-setup-vm.md`, and verify by searching the repository for `warning` and `recommended` in the check, its tests and the documents it touches.
