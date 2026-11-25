## Context

See proposal.md for scope. `docs/how-to-upgrade-deps.md` owns version selection, waiting periods, locked installation and reporting. This change follows it and adds no new process.

The PostgreSQL bundles are recorded at 14.23 to 18.4 (`tools/dependencies/clients.json:254`, `:384`). The verification agent already restores into 14.25 to 18.7 (`backend/internal/agent/container/postgres_versions.json`), so only the clients lag behind. MySQL bundles are at 8.4.11, 9.7.2 and 26.7.0 (`clients.json:474`, `:496`, `:518`), MongoDB tools at 100.19.1 (`clients.json:552`). MongoDB 4.2 is the minimum supported server (`backend/internal/engines/mongodb/version.go:62`, `:78`), and the backend uses mongo-driver v2.8.2 (`backend/go.mod`). The MySQL fixture's wait callback takes a string port (`backend/internal/lib/testcontainers/mysql.go:76`), which testcontainers 0.43 changes. The mise tool pins sit at `mise.toml:150` to `:175`; pnpm repeats in `frontend/browser/environment.json:9`, both `package.json` files and `tools/pnpm/package.json`.

The work is constrained by the module rules: `backend/AGENTS.md` for Go code and tests, `frontend/AGENTS.md` for frontend tooling and formatting, `website/AGENTS.md` for the website. It moves no logic between modules, so `docs/architecture.md` stays as it is.

## Goals / Non-Goals

**Goals:** Every dependency the proposal names reaches its newest eligible release within its current line. The bundled PostgreSQL clients no longer write `pg_attver`. The existing tests pass on the new versions.

**Non-Goals:** Changing a supported server range. Migrating a major line. Shortening the waiting period of any release other than the fixed PostgreSQL minors.

## Decisions

### 1. Take the fixed PostgreSQL clients now, under the critical-fix exception

The fixed minors were published on 2026-11-12, and the fourteen-day period ends on 2026-11-27. The guide allows a shorter wait for a release that fixes a defect breaking users' verification (`docs/how-to-upgrade-deps.md:72`). Issue #921 is that defect. The exception covers only the five PGDG client packages. The commit names the releases, the publication date and the three shortened days. The first commit used the same exception for the verification images.

Waiting until 2026-11-27 was rejected. It would keep three more days of dumps that older servers reject. Taking the clients in a separate later commit was rejected too. It would split one dependency pass in two and gain nothing, because the exception applies either way.

For PostgreSQL 18 the change takes package `18.7-1.pgdg12+2`. The later rebuild `+5` is still inside its waiting period and changes nothing for the client.

### 2. Keep mongo-driver v2.8.2 and the MongoDB 4.2 support

CVE-2026-107324 in `bson.Raw.Validate` is fixed only in v2.10.0. Driver 2.9 and newer drop MongoDB 4.2. Backend code does not call `bson.Raw.Validate`, and govulncheck reports no reachable vulnerability in the module. The driver stays on v2.8.2, and the report names the open advisory.

Upgrading to v2.10.0 was rejected, because it removes a supported server version as a side effect of maintenance. Keeping both drivers was rejected, because two drivers double the MongoDB integration for one advisory. Dropping MongoDB 4.2 is a product decision for a separate change.

### 3. Take minor releases with behavior changes and adapt to them

gin 1.12 changes `ClientIP` for several `X-Forwarded-For` headers and how empty slices bind. kin-openapi 0.153 rejects an empty query value instead of substituting the default, and changes the default `explode`. testcontainers 0.43 changes the port type of the `wait.ForSQL` callback. vacuum 0.41 may add OpenAPI lint findings. steiger 0.9 resolves its configuration from the target path. Each change is adapted where a test, the linter or the compiler shows it. A behavior change visible to API clients is fixed in code so the API keeps its contract; a test is never loosened.

Holding these at their current minors was rejected. Several carry fixes, and a later pass would have to absorb the same changes. React 19.3 is the exception: its rendering change affects every screen, and the test suite cannot show a visual regression.

### 4. Defer releases that are too fresh instead of taking older substitutes

Node 24.22.3, Playwright 1.65.1 and the other releases in the proposal's out-of-scope list reach eligibility between 2026-11-25 and 2026-12-09. They stay out of this change, and the report lists their dates. Node stays on 24.21.0 instead of moving to 24.22.2, because 24.22.3 is the security release and an interim step would repeat the update of every Node pin in a few days.

### 5. Update each package manager's records with its own tool

Go modules change through `go get` and `go mod tidy`. Frontend and website packages change through `pnpm`, Python packages through `uv lock`, mise tools through `mise lock --platform linux-x64,linux-arm64`. Client bundles follow `.agents/skills/client-tool-bundles/SKILL.md`: verify the vendor archive before extraction, record archive and binary hashes for both architectures and run `--version` in the runtime image. Hand-editing a lock file was rejected, because a package manager then cannot vouch for it.

## Risks / Trade-offs

- [A behavior change in gin or kin-openapi alters what an API client sees] → API tests in `mise run test` cover request validation and client addresses; a failing test is fixed in code.
- [The new PostgreSQL client changes dump output in another way] → the full PostgreSQL logical and physical backup matrices run on the refreshed clients.
- [mongo-driver keeps a known advisory] → the call is unreachable from backend code, and the report names it for the decision on MongoDB 4.2.
- [A tool upgrade reformats or relints many files] → formatting changes stay in this commit and are reviewed as part of it; new lint findings are fixed, not suppressed.

## Migration Plan

Contributors run `mise install --locked --include-task-tools` after pulling. Released images carry the new clients. Rolling back means reverting the commit.

## Selection record

Checked on 2026-11-24. Packages, tools and actions needed a publication date on or before 2026-11-16; database clients and images on or before 2026-11-09, except the PostgreSQL exception below.

**Database clients**, from the vendors' official sources:

| Bundle | Version | Source and check | Published |
| --- | --- | --- | --- |
| PostgreSQL 14 to 18 | 14.25-1.pgdg12+1, 15.20-1.pgdg12+1, 16.16-1.pgdg12+1, 17.12-1.pgdg12+1, 18.7-1.pgdg12+2 | apt.postgresql.org `bookworm-pgdg`; `InRelease` signed by the PGDG key `ACCC4CF8`, package hashes from the signed index | Release announced 2026-11-12 |
| MySQL 8.4, 9.7, 26 | 8.4.13, 9.7.4, 26.10.0 `linux-glibc2.28` | cdn.mysql.com; `.asc` signatures by MySQL Release Engineering | 2026-10-20 |
| MongoDB Database Tools | 100.20.1 `ubuntu2204` | fastdl.mongodb.org; SHA256 from the vendor's `release.json` | 2026-11-02 |

The PostgreSQL clients take the critical-fix exception of `docs/how-to-upgrade-deps.md`: the fourteen-day period counted from the 2026-11-12 announcement ends on 2026-11-27, so it is shortened by three days, because the old clients write `pg_attver` (#921). The pool files carry earlier build dates; the guide counts the publication, not the build. MongoDB tools 100.20 drop servers older than 4.2, so the supported range is unchanged. MySQL 26.10.0 ships OpenSSL 3.5.9 in `lib/private`. Every refreshed binary printed its version in the runtime image on x64. The arm64 binaries were checked statically (ELF machine, version string, glibc and library needs) because the host cannot run arm64 binaries.

**Images.** The PostgreSQL fixtures in Compose files, test Dockerfiles and the browser suite use the catalog digests the verification images were pinned to: 16.16, 17.12 and 18.7. They take the same exception as the clients. The CNPG source images stay on 17.11 and 18.6: CNPG builds of the fixed minors are inside their waiting period, and the exception covers only the PGDG releases.

**Backend Go modules**, from proxy.golang.org: pgx v5.13.0 (2026-11-11), `gorm.io/driver/postgres` v1.6.3 (09-14), `jlaffaye/ftp` v0.2.4 (08-22), `golang.org/x/crypto` v0.59.0 (11-11), gorm v1.31.2 (06-22), gin v1.12.0 (02-28), `gin-contrib/cors` v1.7.11 and `gin-contrib/gzip` v1.2.10 (11-08), kin-openapi v0.153.0 (11-07), testcontainers v0.45.0 (11-05), `moby/moby/api` v1.57.0 and `moby/moby/client` v0.6.3 (11-13), goose v3.30.1 (11-12), `lib/pq` v1.12.3 (04-03), minio-go v7.4.1 (10-12), azcore v1.24.0 and azblob v1.8.3 (11-02), `go-sql-driver/mysql` v1.10.1 (09-02), `klauspost/compress` v1.20.4 (11-09), `pkg/sftp` v1.13.12 (10-23), `golang.org/x/oauth2` v0.38.0 (11-06), gopsutil v4.26.10 (10-24), `caarlos0/env` v11.4.2 (11-02), `go.opentelemetry.io/proto/otlp` v1.11.1 (09-30). goose v3.30.1 requires `golang.org/x/tools` at the 2026-11-06 commit `v0.52.1-0.20261106145633-d407941a339a`, so that pseudo-version replaces the planned v0.52.0. The only adaptation was the `network.Port` argument of the MySQL readiness callback.

**Frontend**, from the npm registry: Mantine 9.8.1 (11-15), `tailwindcss` and `@tailwindcss/vite` 4.3.5 (11-06), vite 8.3.8 (11-11), `@vitejs/plugin-react` 6.3.0 (11-09), React and React DOM 19.2.9 (10-14), `lucide-react` 1.73.0 (11-14), recharts 3.10.1 (07-25), `cron-parser` 5.10.6 (11-14), dayjs 1.11.23 (08-17), i18next 26.7.2 and react-i18next 17.1.13 (11-05), `@testing-library/user-event` 14.6.12 (11-16), postcss 8.5.33 (11-13), `color-name` 2.1.1 (07-22), `@redocly/openapi-core` 1.34.24 (11-05), oxlint 1.94.0 and oxfmt 0.79.0 (11-16), steiger 0.10.0 (11-09), `@feature-sliced/steiger-plugin` 0.8.2 (11-01). The lock now resolves `source-map-js` 1.2.2, and `pnpm audit` finds no vulnerability.

**Website:** eslint 10.15.0 (11-13), `tailwindcss` and `@tailwindcss/postcss` 4.3.5 (11-06), `@types/react` 19.2.18 and `@types/react-dom` 19.2.7, held to 19.2 with `~` so a lock refresh cannot take 19.3. One moderate advisory remains, GHSA-g3hv-75wg-m249 in `minimatch` 10.2.6, reached through `eslint-config-next`; its fix needs `minimatch` 11.

**Tools**, from their GitHub releases and registries: mise 2026.11.9 (11-16, bootstrap hashes equal to the release digests), pnpm 12.21.0 (11-13), hk 2.13.0 (11-16), process-compose 1.124.0 (10-26), goose 3.30.1 (11-12), Python 3.14.10 (tagged 10-25), uv 0.13.8 and ruff 0.17.5 (11-12), ty 0.0.104 (11-13), gh 2.104.0 (11-12), helm 4.3.2 (11-13), vacuum 0.41.0 (11-14), trivy 0.76.2 (11-05), chrome-devtools-mcp 1.13.0 (11-11), typescript-language-server 6.1.0 (11-12), OpenSpec 1.17.0 (11-09), semgrep 1.185.0 (11-10), pytest 9.2.0 (10-27).

**GitHub Actions**, pinned by commit: `actions/download-artifact` v8.0.2 and `actions/upload-artifact` v7.0.2 (10-07), `docker/login-action` v4.8.0 (11-05), `github/codeql-action` v4.39.3 (11-04), `jdx/mise-action` v5.2.9 (11-15). The workflows set `cache: false` and no monorepo roots, so the action's new cache key and monorepo install change nothing.

**Deferred**, with the date their waiting period ends: `golang.org/x/net` v0.63.0, `golang.org/x/tools` v0.53.0 and govulncheck v1.10.0 (after 2026-11-25); Node 24.22.3 (after 11-26); mongo-driver v2.10.1 and `klauspost/compress` v1.20.5 (after 11-27); Playwright 1.65.1 (after 11-28); `oapi-codegen` v2.9.0 with its runtime (after 11-29); MariaDB 12.3.4 (after 12-01); `go-sql-driver/mysql` v1.10.2; MySQL 8.4.13.1, 9.7.4.1 and 26.10.0.1 (after 12-09). Newer releases of mise, hk, uv, ruff, ty, gh, vacuum and pnpm were published after 2026-11-16 and wait for the next pass.

**Open advisories:** CVE-2026-107324 in mongo-driver's `bson.Raw.Validate` stays open, because its fix needs driver 2.10, which drops MongoDB 4.2; backend code does not call the function. GO-2026-5051 in `hirochachacha/go-smb2` has no fix, and govulncheck finds it unreachable.

**EOL lines:** PostgreSQL 12 and 13, PostgreSQL 14 after 14.25, MySQL 5.7 and 8.0, MariaDB 13.0.
