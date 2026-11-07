## Why

The release workflow repeats artifact transfers and backend test preparation. A change to retries, artifact settings or database setup can leave equivalent jobs behaving differently. Centralizing those blocks reduces that maintenance risk without changing CI coverage or release behavior.

## What Changes

- Reuse complete artifact upload and download steps through YAML anchors and aliases within `.github/workflows/ci-release.yml`. Preserve three attempts, each attempt's timeout, failure propagation and all artifact settings.
- Reuse the six identical backend preparation steps through whole-step YAML anchors, defined in the Podman check and referenced by the backend matrix. Preserve their commands, ordering, failure handling and visible step boundaries.
- Adapt the repository's workflow checks to inspect reused transfer and preparation steps, retaining assertions for native architectures, required checks and publication gates.
- Document the ownership of these shared blocks in the CI section of `docs/testing.md`.

### Out of Scope

- Shard weights, runner sizing, test concurrency, timeouts and CI performance tuning.
- A new smoke action or changes to `scripts/check_image.sh`, Trivy installation or scan policy.
- Changes to job names, matrices, dependencies, workflow triggers, permissions or publication conditions.
- Refactoring other workflows, including `deploy-dev-image.yml`, or changing application code and tests under `backend/` or `frontend/`.
- Pushing a branch, starting hosted CI or publishing artifacts as part of implementation verification.
- A promised line-count reduction. Shared definitions count toward the maintenance cost too.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is an internal CI refactor that preserves the existing `ci-pipeline` requirements. The change metadata sets `skip_specs: true`; no delta spec is needed.

## Impact

- Implementation: `.github/workflows/ci-release.yml`, affected workflow checks in `scripts/`, and `docs/testing.md`.
- The work follows the root `AGENTS.md`. No module-specific `AGENTS.md` applies to these paths; application module boundaries in `docs/architecture.md` remain unchanged.
- External actions keep their existing full commit SHA pins. No new package or tool dependency is planned.
- No HTTP, database, configuration or caller compatibility change is introduced.
