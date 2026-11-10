# syntax=docker.io/docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# Databasus verification image: the official Postgres image plus a curated set of
# common extensions, so a dump's CREATE EXTENSION statements succeed when the
# agent restores a backup for verification. This mirrors how the agent uses
# timescale/timescaledb for TimescaleDB. Built per PG major and published as
# databasus/verification-postgres:<major>.
#
ARG PG_MAJOR=16

FROM docker.io/library/postgres:12@sha256:2f2a8c2a7d10862e7fba2602e304523554f9df8244c632dafe2628ccb398fb5c AS postgres-12
FROM docker.io/library/postgres:13@sha256:4689940c683801b4ab839ab3b0a0a3555a5fe425371422310944e89eca7d8068 AS postgres-13
FROM docker.io/library/postgres:14@sha256:834a046089d76edd1d2e81c4741f3ad5d1853646ededc86c2708936aad31c1ed AS postgres-14
FROM docker.io/library/postgres:15@sha256:1aac5b49cefd627f02816141ca4ad012d6cc19460dfe02a9b5168a105be3fc53 AS postgres-15
FROM docker.io/library/postgres:16@sha256:f9a7b9ebf6f56201f949db745e47bc423c0f0f8294bafe991cc10bf74b029657 AS postgres-16
FROM docker.io/library/postgres:17@sha256:ff87138949ad9a46f32ff23bd6852b52ce024a13dffe9b4087be98292cdd3b4f AS postgres-17
FROM docker.io/library/postgres:18@sha256:e3d24f6906d0760bf4da0dae872415391ededefad23fd471c47dc53b0ddf3c42 AS postgres-18

FROM postgres-${PG_MAJOR}

# ARGs declared before FROM are not visible in the build stage, so redeclare it.
ARG PG_MAJOR
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
