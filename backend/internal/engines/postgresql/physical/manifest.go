package physical

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"
)

type manifestFileEntry struct {
	Path        string
	Size        int64
	ModTime     time.Time
	ChecksumHex string
}

// rawWalSegmentRe matches a raw WAL segment file name under pg_wal — 24 upper
// hex chars (timeline + log + segment). These are the only regular files PG
// writes into the tar WITHOUT a backup_manifest entry (basebackup.c emits them
// with a bare _tarWriteHeader), so they are the exact set we exclude from
// Files[]. Crucially, pg_wal/archive_status/*.done and pg_wal/*.history are
// regular files PG DOES list in the manifest, so this must match segments only —
// never all of pg_wal/.
var rawWalSegmentRe = regexp.MustCompile(`^pg_wal/[0-9A-F]{24}$`)

func isRawWalSegment(tarPath string) bool {
	return rawWalSegmentRe.MatchString(tarPath)
}

func isManifestFile(header *tar.Header) bool {
	if header.Typeflag != tar.TypeReg {
		return false
	}

	if header.Name == "backup_manifest" {
		return false
	}

	return !isRawWalSegment(header.Name)
}

func walkTarForManifest(
	r io.Reader,
	codec Codec,
) ([]manifestFileEntry, error) {
	decompressed, closeDecompressor, err := newCodecDecompressor(r, codec)
	if err != nil {
		return nil, err
	}
	defer func() { _ = closeDecompressor() }()

	tarReader := tar.NewReader(decompressed)
	hasher := sha256.New()

	var entries []manifestFileEntry

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar header: %w", err)
		}

		if !isManifestFile(header) {
			continue
		}

		hasher.Reset()

		// Each entry is streamed into the hasher, never buffered, so its size does not affect memory.
		// nosemgrep: go.lang.security.decompression_bomb.potential-dos-via-decompression-bomb
		if _, err := io.Copy(hasher, tarReader); err != nil {
			return nil, fmt.Errorf("hash tar entry %q: %w", header.Name, err)
		}

		entries = append(entries, manifestFileEntry{
			Path:        header.Name,
			Size:        header.Size,
			ModTime:     header.ModTime,
			ChecksumHex: hex.EncodeToString(hasher.Sum(nil)),
		})
	}

	return entries, nil
}
