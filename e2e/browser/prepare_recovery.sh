#!/bin/sh
set -eu

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y --no-install-recommends curl zstd docker-cli
rm -rf /var/lib/apt/lists/*
chown postgres:postgres /restore-work
chgrp postgres /run/recovery-engine/docker.sock
chmod g+rw /run/recovery-engine/docker.sock

docker info >/dev/null
docker image load --input /postgres.tar
docker image inspect "$E2E_POSTGRES_IMAGE" >/dev/null
touch /tmp/recovery-ready
exec sleep infinity
