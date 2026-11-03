## ADDED Requirements

### Requirement: Binary downloads bypass HTTP gzip

Databasus SHALL NOT apply HTTP gzip to responses of the logical backup file download, physical restore-set download or verification-agent binary download operations, including when a client sends `Accept-Encoding: gzip`. Successful downloads SHALL have no `Content-Encoding` header and SHALL deliver the same file or archive bytes as a request with `Accept-Encoding: identity`. Backup files and restore-set artifacts SHALL retain their existing compression formats after decryption. A successful verification-agent binary download SHALL retain its exact `Content-Length` and declared download and version headers. A physical restore set SHALL stream without requiring its complete length in advance.

#### Scenario: Logical backup requested with gzip accepted

- **WHEN** a client with a valid download token requests a logical backup file with `Accept-Encoding: gzip`
- **THEN** the response is `200` without `Content-Encoding`
- **AND** its body is the decrypted backup file in its existing compression format

#### Scenario: Physical restore set requested with gzip accepted

- **WHEN** a client with a valid restore token requests a physical restore set with `Accept-Encoding: gzip`
- **THEN** the response is `200` without `Content-Encoding`
- **AND** its body is the restore-set tar archive with the expected artifacts and integrity manifest

#### Scenario: Verification-agent binary requested with gzip accepted

- **WHEN** a client requests an available verification-agent binary for a supported architecture with `Accept-Encoding: gzip`
- **THEN** the response is `200` without `Content-Encoding`
- **AND** its body is the binary for that architecture, its `Content-Length` equals the delivered byte count, and its declared download and version headers are present

### Requirement: Other responses retain compression negotiation

JSON responses outside the three excluded download operations and eligible text static assets SHALL remain gzip-compressed when the client sends `Accept-Encoding: gzip`. Such a response SHALL declare `Content-Encoding: gzip` and `Vary: Accept-Encoding`, and decompress to the original response content. A request with `Accept-Encoding: identity` SHALL receive a response without HTTP gzip. The existing static-asset exclusions for `.png`, `.gif`, `.jpeg`, `.jpg`, `.ico`, `.svg`, `.pdf` and `.mp4` SHALL remain excluded even when the client accepts gzip. Excluding the download operations SHALL NOT exclude neighboring JSON operations.

#### Scenario: JSON requested with gzip accepted

- **WHEN** a client requests a JSON response outside the excluded download operations with `Accept-Encoding: gzip`
- **THEN** the response declares gzip encoding and varies by `Accept-Encoding`
- **AND** decompressing the body yields the original JSON content

#### Scenario: Eligible static asset requested with gzip accepted

- **WHEN** a client requests an eligible JavaScript or CSS static asset with `Accept-Encoding: gzip`
- **THEN** the response declares gzip encoding and decompresses to the original asset

#### Scenario: Client requests identity encoding

- **WHEN** a client requests a JSON response or text static asset with `Accept-Encoding: identity`
- **THEN** the response has no `Content-Encoding` header and contains the original content

#### Scenario: Excluded static asset requested with gzip accepted

- **WHEN** a client requests a static asset with one of the excluded extensions and sends `Accept-Encoding: gzip`
- **THEN** the response has no `Content-Encoding` header and contains the original asset bytes

#### Scenario: Neighboring JSON operation requested with gzip accepted

- **WHEN** a client requests the logical backup list or the server version with `Accept-Encoding: gzip`
- **THEN** the response is gzip-compressed JSON rather than being excluded with the download operations
