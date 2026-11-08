## Why

The successful manual CI and Release run on `develop` took about 30 minutes. On ARM, the remaining backend group took 19.5 minutes, browser UI took 12.6 minutes, and browser lifecycle checks took 15.9 minutes after the candidate image became available. Splitting these checks across runners will shorten the longest jobs while retaining the same required coverage.

## What Changes

- Run backend `other` tests in three duration-balanced shards on each architecture. Assign individual top-level tests, including tests within slow packages, rather than distributing whole packages.
- Give the root storage-provider test package its own backend matrix leg per architecture, retaining its isolated execution and excluding it from all three `other` shards.
- Run browser UI in three shards per architecture through the existing local command, preserving two workers, zero retries, exact screenshot comparisons, the pinned visual environment and separate failure artifacts.
- Keep the real-application browser scenarios in one job per architecture. Move lifecycle checks into three independent jobs per architecture: failures, interruption, and isolation.
- Keep the complete local lifecycle command as the default and add explicit group selection for CI and focused local verification.
- Extend automated coverage and release-gate checks so missing, duplicated or unsuccessful required selections cannot make a commit eligible for release.
- Document the commands, group ownership and measured baseline in the testing documentation.

### Out of Scope

- Application behavior, HTTP contracts, database schemas and interface copy.
- Removing scenarios, weakening assertions, adding retries or increasing per-runner test concurrency.
- Changing the existing physical, physical API, logical or agent end-to-end group assignments.
- Splitting frontend unit tests, changing runner profiles, increasing hosting quotas or redesigning image builds and transfers.
- Making a fixed wall-clock improvement a correctness requirement; queue time and repeated setup can change the result.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: require three remaining-backend shards, a separate isolated provider check, three browser UI shards and independently gated browser lifecycle groups on both architectures. Preserve complete coverage and fail-closed publication.

The application's product specs and the browser suite's behavioral guarantees remain unchanged. The existing CI requirements explicitly prescribe whole groups and combined browser jobs, so this change needs a CI delta spec rather than `skip_specs`.

## Impact

- CI workflow matrices, their existing script tests, backend selection tasks and duration weights.
- Frontend browser tooling and its tests; browser lifecycle selection and its infrastructure tests.
- `docs/testing.md` and the `ci-pipeline` delta spec. No new external dependency is expected.
- The work answers to root `AGENTS.md`, `frontend/AGENTS.md` for frontend tooling and `backend/AGENTS.md` for backend files. The existing module and end-to-end boundaries remain intact.
- **BREAKING (CI check identities only):** CI check names and backend matrix group names change; their old whole-group entries are replaced, not retained as duplicate checks. Repository-hosted branch rules that require an old check name must be assessed before adopting the new workflow. Existing unsharded local commands retain their complete-suite behavior.
