FROM postgres:17@sha256:a63c1ea3f6a3a5a1621f97d2aba3b30faecfab4224c1b0000cc85ced57bb1939
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
