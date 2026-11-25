# Proposal

## Why

Development now happens in an Ubuntu VM that a developer sets up by hand from `docs/how-to-setup-vm.md`. A missed step stays silent until a test fails minutes into a run. A missing kernel module makes the CloudNativePG suite time out, and a missing system library makes a program of a client tool bundle fail to start. `mise run doctor` already runs before every OpenSpec change, but it checks only the tools of the review hooks, so a broken environment passes it.

## What Changes

- `mise run doctor` also runs a development environment check and prints one line per prerequisite. Each failed line has a hint that names the section of the VM setup guide that fixes it.
- Any missing prerequisite makes the command exit with a non-zero code:
  - the kernel applies iptables rules to bridged traffic, through the `br_netfilter` module;
  - the inotify limits are at least the values the CloudNativePG suite needs;
  - Docker answers, runs rootful and has the Compose plugin;
  - every program of the client tool bundles for the machine's architecture finds all its shared libraries;
  - every tool pinned in `mise.toml` is installed;
  - the root filesystem is ext4;
  - the machine has at least 4 logical CPUs and 32 GiB of memory;
  - the system is Ubuntu;
  - the commit hook is installed.
- **BREAKING** for agents: the OpenSpec preflight in `AGENTS.md` stops on a non-zero `doctor` exit. A machine that misses a prerequisite, such as a host without the VM setup, a btrfs root, a guest with fewer than 4 logical CPUs or less than 32 GiB of memory, a system other than Ubuntu or a clone without the commit hook, can no longer start OpenSpec work until it is fixed. A host that cannot give the VM that much while keeping its own share is no longer supported for development; a remote Ubuntu server replaces it. This is the intended behavior. Nothing is kept for the old, narrower check.
- The `doctor` purpose in the command table of `AGENTS.md`, the preflight text, the glossary and `docs/how-to-setup-vm.md` describe the new scope.

## Out of scope

- Fixing the environment. The check reports and never changes system settings, installs packages or loads modules.
- Checks for Lima, WSL or Incus on the host. The check runs inside the guest and looks only at the guest.
- Docker's default address pools, which Docker does not report, and the machine's network routes.
- The Podman and filesystem test suites, which need separate host setup.
- Checking authentication of the review client, which the existing requirement already excludes.

## Capabilities

### New Capabilities

- `development-environment-check`: which prerequisites of the development environment the development environment check verifies, how it reports them, and which of them fail the check.

### Modified Capabilities

- `review-hook-launch`: `mise run doctor` runs the development environment check after the hook tools, prints its report and fails when a prerequisite is missing. The launcher's doctor action keeps covering the hook tools only.

## Impact

- `mise.toml` (`doctor` task), `scripts/doctor.sh`, `scripts/development_environment_check.py` and their tests under `scripts/`. The libedit shim in `scripts/fedora.sh` is reused unchanged.
- `AGENTS.md`: the `doctor` row of the command table and "Environment check before a change".
- `docs/glossary.md`: the hook tool check, the development environment check, and the prerequisite.
- `docs/how-to-setup-vm.md`: the CPU, Memory and Disk requirements state 4 logical CPUs, 32 GiB and ext4 as hard minimums and a minimum host of 6 logical CPUs; `mise run doctor` opens the check section.
- Module rules: the root `AGENTS.md` and the Python rules it holds; the change touches no backend, frontend or website code.
