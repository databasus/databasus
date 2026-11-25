# Proposal

## Why

Development moved from a Fedora host into an Ubuntu VM, and `mise run doctor` now fails on any system other than Ubuntu. The bundled database clients need `libedit.so.2`. Fedora ships that library only as `libedit.so.0`, so `mise run run`, `mise run test`, both agent end-to-end tasks and `mise run doctor` created an alias for it before they started. Ubuntu provides `libedit.so.2` under its own name, and the VM setup guide installs it. The alias step therefore does nothing on any supported machine. It is still sourced by five scripts, stubbed in four test files, repeated in four command blocks of `docs/testing.md` and written into two specs as required behavior.

## What Changes

- **BREAKING** for a developer who still runs the tooling directly on Fedora: the run and test commands no longer create the `libedit.so.2` alias. `scripts/fedora.sh` and its test `scripts/fedora_test.py` are deleted. No replacement or fallback is kept. `mise run doctor` already rejects Fedora.
- `mise run run`, `mise run test`, `mise run agent:e2e`, `mise run agent:e2e-podman` and `mise run doctor` stop sourcing the alias step. `mise run doctor` stops creating its temporary alias directory.
- The tests that stub the alias step or check the alias itself drop those stubs and those cases.
- The client tool bundles check of `mise run doctor` looks up libraries exactly as the system provides them. It names a missing `libedit.so.2` like any other missing library.
- `docs/testing.md` loses the alias step from its manual commands and the paragraph explaining it. `assets/tools/README.md` loses the sentence saying that the run and test commands work around the Fedora library name.
- The specs describe the Ubuntu development VM instead of a Fedora host.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: `mise run test` no longer assumes a Fedora host or makes the bundled clients startable through a library alias. The scenarios for `mise run run` and for pinned tools name the Ubuntu development VM. The capability's Purpose line names the VM as well.
- `development-environment-check`: the client tool bundles check no longer accounts for a library compatibility step. Its requirement is replaced by one without the two scenarios about that step.

## Impact

- Scripts: `scripts/dev.sh`, `scripts/test.sh`, `scripts/doctor.sh`, `mise-tasks/agent/e2e`, `mise-tasks/agent/e2e-podman`; `scripts/fedora.sh` and `scripts/fedora_test.py` are deleted.
- Tests: `scripts/doctor_test.py`, `scripts/test_sh_test.py`, `scripts/test_group_test.py`, `.agent-hooks/review_launch_test.py`.
- Docs: `docs/testing.md`, `assets/tools/README.md`.
- CI is unaffected. Its jobs install `libedit2` with `apt-get` and do not source the alias step.
- The change answers to the root `AGENTS.md` only. None of the touched folders has its own `AGENTS.md`.

## Out of scope

- The Fedora host section of `docs/how-to-setup-vm.md`. There Fedora is the computer that runs the Ubuntu VM, which stays supported.
- Podman support. The verification agent supports Podman as a product feature, and its CI job and `mise run agent:e2e-podman` stay. That task only loses the alias step.
- The libraries listed in `assets/tools/README.md` and the CI `apt-get` package lists.
- Archived OpenSpec changes that mention Fedora.
