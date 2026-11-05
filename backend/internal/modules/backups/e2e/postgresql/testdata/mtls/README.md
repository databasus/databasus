# mTLS test certificates

Certificate chain for the `test-logical-postgres-mtls` container, which enforces mutual TLS -
it rejects any TCP client that does not present a client certificate signed by
`ca.crt`. This reproduces a Cloud SQL instance with "Require trusted client
certificates" enabled. Not used in production.

- `ca.crt` / `ca.key` - test certificate authority that signs the server and client certs
- `server.crt` / `server.key` - server cert, CA-signed, `CN=localhost`, SAN `localhost,127.0.0.1`
- `client.crt` / `client.key` - client cert, CA-signed, `CN=testuser`
- `pg_hba.conf` - requires `clientcert=verify-ca` on every TLS connection

The container mounts `ca.crt` as `ssl_ca_file`; the backup/restore tests pass
`client.crt`, `client.key`, and `ca.crt` through the database API.

To regenerate (run from this directory, requires Bash for the SAN process substitution):

```
openssl req -x509 -newkey rsa:2048 -keyout ca.key -out ca.crt -days 3650 -nodes \
  -subj "/CN=Databasus Test CA"

openssl req -newkey rsa:2048 -keyout server.key -out server.csr -nodes \
  -subj "/CN=localhost"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 3650 \
  -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1")

openssl req -newkey rsa:2048 -keyout client.key -out client.csr -nodes \
  -subj "/CN=testuser"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days 3650

rm server.csr client.csr ca.srl
```
