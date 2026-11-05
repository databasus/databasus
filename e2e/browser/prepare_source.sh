#!/bin/sh
set -eu
printf '%s\n' 'host replication all all scram-sha-256' >> "$PGDATA/pg_hba.conf"
