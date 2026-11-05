# PostgreSQL SSL test certificate

Self-signed server cert + key and an SSL-only `pg_hba.conf`, baked into the image built by
`containers.StartPostgresSSL` (see the `Dockerfile` in this directory). Not used in production.
The cert validates against `CN=localhost`, but the SSL test connects with `sslmode=require`, so
any self-signed cert works.

To regenerate (run from this directory):

```
openssl req -x509 -newkey rsa:2048 -keyout server.key -out server.crt -days 3650 -nodes -subj "/CN=localhost"
```

- `server.crt` / `server.key` — `ssl_cert_file` / `ssl_key_file`. The Dockerfile chowns the key to
  `postgres` and chmods it `600`; PostgreSQL refuses a key that is group/world-readable or not owned
  by the server user, which is why these tests build an image instead of copying the key in.
- `pg_hba.conf` — SSL-only auth rules: rejects every plaintext TCP connection so a silent SSL-drop
  regression fails the test instead of falling back to plaintext.
