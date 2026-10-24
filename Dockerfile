# ========= BUILD FRONTEND =========
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-build

WORKDIR /frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/pnpm/store \
  corepack enable && pnpm install --frozen-lockfile --store-dir /pnpm/store
COPY frontend/ ./

# APP_VERSION changes with every commit and invalidates the build cache of every
# later instruction in its stage, so each stage declares it right before its use.
ARG APP_VERSION=dev
ENV VITE_APP_VERSION=$APP_VERSION
RUN pnpm build

# ========= BUILD BACKEND =========
# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.26.10 AS backend-build

# Make TARGET args available early so tools built here match the final image arch
ARG TARGETOS
ARG TARGETARCH
ARG GOOSE_VERSION=v3.27.3

# Install Go public tools needed in runtime. Use `go build` for goose so the
# binary is compiled for the target architecture instead of downloading a
# prebuilt binary which may have the wrong architecture (causes exec format
# errors on ARM).
RUN git clone --depth 1 --branch "$GOOSE_VERSION" https://github.com/pressly/goose.git /tmp/goose && \
  cd /tmp/goose/cmd/goose && \
  go get golang.org/x/crypto@v0.55.0 && \
  GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
  go build -ldflags "-X main.version=${GOOSE_VERSION}+databasus.1" \
  -o /usr/local/bin/goose . && \
  rm -rf /tmp/goose
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Set working directory
WORKDIR /app

# Install Go dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Create required directories for embedding
RUN mkdir -p /app/ui/build

# Copy frontend build output for embedding
COPY --from=frontend-build /frontend/dist /app/ui/build

# Generate Swagger documentation
COPY backend/ ./
RUN swag init -d . -g cmd/main.go -o swagger

# Compile the backend
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 \
  GOOS=$TARGETOS \
  GOARCH=$TARGETARCH \
  go build -o /app/main ./cmd


# ========= BUILD VERIFICATION AGENT =========
FROM --platform=$BUILDPLATFORM golang:1.26.10 AS verification-agent-build

WORKDIR /agent

COPY agent/verification/go.mod agent/verification/go.sum ./
RUN go mod download

COPY agent/verification/ ./

ARG APP_VERSION=dev

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /verification-agent-binaries/databasus-verification-agent-linux-amd64 ./cmd

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /verification-agent-binaries/databasus-verification-agent-linux-arm64 ./cmd


# ========= RUNTIME =========
# In this final image we ship only what the running app actually needs:
#   - Build-only tooling (compilers, codegen, key fetchers) stays in the earlier
#     stages above and never reaches here.
#   - Anything pulled in for a single build step is purged within the same layer.
#   - We keep this tight because every extra binary widens the attack surface and
#     adds another CVE to track.
FROM debian:bookworm-slim

ARG TARGETARCH
ENV CONTAINER_ARCH=$TARGETARCH

# Set production mode for Docker containers
ENV ENV_MODE=production

# ========= Install all apt packages in a single layer =========
# Base packages and PostgreSQL 17 share one RUN to minimise layer count
# and cache-export overhead.
#
#   - wget: build-only — fetches the repo signing keys, then purged at the end of
#     this RUN (see the "minimal attack surface" note on the runtime stage above).
#   - Repo keys: scoped signed-by keyrings, not the deprecated global apt-key trust
#     store, so a compromised repo key cannot vouch for any other repository.
#   - Codename: hardcoded "bookworm" (base image is pinned), so no lsb-release.
#   - Default cluster: the app runs its own cluster from /databasus-data/pgdata, so
#     the one the package creates at 17/main is never read. Dropped inside this RUN
#     so its data files never reach a layer; the server binaries under
#     /usr/lib/postgresql stay.
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      ca-certificates gosu \
      libncurses5 libncurses6 libmariadb3 libgnutls30 \
      wget; \
    wget -qO /usr/share/keyrings/pgdg.asc https://www.postgresql.org/media/keys/ACCC4CF8.asc; \
    echo "deb [signed-by=/usr/share/keyrings/pgdg.asc] http://apt.postgresql.org/pub/repos/apt bookworm-pgdg main" \
      > /etc/apt/sources.list.d/pgdg.list; \
    apt-get update; \
    apt-get install -y --no-install-recommends postgresql-17; \
    pg_dropcluster --stop 17 main; \
    apt-get purge -y --auto-remove wget; \
    rm -rf /var/lib/apt/lists/*

# ========= Pre-built DB client binaries (PG, MySQL, MariaDB, MongoDB) =========
# All client tools live under /app/assets/tools/<arch>/ — the backend resolves
# them at runtime via runtime.GOARCH. Use a bind mount so only the tree matching
# $TARGETARCH ends up in an image layer (the unused arch never materialises).
ARG TARGETARCH
RUN --mount=type=bind,source=assets/tools,target=/ctx/tools,readonly \
    mkdir -p /app/assets/tools && \
    if [ "$TARGETARCH" = "amd64" ]; then \
      cp -r /ctx/tools/x64 /app/assets/tools/x64; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
      cp -r /ctx/tools/arm /app/assets/tools/arm; \
    fi && \
    chmod +x /app/assets/tools/*/postgresql/*/bin/* \
             /app/assets/tools/*/mysql/*/bin/* \
             /app/assets/tools/*/mariadb/*/bin/* \
             /app/assets/tools/*/mongodb/bin/*

RUN set -eux; \
    groupmod -g 999 postgres; \
    usermod -u 999 postgres; \
    groupmod -n databasus postgres; \
    usermod -l databasus -s /usr/sbin/nologin postgres; \
    chown -R databasus:databasus \
      /var/lib/postgresql /etc/postgresql /var/log/postgresql /run/postgresql; \
    mkdir -p /databasus-data; \
    chown databasus:databasus /databasus-data

WORKDIR /app

# Copy Goose from build stage
COPY --from=backend-build /usr/local/bin/goose /usr/local/bin/goose

# Copy app binary
COPY --from=backend-build /app/main .

# Expose the binary as the `databasus` command on PATH (e.g. `databasus healthcheck`)
RUN ln -s /app/main /usr/local/bin/databasus

# Copy migrations directory
COPY backend/migrations ./migrations

# Copy UI files
COPY --from=backend-build /app/ui/build ./ui/build

# Copy verification agent binaries (both architectures) — served by the backend
# at GET /api/v1/system/verification-agent?arch=amd64|arm64
RUN mkdir -p ./agent-binaries
COPY --from=verification-agent-build /verification-agent-binaries/* ./agent-binaries/

# The image carries no .env file. Every setting comes from the operator (-e,
# compose, k8s) or from the default the backend defines in code, so nothing
# from a development environment can reach a running instance.

COPY --chmod=0755 docker/start.sh /app/start.sh

LABEL org.opencontainers.image.source="https://github.com/databasus/databasus"

ARG APP_VERSION=dev
LABEL org.opencontainers.image.version=$APP_VERSION
ENV APP_VERSION=$APP_VERSION

EXPOSE 4005

# Liveness probe: the runtime image ships no wget/curl, so the binary checks
# itself. Targets the dependency-free /system/version endpoint (not the deep
# /system/health) so a degraded-but-serving instance is never restarted.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD ["databasus", "healthcheck"]

# Volume for PostgreSQL data
VOLUME ["/databasus-data"]

# start.sh starts as root to prepare the data volume, then re-executes itself as the databasus user through gosu.
# nosemgrep: dockerfile.security.missing-user-entrypoint.missing-user-entrypoint
ENTRYPOINT ["/app/start.sh"]
# nosemgrep: dockerfile.security.missing-user.missing-user
CMD []
