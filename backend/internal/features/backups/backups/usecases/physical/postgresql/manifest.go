package usecases_physical_postgresql

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// manifestFileEntry is one row of the reconstructed backup_manifest "Files"
// array, captured while walking the plaintext tar. It holds only metadata —
// the file body is streamed through SHA-256 and never retained.
type manifestFileEntry struct {
	Path        string
	Size        int64
	ModTime     time.Time
	ChecksumHex string
}

// serializeManifestInput is the data serializeManifest needs beyond the file
// list: the values that come from pg_basebackup stderr / the source row rather
// than from the tar itself.
type serializeManifestInput struct {
	Files      []manifestFileEntry
	SystemID   uint64
	TimelineID int
	StartLSN   walmath.LSN
	StopLSN    walmath.LSN
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

// isManifestFile reports whether a tar entry belongs in the backup_manifest
// "Files" array: every regular file EXCEPT raw WAL segments and a stray
// backup_manifest (absent under --no-manifest; skipped defensively). Go's
// tar.Reader normalizes the deprecated TypeRegA to TypeReg, so checking TypeReg
// alone covers pg_basebackup's ustar output.
func isManifestFile(header *tar.Header) bool {
	if header.Typeflag != tar.TypeReg {
		return false
	}

	if header.Name == "backup_manifest" {
		return false
	}

	return !isRawWalSegment(header.Name)
}

// newCodecDecompressor wraps r in the decompressor for codec, returning the
// plaintext-tar reader and a closeFn that MUST be deferred: zstd.Decoder owns
// background goroutines and gzip.Reader holds buffers, so leaking either leaks
// resources per backup.
func newCodecDecompressor(
	r io.Reader,
	codec physical_enums.PhysicalBackupCompression,
) (io.Reader, func() error, error) {
	switch codec {
	case physical_enums.PhysicalBackupCompressionZstd:
		decoder, err := zstd.NewReader(r)
		if err != nil {
			return nil, nil, fmt.Errorf("open zstd reader: %w", err)
		}

		return decoder, func() error { decoder.Close(); return nil }, nil

	case physical_enums.PhysicalBackupCompressionGzip:
		reader, err := gzip.NewReader(r)
		if err != nil {
			return nil, nil, fmt.Errorf("open gzip reader: %w", err)
		}

		return reader, reader.Close, nil

	case physical_enums.PhysicalBackupCompressionNone:
		return r, func() error { return nil }, nil

	default:
		return nil, nil, fmt.Errorf("unknown compression codec %q", codec)
	}
}

// walkTarForManifest reads the codec-compressed tar produced by pg_basebackup
// and returns one manifestFileEntry per regular file that belongs in the
// reconstructed backup_manifest.
//
// The invariants below are what make this correct and safe on TB-scale backups;
// they look like magic to a future reader, so they are spelled out:
//
//  1. r is the LIVE teed pg_basebackup stream, read concurrently with the
//     storage upload (the caller tees stdout into both this reader and
//     SaveFile). We never re-read the artifact back from storage.
//  2. tar is strictly sequential: every 512-byte header carries the member's
//     exact size, so tar.Reader hands back a per-file-bounded reader and we
//     stream the body through SHA-256 between file boundaries. No file — not
//     even a 1 GB segment — is ever held whole in RAM.
//  3. Skipped (non-manifest) members are realigned automatically: tar.Reader.Next
//     discards any unread bytes of the current entry before parsing the next
//     header, so a plain `continue` is safe on a non-seekable pipe.
//  4. Memory is bounded by file COUNT (the compact entry list), never by data
//     size — the two tee branches are independent, and this one parses tar while
//     the storage branch stores opaque compressed bytes.
func walkTarForManifest(
	r io.Reader,
	codec physical_enums.PhysicalBackupCompression,
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

// serializeManifest emits the byte-exact PostgreSQL backup_manifest (version 2)
// for the given files and WAL range, including the trailing Manifest-Checksum.
//
// The layout is PG's own (backup_manifest.c). The Manifest-Checksum is a SHA-256
// over the manifest body up to but NOT including the "Manifest-Checksum" line
// (PG flips still_checksumming off right before appending it), so we build the
// body, hash it, then append the checksum line. encoding/json cannot express
// this self-referential layout — and would also HTML-escape paths differently
// from PG's escape_json — so the body is assembled by hand.
func serializeManifest(in serializeManifestInput) ([]byte, error) {
	var body bytes.Buffer

	// bytes.Buffer.WriteString cannot return a non-nil error (it panics with
	// ErrTooLarge on OOM instead), so every append below discards it. This closure
	// keeps the always-nil discard in one place, silencing the unhandled-error
	// inspection without an unchecked call per line.
	write := func(text string) { _, _ = body.WriteString(text) }

	// Header (InitializeBackupManifest).
	write(`{ "PostgreSQL-Backup-Manifest-Version": 2,` + "\n")
	write(`"System-Identifier": ` + strconv.FormatUint(in.SystemID, 10) + ",\n")
	write(`"Files": [`)

	// Files (AddFileToBackupManifest): the first entry is prefixed with "\n", the
	// rest with ",\n"; entries are emitted in tar order, which is the order they
	// were collected.
	for i, file := range in.Files {
		if i == 0 {
			write("\n")
		} else {
			write(",\n")
		}

		escapedPath, err := escapeJSONString(file.Path)
		if err != nil {
			return nil, fmt.Errorf("escape manifest path %q: %w", file.Path, err)
		}

		write(`{ "Path": ` + escapedPath + ", ")
		write(`"Size": ` + strconv.FormatInt(file.Size, 10) + ", ")
		write(`"Last-Modified": "` + file.ModTime.UTC().Format("2006-01-02 15:04:05") + ` GMT"`)
		write(`, "Checksum-Algorithm": "SHA256", "Checksum": "` + file.ChecksumHex + `"`)
		write(" }")
	}

	// WAL ranges (AddWALInfoToBackupManifest): a single range, since a backup
	// never spans timelines (start tli == stop tli), so PG's loop emits one entry.
	write("\n],\n")
	write(`"WAL-Ranges": [` + "\n")
	write(fmt.Sprintf(`{ "Timeline": %d, "Start-LSN": "%s", "End-LSN": "%s" }`,
		in.TimelineID, in.StartLSN.String(), in.StopLSN.String()))
	write("\n],\n")

	// Manifest-Checksum (SendBackupManifest): SHA-256 of everything above. Sum256
	// fully consumes body.Bytes() before we append, so the later write is safe.
	sum := sha256.Sum256(body.Bytes())
	write(`"Manifest-Checksum": "` + hex.EncodeToString(sum[:]) + `"}` + "\n")

	return body.Bytes(), nil
}

// escapeJSONString renders s as a JSON string literal byte-for-byte the way
// PostgreSQL's escape_json does — NOT encoding/json, which additionally
// HTML-escapes <, >, and &. PG iterates bytes: the named control chars become
// their short escapes, other bytes < 0x20 become \u00xx, and every other byte
// (including UTF-8 lead/continuation bytes >= 0x80) passes through verbatim.
//
// PG emits a non-UTF-8 path as a hex "Encoded-Path" field instead; we do not
// implement that branch (paths are ASCII in practice) and fail loudly rather
// than serialize a manifest a restore would reject.
func escapeJSONString(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", errors.New("non-UTF-8 string not supported (PG would emit Encoded-Path)")
	}

	var b strings.Builder

	// strings.Builder.WriteString/WriteByte cannot return a non-nil error (they
	// panic on OOM, like bytes.Buffer), so these closures discard the always-nil
	// error in one place. writeByte stays byte-wise on purpose: bytes >= 0x80 must
	// pass through verbatim, and string(c) would re-encode them as UTF-8.
	write := func(text string) { _, _ = b.WriteString(text) }
	writeByte := func(c byte) { _ = b.WriteByte(c) }

	writeByte('"')

	for i := range len(s) {
		c := s[i]

		switch c {
		case '\b':
			write(`\b`)
		case '\f':
			write(`\f`)
		case '\n':
			write(`\n`)
		case '\r':
			write(`\r`)
		case '\t':
			write(`\t`)
		case '"':
			write(`\"`)
		case '\\':
			write(`\\`)
		default:
			if c < 0x20 {
				write(fmt.Sprintf(`\u%04x`, c))
			} else {
				writeByte(c)
			}
		}
	}

	writeByte('"')

	return b.String(), nil
}
