## ADDED Requirements

### Requirement: A logical backup download announces its exact file length

A successful logical backup file download SHALL include `Content-Length` equal to the byte count of the complete downloaded file. The length SHALL describe the decrypted file in its existing compression format, including when the backup is encrypted in storage. The server SHALL begin streaming without first reading the complete file into memory or staging another copy. The announced length SHALL be exact rather than an estimate rounded to megabytes.

#### Scenario: Unencrypted backup download

- **WHEN** a client downloads a completed logical backup stored without encryption
- **THEN** `Content-Length` equals the number of file bytes the response delivers
- **AND** the file retains its existing compression format and download name

#### Scenario: Encrypted backup download

- **WHEN** a client downloads a completed logical backup encrypted in storage
- **THEN** the delivered file is decrypted and `Content-Length` equals its byte count
- **AND** encryption overhead in storage is not included in the announced length

#### Scenario: Download size is not a whole megabyte

- **WHEN** a client downloads a completed logical backup whose file size is not an exact multiple of a megabyte
- **THEN** `Content-Length` equals the exact delivered byte count without rounding to a whole megabyte

#### Scenario: Client accepts gzip

- **WHEN** a client downloads a completed logical backup with `Accept-Encoding: gzip`
- **THEN** the response has no `Content-Encoding` header and its `Content-Length` equals the delivered file byte count

#### Scenario: Stored file is truncated

- **WHEN** a completed logical backup's stored file has been truncated and a client downloads it
- **THEN** the announced length remains the complete backup's recorded byte count, allowing the client to detect the incomplete response
- **AND** the download does not change the backup's stored completed status
