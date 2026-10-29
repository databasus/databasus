# syntax=docker/dockerfile:1

# Databasus verification image: the official Postgres image plus a curated set of
# common extensions, so a dump's CREATE EXTENSION statements succeed when the
# agent restores a backup for verification. This mirrors how the agent uses
# timescale/timescaledb for TimescaleDB. Built per PG major and published as
# databasus/verification-postgres:<major>.
#
# BASE_IMAGE is digest-pinned by CI (postgres:<major>@sha256:...). The tag-only
# default exists solely for local `docker build --build-arg PG_MAJOR=16` runs.
ARG PG_MAJOR=16
ARG BASE_IMAGE=postgres:${PG_MAJOR}

FROM ${BASE_IMAGE}

# ARGs declared before FROM are not visible in the build stage, so redeclare it.
ARG PG_MAJOR

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
    rm -rf /var/lib/apt/lists/*; \
    printf "shared_preload_libraries = '%s'\n" \
        "${PRELOAD_LIBRARIES}${version_specific_preload_libraries}" \
        >> "/usr/share/postgresql/${PG_MAJOR}/postgresql.conf.sample"
