package physical

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type serializeManifestInput struct {
	Files      []manifestFileEntry
	SystemID   uint64
	TimelineID int
	StartLSN   walmath.LSN
	StopLSN    walmath.LSN
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

	write := func(text string) { _, _ = body.WriteString(text) }

	write(`{ "PostgreSQL-Backup-Manifest-Version": 2,` + "\n")
	write(`"System-Identifier": ` + strconv.FormatUint(in.SystemID, 10) + ",\n")
	write(`"Files": [`)

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

	write("\n],\n")
	write(`"WAL-Ranges": [` + "\n")
	write(fmt.Sprintf(`{ "Timeline": %d, "Start-LSN": "%s", "End-LSN": "%s" }`,
		in.TimelineID, in.StartLSN.String(), in.StopLSN.String()))
	write("\n],\n")

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
