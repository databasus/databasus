## Context

See `proposal.md` for the problem and scope. The working tree already contains a partial implementation; the revised implementation must remove machinery that the bounded check no longer needs.

| Surface | Evidence in the working tree | Consequence |
| --- | --- | --- |
| Package graphs | `frontend/pnpm-lock.yaml:1`, `website/package-lock.json:1`, `backend/go.sum:1`, `e2e/browser/uv.lock:1` | Use the ecosystem's existing graph and integrity verification. |
| Frozen API generation | `frontend/package.json:14` and its lockfile | Retain the already completed generator installation change. |
| Dependency verifier | `scripts/dependencies/runner.py:300` stages tools and replays package graphs; `scripts/dependencies/python_markers.py:9` interprets markers | Delete this duplicate package-manager logic rather than simplifying its internal framework. |
| Publication history | `scripts/dependencies/baseline.py:18` requires an independently accepted release | Remove history and publication-service dependencies from routine verification. |
| Distribution packages | `Dockerfile:103` and `backend/cmd/verification-agent/images/postgres/Dockerfile:83` install from OS repositories | Report the reproducibility limit; do not implement retained package closures here. |
| Build and test images | `Dockerfile:2` already pins a base digest; `backend/internal/lib/testimages/image.go:29` exposes recorded test references | Retain useful immutable references and finish their known consumers without a registry-verification framework. |

The architecture in `docs/architecture.md:29` deliberately vendors database clients. Preserve that choice and the backend/frontend/website boundaries. Root `AGENTS.md` and each affected module's `AGENTS.md` govern implementation; Python quality and test selection remain owned by the existing repository tooling and `docs/testing.md`.

## Goals / Non-Goals

**Goals:** Make dependency upgrades reviewable, preserve native integrity checks, catch known pin/coordination mistakes locally and in CI, and remove code whose maintenance cost exceeds the agreed guarantees.

**Non-Goals:** Establish an adversarial proof of every dependency's publication history, certify the host toolchain, parse dependency resolution rules ourselves, or retain every OS package needed to reproduce an image.

## Decisions

### 1. Put selection and release-age judgment in the upgrade procedure

The maintainer or agent inventories affected surfaces, reads official release information, chooses suitable versions and explains compatibility and deferred updates before editing. The guide owns the seven-day default for packages/tools/actions and fourteen-day default for images/clients/standalone verification extensions. Related changes use the longer applicable waiting period. Ordinary Dependabot cooldowns and existing native age settings assist selection; retain the existing longer cooldown for unsolicited major proposals.

Exact-pin and security-update behavior can differ from native age filters. The updater reviews those cases and reports unavailable release dates or any departure from the normal waiting period with the reason. Missing publication metadata does not become a local-verifier failure. Do not invent dates or equate an image build timestamp with publication of its digest.

Reject a publication-proof service and trusted observation ledger: maintaining source adapters, historical baselines and bootstrap rules would dominate this task. Reject presenting native cooldowns as complete enforcement: they are an aid to a reviewed update, not the guarantee of the local command.

### 2. Let native tools own package graphs and integrity

Keep native manifests/locks authoritative. Normal development and CI installation use frozen/locked modes, existing Go checksum verification and reviewed lifecycle-script policy. The verifier neither installs packages nor interprets transitive resolution or Python environment markers. Native commands reject stale manifests and corrupted downloads in the installation jobs that already need those dependencies.

Retain `mise.lock` and its required sidecars for supported platforms, the locked pnpm bootstrap and the frozen API generator. Keep separate Go tool modules only where needed to obtain a native frozen graph that the existing tool backend cannot provide. Do not add another compiler migration or bootstrap layer merely to run verification. Existing version/hash pinning of the CI installer may remain; it does not need a custom acceptance stage.

Reject custom native graph replay and tool staging: normal locked installation already exercises those checks. Reject removing existing integrity checks or allowing unlocked installation to make the smaller verifier pass.

### 3. Keep the local check explicit, offline and small

Keep the public `mise run verify-deps` command. Implement it with a small entry point and only the focused helpers needed for known declarations and local checksums. Reuse existing repository checks where they already cover an invariant. A short explicit list of managed paths/reference locations defines its scope; the guide describes other surfaces the updater must review. Do not keep the generic ownership model or scan arbitrary source files to infer dependency declarations.

Check exact development-tool pins and required native-lock presence, full commit pins in known workflows, digest pins in known third-party image declarations, related Go/Node/pnpm/Playwright version declarations and recorded vendored executable/private-library hashes and executable permissions. Validate both supported architecture records and retain documented historical absences. The small client record contains source URL, release/build family, archive checksum and local file hashes; it contains no publication-evidence graph. Native locks retain their own version and integrity data.

Resolve the repository root consistently. Malformed known records, missing pins, local checksum mismatches and conflicting declarations produce actionable diagnostics and a nonzero exit. Read configuration and local files only; do not change the checkout/index, query registries, download archives or install tools. Report the actual checks performed and the boundary of a successful result. Package graph freshness/content remains a native-install check, not an offline claim.

Reject universal discovery and a plugin/provider framework because both would require maintaining a second dependency inventory engine. Reject a successful result described as complete dependency acceptance.

### 4. Authenticate downloads when updating or consuming them

Client refresh tooling verifies an expected archive digest before extraction. During refresh, use official checksum/signature information and standard vendor/package utilities where available, then record local file hashes for subsequent offline verification. Preserve existing source-distribution equality requirements, sibling build consistency, executable modes and both architectures. Do not silently regenerate expected hashes after a mismatch. A locally calculated hash alone does not authenticate a publisher.

Standalone build downloads, including verification extensions, verify an expected digest before installation. Native tools own package archive verification; no custom OpenPGP or Debian index parser is needed. Database clients remain committed and are not fetched during application image builds.

Reject downloading and re-extracting every client archive during each verification run: authenticity is established at refresh and local hashes detect later drift. Reject the checksum-printing-only refresh path, which does not verify downloaded content.

### 5. Pin image inputs without promising a complete OS-package lock

Use committed digest references for repository-selected third-party build/test images and verification bases. Local builds and CI consume the same references instead of resolving a current tag during publication. Keep first-party candidate images, operator-selected runtime references and intentionally historical compatibility fixtures distinct; do not upgrade their release lines just to simplify checks.

Additional packages installed through apt remain under normal distribution signature verification. Record this reproducibility limit in the guide. Do not add retained repositories, package closures or per-package publication evidence. Pin and checksum standalone extensions separately because their bytes are directly selected by this repository.

Reject claiming that a base-image digest freezes packages installed later. Reject implementing a package-retention service within this upgrade-process change.

### 6. Gate publication with the bounded check and existing native checks

Run the same offline command locally and in CI for the candidate commit. Require success before release version selection, release publication and both development-image publication routes. Failed, skipped, missing or cancelled required results do not permit publication. Preserve normal frozen installation, architecture test matrices, vulnerability scans and secret scans; a local pin check does not replace any of them.

Use normal repository-pinned CI tools. Remove the requirement to authenticate candidate toolchains before executing the verifier, the trusted-baseline environment variables and special host-Python staging. A focused CI job or an existing required validation job is enough; choose the simpler wiring while proving every publication route depends on it.

Reject a release-evidence network gate, which adds external availability as a prerequisite to ordinary releases. Reject weakening existing test/security gates when adding the small check.

## Risks / Trade-offs

- A maintainer can select a too-fresh exact pin or miss a new dependency surface. The documented review and native selection aids reduce that risk; the command does not prove universal age compliance or discovery.
- Native tools and registries still participate in normal installation. Their integrity checks remain enabled, and installation failures fail their jobs.
- OS-package repositories can change after a base image is pinned. Image-level reproducibility remains limited until a separately scoped package-locking effort.
- Source authenticity depends on careful refresh work. Preserve vendor source information, check expected bytes at download time and use available standard signature verification.
- The partial implementation has useful locks and consumers mixed with discarded machinery. Remove only this change's obsolete code/data/tests, preserve other contributors' changes and run the affected checks after removing references.

## Migration Plan

1. Remove obsolete verifier mechanisms and their exclusive fixtures/data/wiring. Preserve useful pins and native locks, and keep any still-needed small checksum helper with its consumer.
2. Finish native locked installation and the bounded offline checks; verify known pin, coordination, local hash and executable-mode failures.
3. Finish immutable image consumers and expected-digest download checks without introducing OS-package retention.
4. Wire the local check into publication prerequisites, preserving existing native and security checks.
5. Rewrite the upgrade guide and glossary, correct command/document references and public security claims, and verify the complete bounded baseline before marking tasks done.

No application-data migration is required. A rollback restores related manifests, locks and build consumers together. It must not disable existing security checks.
