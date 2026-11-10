# Upgrading dependencies

The person or agent preparing an update owns release selection, upstream review and compatibility judgment. Native package managers verify locked package graphs and downloaded package content. `mise run verify-deps` catches common mistakes in known pins, related versions and committed client files; it does not prove that every dependency is old enough or safe.

## Inventory and sources

Review every surface the requested update affects, including new dependencies introduced by the change. The table is the starting point, not a promise that a scanner will discover every external input.

| Surface | Authoritative inputs | Official source |
| --- | --- | --- |
| Backend | `backend/go.mod`, `go.sum` | Go module/checksum services and upstream releases |
| Go development tools | `tools/go/<tool>/go.mod`, `go.sum`; compiler in `mise.toml` | The tool's upstream repository and Go services |
| Frontend and API generator | `frontend/package.json`, `pnpm-workspace.yaml`, `pnpm-lock.yaml` | npm registry and upstream releases |
| Website | `website/package.json`, `package-lock.json` | npm registry and upstream releases; keep npm |
| Development tools | `mise.toml`, `mise.lock`, referenced `.mise/locks/` sidecars | Vendor releases and the backend's native registry |
| pnpm bootstrap | `tools/pnpm/package.json`, `package-lock.json`; mise's `github:pnpm/pnpm` declaration | Official pnpm GitHub binaries for mise; the official pnpm npm package for containers |
| Python browser suite | `e2e/browser/pyproject.toml`, `uv.lock` | PyPI and upstream releases |
| Browser environment | `frontend/browser/environment.json`, browser Dockerfile and Compose configuration | Node distributions and Microsoft Playwright images |
| CI installer and actions | `tools/dependencies/mise-bootstrap.json`, workflow action commit pins | Upstream releases and repositories |
| Build/test images | Dockerfiles, Compose files, `backend/internal/lib/testimages/images.json` | The publisher's container registry |
| Database clients | `assets/tools/README.md`, files under `assets/tools/`, `tools/dependencies/clients.json` | PGDG, Oracle MySQL, MariaDB archives and MongoDB Database Tools |
| Verification extensions | Verification Dockerfile and its `vectorchord.sha256` | PGDG/vendor repositories and official extension releases |

Distinguish third-party inputs from first-party candidate images, operator-selected runtime references and host prerequisites. Historical compatibility fixtures retain the release line they test. Do not change supported database versions or upgrade a historical fixture merely to simplify pinning.

## Plan before editing

1. Inspect the working tree and preserve other contributors' changes.
2. Identify current versions, suitable candidates, official release notes, security advisories, peer/runtime constraints and related declarations.
3. Explain selected versions, compatibility breaks, required checks, deferred major migrations and EOL dependencies. Agree the upgrade scope before implementing it.
4. Update manifests with native tools and commit their corresponding locks. Update related runtime/package/image pins and both supported architecture records together.
5. Review changed installation scripts and this procedure for obsolete commands or unnecessary work. Repair routine defects within the agreed upgrade; propose broader process changes separately.

## Review release age

The normal waiting period is seven complete days for packages, development tools and actions, and fourteen days for container images, database clients and standalone verification extensions. Use the longer period for coordinated updates. Ordinary Dependabot updates use the configured cooldowns; unsolicited major proposals retain the longer thirty-day delay. Existing native minimum-age settings assist selection where supported.

Review official publication information for selected releases. Exact pins, newly selected transitives and security updates can fall outside native filters. Review the lock diff, especially unexpected new packages, publisher changes and installation scripts. Report missing dates, uncertain source information or a departure from the normal waiting period with its reason. Do not invent dates, treat an image build timestamp as its digest's publication date or present an ordinary cooldown as a guarantee about security updates.

Choose the newest eligible patch in the agreed release line. These examples describe updater decisions; the offline command does not query a release catalog:

| Case | Selection |
| --- | --- |
| Maintained client line, newest patch has completed fourteen days | Select that patch. |
| EOL client line, final patch has completed fourteen days | Select the final patch and report EOL status. |
| Newest client patch is too fresh, predecessor is eligible | Keep the predecessor and report the deferred patch and eligibility date. |

If no suitable release has completed the period, report the deferral or explain why the update needs to proceed sooner. Keep the designated release line and documented architecture support intact.

## Use native frozen installation

Use the pinned tools. Tasks do not automatically install missing tools; prepare them through the explicit locked installation below. Refresh a native lock only for inputs the update changes, retaining both platform records and all referenced sidecars.

```sh
mise lock --platform linux-x64,linux-arm64
mise install --locked --include-task-tools
go -C backend mod tidy -diff
pnpm --dir frontend install --frozen-lockfile --ignore-scripts
npm --prefix website ci --ignore-scripts --no-audit --no-fund
uv sync --locked --project e2e/browser
```

For an unchanged mise tool whose transitive graph needs refreshing, target it with `mise lock --bump <tool>`. Review its resulting sidecar diff. Go preparation uses `go mod tidy -diff` to download dependencies while rejecting any required change to committed `go.mod` or `go.sum`; image compilation also uses `-mod=readonly`. Do not run a mutating `go mod download` before that validation. Do not replace native locks with another resolved-version inventory. Frozen installation and native checksum verification remain the authority for package graph consistency and package content; the offline checker only verifies required lock presence and known declarations.

Go development tools use separate native modules where their dependency graphs require isolation. The hidden `tools:go` task selects their pinned compiler and readonly modules. Run the installed wrapper from the project directory being analyzed. API generation uses the frontend's locked `openapi-typescript` package:

```sh
mise run generate-api
mise run frontend:lint
```

Do not restore independently resolved `pnpm dlx`, global npm installs or an unlocked package-manager bootstrap. Review any dependency lifecycle scripts before allowing them; ordinary checks must not turn verification off to make installation succeed.

mise installs pnpm's official native binary. Container builds install the frozen `tools/pnpm` npm graph with lifecycle scripts disabled, then explicitly run pnpm's reviewed installer to link its locked platform binary. Frontend overrides belong in `pnpm-workspace.yaml`; keep it in the Docker build inputs and browser-environment hash.

## Check clients, images and direct downloads

Read the [client bundle procedure](../.agents/skills/client-tool-bundles/SKILL.md) and [client inventory](../assets/tools/README.md) before a refresh. Record the exact vendor release, `build_family`, source URL, archive SHA256 and committed executable/private-library hashes in `tools/dependencies/clients.json`. The build family identifies the vendor distribution or ABI variant, such as MongoDB's `ubuntu2204` or MySQL's `linux-glibc2.28`, and must match across sibling files and architectures even when the release version is unchanged. Check official checksum information and available vendor signatures with standard tools. A locally computed hash identifies bytes; it does not authenticate their publisher.

Verify expected archive bytes before extraction, then verify the resulting files and executable permissions. Preserve existing source-distribution equality guarantees, including the unmodified MongoDB tools. Local file hashes detect drift in committed clients; they do not assert that the existing MySQL executables are byte-identical to their archive members. Update sibling tools and both supported architectures together. MySQL 5.7 remains intentionally absent on arm64.

For MariaDB, update its exact package/source records before running `tools/refresh-mariadb-bundle.sh <bundle> <arch>`. The script uses Python, `ar` and the standard archive reader, verifies package and member hashes before replacing files, and does not select versions. Run refreshed binaries in the runtime image and follow the engine test requirements in the bundle procedure. Never replace expected hashes automatically after a mismatch.

Keep third-party image digests in their existing declarations or the test-image catalog. Verification-image bases are pinned per PostgreSQL major directly in their Dockerfile, so `PG_MAJOR` selects the same base locally and in CI. Standalone VectorChord downloads check their recorded SHA256 before installation. Keep database clients committed; application image builds do not download them.

A base digest fixes the base image's content. Additional packages installed through apt still come from distribution repositories and use their normal signature verification; their complete dependency closure is not locked or retained here. Consequently this procedure does not promise identical image bytes on every later rebuild.

## Run the checks and report the result

```sh
mise run verify-deps
```

The command works from the root or a module directory with the normal Python runtime available. It checks exact tool pins and required native-lock files, known action/image pins, coordinated Go/Node/pnpm/Playwright versions, recorded client architecture/build consistency, local file hashes and executable permissions in the working copy and existing Git index. It reads known local files and never queries registries, installs dependencies or writes the checkout/index. Its source lists define its scope; review and extend them when introducing new dependency surfaces.

CI runs the same command in the required shared Python-quality job before release version selection and release/dev-image publication. Frozen installations, architecture tests, vulnerability scans and secret scans remain required separately. A green offline result establishes only the configured local checks.

Select functional verification through [docs/testing.md](testing.md#checking-work). Report selected identities and official sources, compatibility effects, coordinated changes, release-age review and exceptions, deferred updates and EOL lines, the local verification result and functional checks actually run. Identify incomplete checks beside the claims they limit.
