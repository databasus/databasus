#!/bin/sh
set -eu

mkdir -p /var/lib/postgresql/patroni
mkdir -p /var/lib/postgresql/configuration
touch /var/lib/postgresql/configuration/postgresql.conf /var/lib/postgresql/configuration/pg_ident.conf
chown -R postgres:postgres /var/lib/postgresql/configuration
chown postgres:postgres /var/lib/postgresql/patroni
chmod 0700 /var/lib/postgresql/patroni
exec gosu postgres patroni /etc/patroni.json
