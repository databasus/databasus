package s3

import "errors"

// These are terminal on the read path: retrying re-reads the same wrong bytes, and passing them
// downstream would corrupt a stream the decryptor cannot resynchronise (see chunkedManifest).
var (
	ErrChunkSizeMismatch     = errors.New("s3 chunk size mismatch")
	ErrChunkChecksumMismatch = errors.New("s3 chunk checksum mismatch")
)
