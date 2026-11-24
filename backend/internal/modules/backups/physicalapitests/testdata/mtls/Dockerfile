FROM docker.io/library/postgres:17@sha256:846a7ddaa67a4b354ae1d21087b683901aa6113a4ed626d3787cfbf6c85e3e07
COPY server.key /etc/ssl-test/server.key
COPY server.crt /etc/ssl-test/server.crt
COPY ca.crt /etc/ssl-test/ca.crt
COPY pg_hba.conf /etc/ssl-test/pg_hba.conf
RUN chown postgres:postgres /etc/ssl-test/server.key /etc/ssl-test/server.crt /etc/ssl-test/ca.crt /etc/ssl-test/pg_hba.conf \
 && chmod 600 /etc/ssl-test/server.key \
 && chmod 644 /etc/ssl-test/server.crt /etc/ssl-test/ca.crt /etc/ssl-test/pg_hba.conf
# The upstream entrypoint initializes ownership as root and execs PostgreSQL through gosu.
# nosemgrep: dockerfile.security.missing-user.missing-user
CMD ["postgres", \
     "-c", "ssl=on", \
     "-c", "ssl_cert_file=/etc/ssl-test/server.crt", \
     "-c", "ssl_key_file=/etc/ssl-test/server.key", \
     "-c", "ssl_ca_file=/etc/ssl-test/ca.crt", \
     "-c", "hba_file=/etc/ssl-test/pg_hba.conf", \
     "-c", "wal_level=logical", \
     "-c", "summarize_wal=on", \
     "-c", "max_wal_senders=10", \
     "-c", "max_replication_slots=10", \
     "-c", "wal_keep_size=512MB"]
