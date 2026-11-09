## MODIFIED Requirements

### Requirement: Archiving keeps up with a busy source when the storage is far away

For a storage of type S3-compatible, Azure Blob, SFTP, FTP, SMB, rclone or local, four concurrent writers SHALL store at least two pairs per second of a 16 MB segment-sized file and a 1 KB metadata file, when every response from the storage arrives 50 ms later than on a local network. These are the files WAL archiving writes for each segment.

#### Scenario: Storage in another region

- **WHEN** a storage answers every request 50 ms late and four concurrent writers store twenty-four pairs of a 16 MB file and a 1 KB file
- **THEN** all twenty-four pairs are stored within twelve seconds

#### Scenario: Local storage

- **WHEN** four concurrent writers store twenty-four pairs of a 16 MB file and a 1 KB file in a local storage
- **THEN** all twenty-four pairs are stored within twelve seconds

### Requirement: A storage that stops answering does not hold an upload

For a storage of type S3-compatible, Azure Blob, SFTP, FTP or SMB, when the storage stops answering in the middle of a write, the write SHALL fail no later than three seconds after the deadline of the operation that started it, and the connection it used SHALL be closed. A later write SHALL NOT reuse that connection. rclone is not covered, because how it waits depends on the remote service behind it.

#### Scenario: Storage goes silent mid-write

- **WHEN** a storage stops answering while a segment is being written and the segment's upload deadline passes
- **THEN** the write fails within three seconds after the deadline
- **AND** the upload slot is free for the next segment
- **AND** the next write to that storage opens a new connection
