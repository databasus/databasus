# Design

## Context

See proposal.md for the motivation; the specs state what the check verifies.

Current state:

- The `doctor` task in `mise.toml:26-29` runs `.agent-hooks/run.sh doctor`, which starts `check_environment` in `.agent-hooks/review_gate.py:1649`. That function, the hook tool check, prints the review client and the path of Python, git, OpenSpec and the client CLI. For Codex it also checks the project hooks. It returns 1 when any of them fails.
- The hook tests call `review_gate.py doctor` directly (`.agent-hooks/review_gate_test.py:399`, `.agent-hooks/review_launch_test.py:12`). They run locally under `mise run test-full` (`scripts/test.sh:271`) on whatever machine the developer uses. CI runs only the script tests, `python3 -m unittest discover -s scripts -p '*_test.py'` (`.github/workflows/ci-release.yml:156`), on runners without the VM setup.
- `mise run run` and `mise run test` source `scripts/fedora.sh` and call `apply_libedit_shim` before they start the services and tests that use the client tool bundles (`scripts/dev.sh:47`, `scripts/test.sh:127`). On a host without `libedit.so.2` four programs of the bundles link it: three `mariadb` builds and `psql` of PostgreSQL 12. With the shim applied, `ldd` finds every library on this Fedora host. The shim returns 1 when the host has no `libedit` at all (`scripts/fedora.sh:17-20`).
- `AGENTS.md`, "Environment check before a change", makes an agent stop on a non-zero `doctor` exit before any OpenSpec work.

## Goals / Non-Goals

**Goals:**

- One command reports both the hook tools and the development environment, in one run, even when the first part fails.
- Each check is testable without the real machine state, so the new script tests pass in CI on runners that do not meet the requirements.
- The check reads state only and finishes in a few seconds.

**Non-Goals:**

- A general system inventory. The check covers what the specs list and nothing else.
- Running the check from the review hooks. Hooks keep checking only their own tools.

## Decisions

### The development environment check is a separate Python module under `scripts/`, not part of `review_gate.py`

`scripts/development_environment_check.py` holds the checks, and `scripts/development_environment_check_test.py` tests them.

The hook tests run under `mise run test-full` on any developer machine, including one that does not meet the VM requirements. If the checks lived in `check_environment`, every hook doctor test would depend on the real machine state or need a fake for every probe inside the hook module. A separate module keeps the hook tests as they are, and its own tests run both locally and in CI. It also keeps the project's runtime environment out of the review tooling, which `review_gate.py` already spends about 1700 lines on.

- **Rejected: adding the checks to `check_environment` in `review_gate.py`**, for the reasons above.
- **Rejected: a separate task such as `mise run doctor:vm`.** The OpenSpec preflight runs only `mise run doctor`, and a second command would be forgotten. `local-development-workflow` also fixes the everyday task list at nine.

### The `doctor` task runs both parts through a small shell script

`scripts/doctor.sh` does the following:

1. Runs `.agent-hooks/run.sh doctor` with the forwarded `--client` and remembers its exit code.
2. Sources `scripts/fedora.sh` and calls `apply_libedit_shim` with `LIBEDIT_SHIM_DIR` set to a temporary directory that a trap removes on exit, so the check leaves no symlink in the home directory. A failure of the shim does not stop the script. The check then runs without the shim and reports `libedit.so.2` as missing for each program that needs it, which fails the check.
3. Runs `python3 -I scripts/development_environment_check.py` and remembers its exit code.
4. Exits with the hook tool check's code when that part failed, so a usage error such as an unknown client keeps its code 2. Otherwise it exits with the development environment check's code.

The script does not use `set -e`, so no failure skips a later step. `.agent-hooks/review_launch_test.py` runs the real `doctor` task in a fixture repository to check how it forwards the client; that test copies `scripts/doctor.sh` and stubs the shim and the development environment check, so it stays independent of the machine. The `doctor` task in `mise.toml` calls this script, and its description names both parts. The launcher's doctor action keeps covering the hook tools only.

- **Rejected: a `run` list in `mise.toml`.** mise stops a task at the first failing command, so a missing reviewer CLI would hide the environment report. The spec requires both reports in one run.
- **Rejected: applying the shim inside Python.** The shim lives in `scripts/fedora.sh`, and the run and test commands use it from there. A second implementation in Python could drift from it, and the spec requires the check to see libraries the way those commands do.

### Each check is a function over a `MachineProbe`

`MachineProbe` holds the reads the checks need: file contents under `/proc` and `/etc`, command output with a timeout, the CPU count, the total memory and the repository root. The real probe reads the machine. Tests pass a fake probe with prepared values.

Every check returns a `PrerequisiteCheckResult` with its name, its outcome (passed or failed) and a hint with the anchor of a section in `docs/how-to-setup-vm.md`. The module prints one line per result and exits with 1 when any check failed.

- **Rejected: patching `subprocess` and `open` in tests.** Patches couple the tests to how each check reads, and a missed patch would read the real machine and make the tests depend on the runner.

### How each check reads the machine

| Check | Reads | Fails when |
|---|---|---|
| Bridge netfilter | `/proc/sys/net/bridge/bridge-nf-call-iptables` | the file is missing, or its value is not `1` |
| inotify | `/proc/sys/fs/inotify/max_user_instances` and `max_user_watches` | below 1024 or 1048576 |
| Docker | `docker info --format '{{json .}}'` with a 10-second timeout | no `docker` command, a timeout, a non-zero exit, or `name=rootless` in `SecurityOptions` |
| Docker Compose | `docker compose version` with a 10-second timeout | a non-zero exit or a timeout |
| Client tool bundles | `ldd` on every executable file under `assets/tools/<arch>/`, where `<arch>` is `x64` for `x86_64` and `arm` for `aarch64` | any `not found` line; the result lists program and library. An unknown architecture fails with its name |
| Pinned tools | `mise ls --current --missing` in the repository root, and the `tools` of each task from `mise tasks --hidden --json` against `mise ls --installed --json` | a top-level or task tool pin is not installed, or `mise` fails |
| Ubuntu | `ID` in `/etc/os-release` | not `ubuntu` |
| ext4 | `findmnt -no FSTYPE /` | not `ext4` |
| CPUs | `os.cpu_count()` | below 4 |
| Memory | `MemTotal` in `/proc/meminfo` | below 30 GiB |
| Commit hook | `git config --get-all hook.hk-pre-commit.command`, then the file at `git rev-parse --git-path hooks/pre-commit` | neither runs `hk run pre-commit --from-hook` |

A missing command, such as `docker` or `findmnt`, gives its check the failed outcome with the command's name instead of raising.

hk installs the commit hook in one of two forms. On Git 2.54 and later it writes a `hook.hk-pre-commit.command` entry to git config and leaves `.git/hooks/` untouched; on older Git, such as the 2.43 of Ubuntu 24.04, it writes a `pre-commit` script. The check accepts either. `--git-path` resolves `core.hooksPath` and the common directory of a worktree.

`ldd` exits with 1 and prints `not a dynamic executable` for a statically linked program. That program needs no shared library, so the check counts it as passing.

The bridge netfilter check reads the sysctl the module creates instead of `lsmod`, because the module can also be built into the kernel, as in some WSL kernels. Reading the value also catches a loaded module whose setting someone turned off.

### Which prerequisites the check covers

The check covers what makes a project command or test fail, the floor at which a run is practical, and conformity with the reference environment. The floor is an ext4 root, 4 logical CPUs and 32 GiB of memory. The floor is not the reference machine of `docs/testing.md`, which has 16 logical CPUs. A guest between the floor and the reference machine passes and runs the suites more slowly. Memory has no such slack, because the tests take about 32 GB whatever the CPU count. Conformity covers Ubuntu as the distribution and the installed commit hook. Another distribution can run the tests, but it drifts from the system CI runs and the tooling targets, and a failure there is then hard to tell from a real one. Without the hook, the format and lint check moves from commit time to review.

The guide keeps the 75% rule for the host and adds the hard minimums: the VM gets at least 4 logical CPUs, at least 32 GiB of memory and an ext4 root. Both CPU rules hold on a host with at least 6 logical CPUs, so the guide states that as the minimum host size.

- **Rejected: warnings for some prerequisites.** A warning is easy to ignore. On btrfs, on fewer than 4 CPUs or with less than 32 GiB, runs become slow or run out of memory, and the failure then shows up minutes into a run instead of before work starts. A missed hook or another distribution likewise surfaces only later, at review or as a failure that does not reproduce in CI.
- **Rejected: the reference machine's 16 logical CPUs as the minimum.** It would shut out ordinary developer machines whose guests run the suites correctly, only more slowly.

The memory threshold is 30 GiB of `MemTotal`. A guest given 32 GiB reports about 31.3 GiB after the kernel's reservation, so a 2 GiB margin keeps that guest from failing.

## Risks / Trade-offs

- [A Fedora workstation without the VM setup now fails `doctor`, so agents there stop before OpenSpec work] → This is the intended effect, stated as breaking in the proposal. The failing line names the fix, and the user can tell the agent to go on, as `AGENTS.md` already allows.
- [`ldd` runs the dynamic loader on the vendored binaries] → The binaries are the repository's own client tool bundles with recorded hashes, and `ldd` does not run their code on glibc. The check runs it only on files under `assets/tools/<arch>/`.
- [A host with fewer than 6 logical CPUs cannot give the VM 4 of them and keep 25% for itself] → Such a host does not meet the guide's minimum host size. The guide says so and points to a remote Ubuntu server as the way out. On macOS and WSL the same arithmetic applies to the CPU count.
- [A host with less than about 40 GB of memory cannot give the VM 32 GiB and keep enough for its desktop] → The guide's memory section already requires the host's own memory on top of the VM's limit; a smaller host uses a remote Ubuntu server.
- [Docker can be slow to answer under heavy test load] → The 10-second timeout makes a slow daemon fail the check instead of hanging it. Running `doctor` during `test-full` is unusual.
- [Codex runs `doctor` outside its sandbox] → The development environment check reads the same kind of state as the existing hook check, so the existing rule about running outside the sandbox covers it.
