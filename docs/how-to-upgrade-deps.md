# Upgrading dependencies

Databasus uses software maintained by other projects: libraries, development tools, container images and database clients. These are its **dependencies**. Updating them brings fixes and improvements, but can also change behavior or break a build.

A dependency update means choosing a suitable release, recording its exact version and checking that Databasus still works with it. Some versions appear in several places, such as local development settings, CI workflows and Dockerfiles. Those declarations need to agree.

Two kinds of files usually describe a package installation:

- A **manifest**, such as `package.json` or `go.mod`, declares the dependencies the project needs.
- A **lock file**, such as `pnpm-lock.yaml`, records the exact versions to install, including dependencies required by other packages. Those indirect dependencies are also called **transitive dependencies**.

Go uses `go.mod` and `go.sum` to select module versions and verify downloaded content.

**The person or agent preparing the update is responsible for choosing releases and checking compatibility.** Read the maintainers' release notes and security notices. Package managers check that the recorded dependencies and downloaded content agree. `mise run verify-deps` catches common mistakes in version declarations and committed client files. Neither check proves that every dependency is old enough or safe.

## Inventory and sources

Start with the files below. Check every area affected by the requested update, including any dependencies it adds. **Do not rely on automated checks to discover every dependency.** Review affected files outside this list too.

| Area | Files that record the dependencies | Official source |
| --- | --- | --- |
| Backend | `backend/go.mod`, `go.sum` | Go module and checksum services; official project releases |
| Go development tools | `tools/go/<tool>/go.mod`, `go.sum`; compiler in `mise.toml` | The tool's official repository and Go services |
| Frontend and API generator | `frontend/package.json`, `pnpm-workspace.yaml`, `pnpm-lock.yaml` | npm registry and official project releases |
| Website | `website/package.json`, `pnpm-workspace.yaml`, `pnpm-lock.yaml` | npm registry and official project releases |
| Development tools | `mise.toml`, `mise.lock`, referenced `.mise/locks/` sidecars | Vendor releases and the registry used by the mise backend |
| pnpm installation | `tools/pnpm/package.json`, `package-lock.json`; mise's `pnpm` declaration and backend alias | Official pnpm GitHub binaries for mise; the official pnpm npm package for containers |
| Python browser tests | `e2e/browser/pyproject.toml`, `uv.lock` | PyPI and official project releases |
| Browser environment | `frontend/browser/environment.json`, browser Dockerfile and Compose configuration | Node distributions and Microsoft Playwright images |
| CI installer and actions | `tools/dependencies/mise-bootstrap.json`, workflow action commit pins | Official project releases and repositories |
| Build/test images | Dockerfiles, Compose files, `backend/internal/lib/testimages/images.json` | The publisher's container registry |
| Database clients | `assets/tools/README.md`, files under `assets/tools/`, `tools/dependencies/clients.json` | PGDG, Oracle MySQL, MariaDB archives and MongoDB Database Tools |
| Verification extensions | Verification Dockerfile and its `vectorchord.sha256` | PGDG/vendor repositories and official extension releases |

This procedure covers dependencies from other projects. Treat images built from Databasus for testing separately, along with runtime images selected by operators and software required on the host.

**Keep supported database versions intact.** Some test fixtures use older releases to check compatibility with them. Keep each fixture on the release line it tests; do not upgrade it just to make version declarations easier to manage.

## Plan before editing

1. Inspect the working tree and preserve other contributors' changes.
2. Find the current versions and suitable releases. Read official release notes and security notices. Check required runtime versions, compatibility with other packages and every place that declares a related version.
3. **Agree the upgrade scope before editing.** Explain the selected versions, breaking changes and required checks. List major-version migrations left for later. Flag dependencies that have reached end of life (EOL), which means their maintainers no longer support them.
4. Update manifests with their package managers and commit the matching lock files. Update related runtime, package and image versions together. Keep the records for both supported architectures in sync.
5. Review installation scripts changed by the update. Check this procedure for obsolete commands or unnecessary work. Fix routine defects within the agreed scope; propose broader process changes separately.

## Check release dates

**Wait before selecting a new release.** The normal waiting periods are:

| Dependency | Waiting period after publication |
| --- | --- |
| Packages, development tools and GitHub Actions | Seven complete days |
| Container images, database clients and standalone verification extensions | Fourteen complete days |

Use the longer period when several dependencies must be updated together. Package managers' minimum-age settings help filter releases where supported.

**Check official publication dates yourself.** Age filters may not cover exact version selections, newly selected transitive dependencies or security updates. Review the lock-file changes for unexpected packages, publisher changes and installation scripts.

Report missing dates, uncertain sources and any exception to the waiting period. Explain the reason for an exception. Do not invent dates or use an image's build timestamp as the publication date of its digest. A digest identifies exact image content; the content may have been built before it was published.

Choose the newest patch release that has completed the waiting period within the agreed release line. A patch updates an existing line, such as a later `8.0.x` release. You make this choice; `mise run verify-deps` works offline and does not look up available releases.

| Case | Selection |
| --- | --- |
| Maintained client line; newest patch was published at least fourteen complete days ago | Select that patch. |
| EOL client line; final patch was published at least fourteen complete days ago | Select the final patch and report EOL status. |
| Newest client patch is too recent; the previous patch has completed the waiting period | Keep the previous patch. Report the deferred patch and the date its waiting period ends. |

If no suitable release has completed the waiting period, report that the update is deferred. If it must proceed sooner, explain why. Keep the agreed release line and documented architecture support intact.

## Install from lock files

**Install the recorded versions without changing them.** This is called a frozen or locked installation. It checks that the manifests and lock files agree instead of choosing new versions during installation.

Use the tool versions pinned in the repository. Tasks do not install missing tools automatically. Prepare them with the locked installation below.

Refresh a lock file only when the update changes its inputs. For mise, retain the records for both platforms and all referenced files under `.mise/locks/`. These extra lock files are called sidecars.

```sh
mise lock --platform linux-x64,linux-arm64
mise install --locked --include-task-tools
go -C backend mod tidy -diff
pnpm --dir frontend install --frozen-lockfile --ignore-scripts
pnpm --dir website install --frozen-lockfile --ignore-scripts
uv sync --locked --project e2e/browser
```

To refresh an unchanged mise tool's transitive dependencies, use `mise lock --bump <tool>`. Review the changes to its sidecar files.

Go preparation uses `go mod tidy -diff`. It downloads dependencies but fails if the committed `go.mod` or `go.sum` needs to change. Image compilation also uses `-mod=readonly`. **Do not run a `go mod download` command that changes those files before this check.**

Keep each package manager's own lock files. A separate list of resolved versions cannot replace them. Frozen installation and checksum verification check the dependency selection and downloaded content. The offline checker only checks that required lock files exist and that known declarations agree.

### Go tools and API generation

Go development tools use separate modules when their dependencies need isolation. The hidden `tools:go` task selects the pinned compiler and uses the modules without changing them. Run the installed wrapper from the project directory you want to analyze.

API generation uses the frontend's locked `openapi-typescript` package:

```sh
mise run generate-api
mise run frontend:lint
```

### Installation scripts and pnpm

**Review package installation scripts before allowing them to run.** Package managers may call these lifecycle scripts during installation. Do not disable verification checks just to make installation succeed.

Keep package-manager installation locked too. Do not reintroduce `pnpm dlx` commands that resolve packages independently, global npm installs or package-manager setup without a lock.

mise declares pnpm under the short name `pnpm` and maps it to `github:pnpm/pnpm` in `[tool_alias]`. The `pnpm` lockfile entry records that backend and the official native binaries for both Linux architectures, so `mise install --locked node pnpm` uses the same version locally and in CI. Container builds install the locked npm dependencies in `tools/pnpm` with lifecycle scripts disabled. They then run pnpm's reviewed installer explicitly to link the locked binary for that platform.

Dependency overrides and build-script permissions belong in each project's `pnpm-workspace.yaml`. Keep the frontend file among the Docker build inputs and the files used to calculate the browser-environment hash.

## Check clients, images and direct downloads

### Database clients

Read the [client bundle procedure](../.agents/skills/client-tool-bundles/SKILL.md) and [client inventory](../assets/tools/README.md) before refreshing a bundle. In `tools/dependencies/clients.json`, record:

- The exact vendor release and `build_family`.
- The source URL and archive SHA256 checksum.
- The hashes of committed executables and their private libraries.

The `build_family` identifies the vendor's distribution or binary compatibility variant. Examples are MongoDB's `ubuntu2204` and MySQL's `linux-glibc2.28`. It must match across related files and both architectures, even when the release version stays the same.

**Verify the download before extracting it.** Compare its checksum with the vendor's official checksum and check available vendor signatures with standard tools. A SHA256 checksum, also called a hash, identifies file content. Computing one locally does not prove who published that content.

After extraction, check the resulting files and executable permissions. Preserve existing requirements for files to match the vendor distribution, including the unmodified MongoDB tools. Local hashes detect changes to committed clients. They do not establish that the existing MySQL executables are byte-for-byte copies of their archive members.

Update related tools and both supported architectures together. MySQL 5.7 remains intentionally absent on arm64.

For MariaDB, update the exact package and source records before running `tools/refresh-mariadb-bundle.sh <bundle> <arch>`. The script uses Python, `ar` and the standard archive reader. It checks package and archive-member hashes before replacing files; it does not choose versions.

Run refreshed binaries in the runtime image. Follow the database engine's test requirements in the bundle procedure. **Never replace expected hashes automatically after a mismatch.**

### Images and verification extensions

Keep third-party image digests in their existing declarations or the test-image catalog. The verification Dockerfile pins a base image for each PostgreSQL major version. This makes `PG_MAJOR` select the same base locally and in CI.

Standalone VectorChord downloads check their recorded SHA256 before installation. Keep database clients committed to the repository; application image builds do not download them.

**A fixed base image does not guarantee an identical rebuilt image.** Extra packages installed through apt still come from distribution repositories and use their normal signature checks. This repository does not lock or retain all those packages and their dependencies, so their content can change on a later rebuild.

## Run the checks and report the result

```sh
mise run verify-deps
```

Run this from the root or a module directory with the normal Python runtime available. It checks:

- Exact tool versions, required lock files, matching backend aliases, and download URLs and checksums for both Linux architectures.
- Static single-line workflow `install_args` lists against mise declarations and native locks, including explicitly requested task-tool versions. Dynamic or multiline lists require extending the checker before use.
- Known GitHub Action and container image version declarations.
- Matching Go, Node, pnpm and Playwright versions across related declarations.
- Consistent client architectures and builds.
- Local file hashes and executable permissions, both in the working copy and in the existing Git index, which holds files staged for commit.

The command only reads known local files. It does not query registries, install dependencies or write to the checkout or Git index. Its source lists define what it checks. Review and extend those lists when adding a new kind of dependency.

CI runs the same command in the required shared Python-quality job. It runs before release version selection and publication of release and development images.

**A passing `verify-deps` check covers only those local checks.** Frozen installations, architecture tests, vulnerability scans and secret scans remain separate requirements. Choose functional checks using [docs/testing.md](testing.md#checking-work).

When reporting the update, include:

- Selected versions or digests and their official sources.
- Compatibility effects and related declarations updated together.
- The release-age review and any exceptions.
- Deferred updates and EOL release lines.
- The local verification result and functional checks actually run.

Put any incomplete check beside the claim it limits, so the reader can see what remains unverified.
