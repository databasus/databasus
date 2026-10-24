Pre-built DB client binaries committed to the repo so that local dev,
CI, and the Docker image all read from one place. The Go backend
resolves them at runtime via `runtime.GOOS`+`runtime.GOARCH` →
`assets/tools/<arch-key>/<db>/<db>-<v>/bin/<command>`.

The checklist for adding a version and for refreshing a patch level is the
`client-tool-bundles` skill in `.agents/skills/client-tool-bundles/SKILL.md`.

Layout (one subtree per arch, identical shape):

```
assets/tools/<arch>/
  postgresql/postgresql-{12,13,14,15,16,17,18}/bin/
    pg_dump, pg_restore, psql
  mysql/mysql-{5.7,8.0,8.4,9,26}/bin/
    mysql, mysqldump
  mysql/mysql-26/lib/private/
    libssl.so.3, libcrypto.so.3
  mariadb/mariadb-{10.6,12.3,13.0}/bin/
    mariadb, mariadb-dump
  mongodb/bin/
    mongodump, mongorestore
```

`<arch>` keys (mapping in `backend/internal/util/tools/paths.go`):

| GOOS / GOARCH     | key   | size    |
|-------------------|-------|---------|
| `linux` / `amd64` | `x64` | ~157 MB |
| `linux` / `arm64` | `arm` | ~146 MB |

A directory name is a release line, not a patch level: `mysql-26` holds one
26.x build and serves every MySQL 26.x server, `mariadb-13.0` one 13.0.x build
and every MariaDB 13.x server. The startup check runs each binary's version
flag and rejects a bundle whose version belongs to a different line, so the
name and the bytes cannot drift apart unnoticed.

Bundled PostgreSQL minors (identical on both arches, Debian bookworm pgdg):

| major | minor | source package                    |
|-------|-------|-----------------------------------|
| 12    | 12.22 | `12.22-3.pgdg12+1` (final, EOL)   |
| 13    | 13.23 | `13.23-1.pgdg12+1` (final, EOL)   |
| 14    | 14.23 | `14.23-1.pgdg12+1`                |
| 15    | 15.18 | `15.18-1.pgdg12+1`                |
| 16    | 16.14 | `16.14-1.pgdg12+1`                |
| 17    | 17.10 | `17.10-1.pgdg12+1`                |
| 18    | 18.4  | `18.4-1.pgdg12+1`                 |

Majors 12 and 13 are past upstream EOL, so those minors are the last ones that
will ever exist. Keep every binary within one major on the same package build —
a `pg_dump` older than its sibling `pg_basebackup` is how issue #725 (pg_dump
18.1 silently emitting wrong sequence values) survived unnoticed.

Bundled MySQL patches (identical on both arches, upstream glibc2.28 tarballs
from `https://cdn.mysql.com/Downloads/MySQL-<dir>/mysql-<patch>-linux-glibc2.28-{x86_64,aarch64}.tar.xz`,
where `<dir>` is the line for 5.7 through 9 and `<major>.<minor>` from 26 on,
for example `MySQL-26.7`):

| bundle     | patch  | notes                                     |
|------------|--------|-------------------------------------------|
| `mysql-5.7`  | 5.7.44 | final patch of the line; amd64 only      |
| `mysql-8.0`  | 8.0.46 |                                           |
| `mysql-8.4`  | 8.4.11 | long-term support line                    |
| `mysql-9`    | 9.7.2  | serves every 9.x server                   |
| `mysql-26`   | 26.7.0 | calendar scheme; serves every 26.x server; ships its own OpenSSL |

Bundled MariaDB patches, unpacked from the vendor's own package repositories
by `tools/refresh-mariadb-bundle.sh`:

| bundle         | patch    | source repository and suite            |
|----------------|----------|----------------------------------------|
| `mariadb-10.6` | 10.6.28  | `repo/10.6/ubuntu`, `ubu2204`          |
| `mariadb-12.3` | 12.3.3   | `repo/12.3/debian`, `deb12` (bookworm) |
| `mariadb-13.0` | 13.0.2   | `repo/13.0/ubuntu`, `ubu2204`          |

MariaDB ships three client tiers, mapped to server versions in
`backend/internal/util/tools/mariadb.go`:

- `10.6` (legacy) serves MariaDB 5.5 and 10.1. It exists because the newer
  clients query `generation_expression`, a column MariaDB added in 10.2.
  In 10.6 the real programs are still named `mysql` and `mysqldump`; the
  refresh script installs them under the `mariadb` names the backend uses.
- `12.3` (modern) serves every line from 10.2 through 12.x.
- `13.0` serves the 13 line, which the 12.3 client predates.

Bundled MongoDB Database Tools, `mongodump` and `mongorestore` copied
unmodified from the vendor's `ubuntu2204` archives listed in
`https://downloads.mongodb.org/tools/db/release.json`:

| arch  | release  | archive                                                                                           | archive sha256                                                     |
|-------|----------|---------------------------------------------------------------------------------------------------|--------------------------------------------------------------------|
| `x64` | 100.19.1 | `https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2204-x86_64-100.19.1.tgz`       | `8312f251c360ddfbcdd430c64758bee58b7050e1041e8c9df2906761098acd49` |
| `arm` | 100.19.1 | `https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2204-arm64-100.19.1.tgz`        | `e5d37e679ff815c45a86c17bc17de200c3a215a6eafc82516e5ba271f563373e` |

One release serves every supported server: the vendor lists 4.2 through 9.0
for 100.19.1, and the backend accepts every release of the 9 line. Take both
architectures from the same build family; `debian12` has no arm64 archive.
MongoDB 4.0 is not supported (wire version 7, requires older mongodump).

Notes:
- MySQL `5.7` is amd64-only — `arm/mysql/mysql-5.7/` is intentionally absent,
  because upstream never built a 5.7 client for aarch64. A MySQL 5.7 server is
  refused at the connection test on an arm64 deployment.

Shared libraries the binaries need from the runtime image
(`debian:bookworm-slim`, glibc 2.36; every bundled binary targets glibc 2.28
or lower, except the MongoDB tools, whose `ubuntu2204` build needs glibc 2.34):

| library                             | needed by                                    |
|-------------------------------------|----------------------------------------------|
| `libc`, `libm`, `libgcc_s`, `libstdc++` | everything                                |
| `libssl.so.3`, `libcrypto.so.3`     | MySQL 8.0 to 9 and MariaDB clients, `libpq`   |
| `libz.so.1`, `libzstd.so.1`, `liblz4.so.1` | MariaDB clients, `libpq`               |
| `libncurses.so.6`, `libtinfo.so.6`  | MySQL 8.0+ and MariaDB interactive clients    |
| `libncurses.so.5`, `libtinfo.so.5`  | the MySQL 5.7 interactive client only         |
| `libreadline.so.8`                  | `psql` for PostgreSQL 13 and later            |
| `libedit.so.2`                      | every `mariadb` interactive client, and `psql` for PostgreSQL 12 |
| `libpq.so.5`                        | the PostgreSQL clients                        |
| `libgssapi_krb5.so.2`               | the MongoDB tools                             |

The MySQL 26 clients are the exception for OpenSSL: they call symbols from
OpenSSL 3.2, while bookworm ships 3.0. Their `RUNPATH` is
`$ORIGIN/../lib/private`, so `mysql-26/lib/private/` holds the `libssl.so.3`
and `libcrypto.so.3` (OpenSSL 3.5.7) from the same upstream tarball as the
binaries, and the loader picks them before the image's copies.

`libedit.so.2` is not installed explicitly in the `Dockerfile`: it reaches the
image as a dependency of `postgresql-common`. A change that drops that package
takes the MariaDB interactive client with it. It is also the one soname a
non-Debian development host is missing — Fedora ships the upstream
`libedit.so.0` — which is what `just run` and `just test` at the repository
root shim around.

To refresh a tool set, run the script for that engine, or drop the
corresponding `bin/` contents in place, and commit. For PostgreSQL the source
is `postgresql-client-<major>` from `apt.postgresql.org` for `bookworm`, both
`amd64` and `arm64`, matching the `debian:bookworm-slim` runtime base in the
Dockerfile. Update the tables above in the same commit.
