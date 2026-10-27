FROM postgres:18
COPY server.key /etc/ssl-test/server.key
COPY server.crt /etc/ssl-test/server.crt
COPY pg_hba.conf /etc/ssl-test/pg_hba.conf
RUN chown postgres:postgres /etc/ssl-test/server.key /etc/ssl-test/server.crt /etc/ssl-test/pg_hba.conf \
 && chmod 600 /etc/ssl-test/server.key \
 && chmod 644 /etc/ssl-test/server.crt /etc/ssl-test/pg_hba.conf
CMD ["postgres", \
     "-c", "ssl=on", \
     "-c", "ssl_cert_file=/etc/ssl-test/server.crt", \
     "-c", "ssl_key_file=/etc/ssl-test/server.key", \
     "-c", "hba_file=/etc/ssl-test/pg_hba.conf"]
