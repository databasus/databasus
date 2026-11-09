## Why

Dependency updates span application packages, development tools, workflow actions, container images and vendored database clients. Maintainers need one practical upgrade procedure and a few reliable checks for common mistakes, without maintaining a second dependency-management system.

## What Changes

- Add `docs/how-to-upgrade-deps.md` with the dependency inventory, official sources, release selection, coordinated pins, functional checks and final report format for people and agents.
- Keep release-age review with the person or agent preparing the update. Use seven days for packages, tools and actions, and fourteen days for images, database clients and standalone verification extensions. Reuse native age filters and ordinary-update cooldowns where supported; do not claim universal enforcement for exact pins, transitive packages or security updates.
- Use native lockfiles, frozen installations and package-manager integrity checks for application and tool dependency graphs. Keep API generation inside the frontend's frozen graph.
- Keep `mise run verify-deps` as a small, offline, read-only check of explicitly listed pins, related version declarations and recorded local artifact hashes. It does not discover arbitrary dependency inputs, query publication services or install dependencies.
- Pin repository-controlled third-party Actions to full commits and container inputs to digests, including verification-image bases and known test images. Keep exact tool versions and their supported native locks for both architectures.
- Verify downloaded client archives and standalone binaries against expected checksums before extraction or execution during refresh/build. Record source URLs and local vendored-file hashes, and review available vendor signatures using standard tools.
- Delete the unfinished universal verifier, publication-evidence history, isolated tool staging, native graph replay, custom lock/marker interpretation and their dedicated fixtures. Remove data and wiring used only by those mechanisms while retaining useful locks, pins and checksum checks.
- Require the small local check before release and development-image publication, alongside existing frozen installation, test and security checks.
- **BREAKING (build configuration):** remove unlocked API-generator/package-manager installation and build-time selection of moving third-party image tags. Full retention and locking of additional OS-package dependency closures is outside this change; builds that use distribution repositories remain dependent on those repositories.

## Capabilities

### New Capabilities

- `dependency-maintenance`: A maintainer-led upgrade procedure with native dependency verification and a bounded local check for known pins and artifacts.

### Modified Capabilities

- `ci-pipeline`: Require the bounded dependency check before publication while preserving native installation and security gates.
- `client-tool-bundles`: Select the newest eligible patch during a client refresh, with a documented fourteen-day waiting period.
- `local-development-workflow`: Expose `verify-deps` as the ninth everyday command.

## Impact

Affected areas are dependency/build configuration, native lockfiles, small scripts and their tests, container references, client refresh tooling, CI and maintenance documentation. The application API, stored data and supported database versions do not change. Root `AGENTS.md` and the module rules in `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md` apply wherever their files are touched. Instruction changes stay limited to accurate command and document references.

### Out of scope

- A resolver, updater, agent-launching skill, automatic commits, scheduled maintenance or publication.
- A machine proof of publication age for every exact artifact or transitive package, trusted historical baselines, first-observation records or a custom bootstrap trust chain.
- Universal repository dependency discovery, a second interpretation of native dependency graphs, or isolated reinstallation of every graph by the verifier.
- Retained OS-package repositories, signed-index inventories or complete apt dependency locks; these need a separate reproducible-build proposal.
- Bulk version refreshes and the major migrations in `upgrade-major-dependencies`.
- Claims that a passing local check proves upstream safety, runtime compatibility or bit-for-bit reproducible images.
