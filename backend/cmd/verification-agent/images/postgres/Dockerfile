# syntax=docker.io/docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# Databasus verification image: the official Postgres image plus a curated set of
# common extensions, so a dump's CREATE EXTENSION statements succeed when the
# agent restores a backup for verification. This mirrors how the agent uses
# timescale/timescaledb for TimescaleDB. Built per PG major and published as
# databasus/verification-postgres:<major>.<minor>, the tag the agent restores
# into, and as :<major> and :<major>-v<Databasus version> for manual use.
#
ARG PG_MAJOR=16

FROM docker.io/library/postgres:12@sha256:2f2a8c2a7d10862e7fba2602e304523554f9df8244c632dafe2628ccb398fb5c AS postgres-12
FROM docker.io/library/postgres:13@sha256:4689940c683801b4ab839ab3b0a0a3555a5fe425371422310944e89eca7d8068 AS postgres-13
FROM docker.io/library/postgres:14@sha256:d4f56810bfc965cb9f2db30d2631f79d107efb02bf9f2431fc4889178555cb95 AS postgres-14
FROM docker.io/library/postgres:15@sha256:5a407822929c3789ff5154d3d35bd7e2090318149344c8a3d4ca942b035822ed AS postgres-15
FROM docker.io/library/postgres:16@sha256:63e83b5a3dc36a917dccb76fe1bd2ac51f1ec9c1deb47ff20d4f04233bf6eae9 AS postgres-16
FROM docker.io/library/postgres:17@sha256:846a7ddaa67a4b354ae1d21087b683901aa6113a4ed626d3787cfbf6c85e3e07 AS postgres-17
FROM docker.io/library/postgres:18@sha256:b9f6b8ff03b62f3cfd9d9ba3fe56226a9991c79d3fb46fd46536263bf02be553 AS postgres-18

FROM postgres-${PG_MAJOR}

# ARGs declared before FROM are not visible in the build stage, so redeclare it.
ARG PG_MAJOR
# The exact server version this image is published under, such as 18.7. The
# release reads it from the agent's pinned PostgreSQL versions.
ARG PG_SERVER_VERSION
ARG TARGETARCH
ARG VCHORD_VERSION=1.1.1
COPY vectorchord.sha256 /opt/vectorchord.sha256

# Extensions whose CREATE EXTENSION works once the files are present, plus the
# Tier-2 monitoring extensions that additionally need shared_preload_libraries
# (set below). The official image already configures the apt.postgresql.org repo
# these packages come from. Every entry must be installable from PGDG apt for all
# majors 12-18. pg_uuidv7 and DocumentDB are installed separately because PGDG
# only packages them for majors 13-18 and 15-18 respectively, so a PG 12 dump
# that uses pg_uuidv7 still fails to restore. A missing package fails the build
# because the agent tolerates missing extensions at restore time and could
# otherwise verify an incomplete restore.
#
# Intentionally NOT bundled: topn (Citus), pg_hashids, anonymizer
# (postgresql_anonymizer) and pg_stat_monitor (Percona). PGDG apt does not
# package these. They need source builds or Percona's own repo. Add them via
# that route (not this list) if verification coverage for them is ever required.
ARG EXTENSION_PACKAGES="\
postgresql-${PG_MAJOR}-postgis-3 \
postgresql-${PG_MAJOR}-pgrouting \
postgresql-${PG_MAJOR}-h3 \
postgresql-${PG_MAJOR}-pgvector \
postgresql-${PG_MAJOR}-rum \
postgresql-${PG_MAJOR}-partman \
postgresql-${PG_MAJOR}-repack \
postgresql-${PG_MAJOR}-hypopg \
postgresql-${PG_MAJOR}-hll \
postgresql-${PG_MAJOR}-cron \
postgresql-${PG_MAJOR}-set-user \
postgresql-${PG_MAJOR}-pgaudit \
postgresql-${PG_MAJOR}-ip4r \
postgresql-${PG_MAJOR}-semver \
postgresql-${PG_MAJOR}-http \
postgresql-${PG_MAJOR}-mysql-fdw \
postgresql-${PG_MAJOR}-tds-fdw \
postgresql-${PG_MAJOR}-pg-stat-kcache \
postgresql-${PG_MAJOR}-pg-qualstats \
postgresql-${PG_MAJOR}-pg-wait-sampling \
postgresql-plpython3-${PG_MAJOR} \
postgresql-plperl-${PG_MAJOR} \
postgresql-pltcl-${PG_MAJOR}"

# Tier-2 extensions refuse to CREATE unless their library is preloaded. This list
# must match the packages installed above: a name whose library is absent makes
# Postgres refuse to start. Appending to the initdb sample means every fresh
# PGDATA (the verification container is ephemeral) inherits it; the agent does not
# pass -c shared_preload_libraries, so this value stands at runtime.
ARG PRELOAD_LIBRARIES="pg_stat_statements,pg_cron,pg_stat_kcache,pg_qualstats,pg_wait_sampling,pgaudit,set_user"

# The build installs packages as root, the base image's user. At runtime the agent
# starts the container as the postgres user, so nothing in it runs as root.
RUN set -eux; \
    version_specific_extension_packages=""; \
    version_specific_preload_libraries=""; \
    case "${PG_MAJOR}" in \
        13|14|15|16|17|18) \
            version_specific_extension_packages="postgresql-${PG_MAJOR}-pg-uuidv7" \
            ;; \
    esac; \
    case "${PG_MAJOR}" in \
        15|16|17|18) \
            version_specific_extension_packages="${version_specific_extension_packages} postgresql-${PG_MAJOR}-documentdb"; \
            version_specific_preload_libraries=",pg_documentdb_core,pg_documentdb" \
            ;; \
    esac; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        ${EXTENSION_PACKAGES} ${version_specific_extension_packages}; \
    case "${PG_MAJOR}" in \
        14|15|16|17|18) \
            apt-get install -y --no-install-recommends ca-certificates curl; \
            curl --fail --location --silent --show-error \
                --output /tmp/vchord.deb \
                "https://github.com/tensorchord/VectorChord/releases/download/${VCHORD_VERSION}/postgresql-${PG_MAJOR}-vchord_${VCHORD_VERSION}-1_${TARGETARCH}.deb"; \
            vchord_file="postgresql-${PG_MAJOR}-vchord_${VCHORD_VERSION}-1_${TARGETARCH}.deb"; \
            vchord_sha256="$(awk -v name="${vchord_file}" '$2 == name { print $1 }' /opt/vectorchord.sha256)"; \
            test "${#vchord_sha256}" -eq 64; \
            printf '%s  %s\n' "${vchord_sha256}" /tmp/vchord.deb | sha256sum --check --strict; \
            apt-get install -y --no-install-recommends /tmp/vchord.deb; \
            rm /tmp/vchord.deb; \
            version_specific_preload_libraries="${version_specific_preload_libraries},vchord" \
            ;; \
    esac; \
    rm -rf /var/lib/apt/lists/*; \
    printf "shared_preload_libraries = '%s'\n" \
        "${PRELOAD_LIBRARIES}${version_specific_preload_libraries}" \
        >> "/usr/share/postgresql/${PG_MAJOR}/postgresql.conf.sample"

# The agent restores into the tag's exact server version, and installing the
# extensions above can upgrade the server past the base image. Refuse an image
# whose base or installed server is another version than the one it is
# published under.
RUN set -eu; \
    if [ -z "${PG_SERVER_VERSION:-}" ]; then \
        echo "PG_SERVER_VERSION is required" >&2; \
        exit 1; \
    fi; \
    base_version="${PG_VERSION%%-*}"; \
    installed_version="$(postgres --version | awk '{ print $3 }')"; \
    if [ "${base_version}" != "${PG_SERVER_VERSION}" ] || [ "${installed_version}" != "${PG_SERVER_VERSION}" ]; then \
        echo "verification image for PostgreSQL ${PG_SERVER_VERSION}: base image is ${base_version}, installed server is ${installed_version}" >&2; \
        exit 1; \
    fi
