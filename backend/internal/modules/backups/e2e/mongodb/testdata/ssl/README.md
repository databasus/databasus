# MongoDB SSL test certificate

Self-signed cert + key used by the MongoDB SSL test (`containers.StartMongodbSSL`). Not used in
production. The cert validates against `CN=localhost`, but the SSL test connects with
`tlsInsecure`, so any self-signed cert works.

To regenerate (run from this directory):

```
openssl req -x509 -newkey rsa:2048 -keyout server.key -out server.crt -days 3650 -nodes -subj "/CN=localhost"
cat server.crt server.key > server.pem
rm server.key
```

- `server.pem` — combined cert+key, passed as `--tlsCertificateKeyFile`
- `server.crt` — passed as `--tlsCAFile`

MariaDB and MySQL SSL tests don't use these files — those images auto-generate their own cert and
`--require_secure_transport=ON` rejects non-TLS clients. PostgreSQL SSL/mTLS fixtures live under
`e2e/postgresql/testdata/{ssl,mtls}/`.
