## Context

See [proposal.md](proposal.md) for motivation and scope. This design is needed because moving retries can change failure and timeout behavior, and the existing workflow tests inspect the unexpanded YAML text.

Current evidence:

- The workflow has 1,491 lines. Its eight artifact-transfer chains each contain three attempts: candidate download in the browser, filesystem and upgrade jobs; candidate upload; application and verification digest upload; and their two manifest downloads (`.github/workflows/ci-release.yml:264`, `:616`, `:702`, `:779`, `:1058`, `:1104`, `:1228`, `:1277`).
- Candidate uploads allow five minutes per attempt; every other transfer allows three. Only the first two attempts use `continue-on-error`; later attempts depend on the previous attempt's `outcome` (`.github/workflows/ci-release.yml:616`, `:630`, `:644`).
- The Podman check and backend matrix repeat client-library installation, module download, `.env` creation, test PostgreSQL startup, readiness waiting and backup-directory creation (`.github/workflows/ci-release.yml:383`, `:514`).
- Smoke and image scanning already have a shared shell entry point (`.github/workflows/ci-release.yml:856`, `scripts/check_image.sh:1`).
- Workflow checks currently rely on raw job text and inline step names (`scripts/workflow_jobs.py:6`, `scripts/frontend_ci_test.py:18`, `scripts/native_release_test.py:52`). Some checks count the three inline artifact names (`scripts/native_release_test.py:104`).

The root `AGENTS.md` governs every affected path. No application module's `AGENTS.md` applies. The deployment and backend module boundaries in `docs/architecture.md` remain intact: preparation belongs to CI, not to an application package. Verification follows `docs/testing.md:130` and the existing native-CI guarantees at `docs/testing.md:180`.

## Goals / Non-Goals

**Goals:** Preserve the effective transfer steps, preparation commands and release graph while giving each shared block one definition inside the release workflow. Keep local checks able to detect an unsafe change inside that definition or at a call site.

**Non-Goals:** Build a generic workflow framework, a general YAML parser or an expression evaluator. Change the application architecture, add a dependency or establish hosted runtime success from local structural tests. The proposal lists the product and operational exclusions.

## Decisions

### 1. Keep artifact retries as workflow steps and reuse them with YAML anchors

Define three upload steps at their first real use and three download steps at their first real use. Anchor each complete step mapping and use a step alias at the other call sites. This preserves the existing execution boundary of each attempt. Do not introduce arbitrary top-level keys, YAML merge keys or nested step sequences.

Use job environment values for the transfer settings that differ between callers. Upload callers supply the artifact name, path, compression level and attempt timeout. Download callers supply a name or pattern, destination and merge setting. Keep the action SHA, retry conditions, overwrite policy, missing-file policy and retention policy in the canonical steps. Artifact inputs continue to refer to the current matrix architecture and, for verification digests, PostgreSQL version.

The upload timeout remains five minutes for the candidate archive and three for digest files. Download attempts remain three minutes. An expression used for a numeric timeout must convert the job's string value to a number. Candidate uploads retain compression level zero; digest uploads retain the upstream default. Both retain one-day retention, overwrite and missing-file failure. Candidate downloads use a name without merging; digest downloads use a pattern with merging.

Only attempts one and two may ignore failure. Attempt two runs only when attempt one fails; attempt three runs only when attempt two fails. The last failure remains fatal. Cancellation and an earlier unrelated failure must not start another transfer attempt. Repeated IDs are safe only across jobs: each job has exactly one upload or download chain.

Keep archive export, archive loading and the native-image check in their current jobs. Those commands are short, make the image boundary visible and need no additional action.

GitHub documents [workflow anchors and aliases](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#yaml-anchors-and-aliases). Its [composite metadata reference](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runs-for-composite-actions) documents `continue-on-error`, but does not document a timeout for an inner step. Workflow-level anchors avoid depending on that missing timeout facility.

**Alternatives:** A transfer composite would obscure or change the per-attempt timeout boundary. A reusable workflow would create new jobs and change runner and artifact boundaries. A custom JavaScript transfer action or shell wrapper would add another runtime or artifact client to maintain. Retaining all inline copies leaves the original maintenance risk. Anchoring only `with` mappings saves fewer lines and leaves the retry policy duplicated.

### 2. Reuse backend preparation with the same whole-step anchors

Anchor each of the six existing preparation step mappings at its first use in `e2e-verification-agent-podman`. Replace the matching steps in `test-backend` with aliases, preserving their positions. Both jobs need exactly the same commands and no parameters. The shared steps remain, in order:

1. Installing the current database-client system dependencies and creating the two ABI compatibility symlinks.
2. Downloading backend Go modules.
3. Copying `.env.example` to `.env`.
4. Starting `backend-db-test` through the existing compose command.
5. Waiting with the existing 60-second `pg_isready` check.
6. Creating the existing backup and temporary directories.

Preserve each step's name, commands, working directory and existing shell failure behavior. Keep all six visible as individual workflow steps. The callers retain checkout, mise tool selection, Go cache, backend image pre-pulls, Podman setup and cleanup, test execution, and diagnostics. Preparation does not stop the test database before its caller finishes.

**Alternatives:** A composite action would add a second reuse mechanism, metadata and separate action traversal in checks for six unparameterized steps whose only callers are in this workflow. A preparation shell script would collapse visible step boundaries. A reusable workflow would separate preparation from the runner that executes tests. Whole-step anchors already serve the transfer steps and need no additional files or reader support.

### 3. Make existing checks follow the shared definitions

Extend the workflow reader in `scripts/workflow_jobs.py` with narrowly scoped support for the complete-step anchors and aliases used here. Expand them before assertions inspect a job's steps; preserve access to the original job environment and matrix values. Normalize anchored step declarations into the existing step representation. Reject missing or duplicate anchors rather than silently returning incomplete steps. Leave full YAML syntax and expression validation to the pinned `actionlint` command. Do not grow a general YAML implementation or add a parser dependency.

Update affected checks to inspect both the shared transfer steps and each caller's environment settings. Replace raw occurrence counts with assertions about the three effective attempts and the matching artifact settings. Keep mutation checks that reject wrong architectures, missing suites, fatal failures marked as allowed, broken retry conditions, missing timeouts, incorrect action pins and disconnected publication gates. Restrict `continue-on-error` to the first two transfer attempts, including after alias expansion.

Checks of backend preparation must inspect the expanded steps in both jobs and verify the same six commands, their order and working directories. Exercise those shell commands in a temporary workspace using subprocess substitutes only for external programs such as `sudo`, `go` and `docker`. Verify successful preparation, a failed setup command stopping later commands, and failed readiness preventing subsequent preparation. Do not start or stop the shared host test database for these checks. Retain assertions that preparation failures are fatal and the steps keep their visible boundaries.

Audit existing consumers of `get_jobs`, local `get_job` readers and raw workflow splits before replacing inline steps. Update only the assertions affected by reuse; retain the existing tests of backend execution and release conditions.

**Alternatives:** Merely allowing aliases would stop the tests from checking the shared behavior. Keeping raw name counts would reject valid reuse. A generic YAML and expression engine would expand this refactor substantially. The bounded reader supports the repository's chosen representation, while `actionlint` remains the syntax authority.

### 4. Preserve behavior and keep validation local

Set `skip_specs: true`: the existing `ci-pipeline` specification already covers native execution, required suites, matching candidates and publication gating, and those requirements do not change. Add a short ownership paragraph to the CI section of `docs/testing.md`; no architecture or agent-instruction changes are needed.

Use workflow and script tests, workflow lint, Python quality checks and shell lint for implementation acceptance. Format affected Python files before the final checks. Do not run application suites merely because their CI preparation moved: no application source or scenario changes. Hosted CI is useful later evidence for the real runners, but is not an implementation task or a prerequisite requiring a push.

**Alternatives:** A new delta spec would prescribe implementation layout rather than change observable behavior. Requiring a full hosted run would make local completion depend on a push and infrastructure access. Claiming local checks prove native ARM or Podman execution would overstate their evidence.

## Risks / Trade-offs

- **Alias expansion hides an unsafe change from a raw-text test.** Inspect effective steps and caller settings; include mutations of both definitions and call sites, then run the full script suite.
- **Generic transfer settings accidentally mix candidate and digest behavior.** Pin the existing settings for all eight callers, both architectures and the verification-version matrix; retain the image check before every consuming suite.
- **An anchor is referenced before its definition or duplicates a step ID within a job.** Define anchors at the first existing call site and run workflow lint; reject unresolved aliases in the workflow reader.
- **A preparation alias changes ordering, shell failure handling or working directory.** Preserve the original step mappings and positions, execute the expanded commands against external-program substitutes, and verify that failures stop preparation.
- **Local checks do not execute GitHub's runner engine.** Report that limit explicitly. Retain the existing workflow triggers and manual verification-only behavior for subsequent hosted confirmation.

## Migration Plan

Land the workflow-reader checks first, convert candidate and digest transfers in separate slices, then reuse the preparation steps through anchors. Each slice must leave local checks passing. Keep all job identities and dependency edges unchanged throughout. This requires no application deployment, schema migration or compatibility shim. Reverting the refactor restores the inline definitions without changing artifact formats or consumers.
