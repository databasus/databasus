package archive

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/crypto"
)

// Reader yields the dump bytes of a Databasus backup file. It returns io.EOF only after END matched
// the dump's length and digest and the stream ended right after it, so a consumer that read to
// io.EOF has read a complete, unmodified dump.
type Reader struct {
	decryptor  *crypto.DecryptingReader
	manifest   Manifest
	pending    []byte
	digest     hash.Hash
	dumpLength uint64
	isEnded    bool
	failure    error
}

// Open reads the plaintext header and the MANIFEST record. A key that does not decrypt the first
// chunk answers ErrMasterKeyMismatch.
func Open(input io.Reader, masterKey *crypto.MasterKey) (*Reader, error) {
	backupID, err := readOuterHeader(input)
	if err != nil {
		return nil, err
	}

	streamHeader, stream, err := crypto.ReadStreamHeader(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArchiveIncomplete, err)
	}

	decryptor, err := crypto.NewDecryptingReader(stream, masterKey, backupID, streamHeader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArchiveCorrupted, err)
	}

	reader := &Reader{
		decryptor:  decryptor,
		manifest:   Manifest{},
		pending:    nil,
		digest:     sha256.New(),
		dumpLength: 0,
		isEnded:    false,
		failure:    nil,
	}

	recordType, payload, err := reader.readRecord(maxManifestLength, true)
	if err != nil {
		return nil, err
	}

	if recordType != recordTypeManifest {
		return nil, fmt.Errorf("%w: the file does not start with a manifest", ErrArchiveCorrupted)
	}

	if err := json.Unmarshal(payload, &reader.manifest); err != nil {
		return nil, fmt.Errorf("%w: the manifest is not readable: %w", ErrArchiveCorrupted, err)
	}

	if reader.manifest.BackupID != backupID || reader.manifest.FormatVersion != int(formatVersion) {
		return nil, fmt.Errorf("%w: the manifest does not match the header", ErrArchiveCorrupted)
	}

	return reader, nil
}

// Validate reads the whole file and returns its manifest, or the reason it cannot be recovered.
func Validate(input io.Reader, masterKey *crypto.MasterKey) (Manifest, error) {
	reader, err := Open(input, masterKey)
	if err != nil {
		return Manifest{}, err
	}

	if _, err := io.Copy(io.Discard, reader); err != nil {
		return Manifest{}, err
	}

	return reader.GetManifest(), nil
}

func (r *Reader) GetManifest() Manifest {
	return r.manifest
}

func (r *Reader) Read(p []byte) (int, error) {
	for len(r.pending) == 0 {
		if r.failure != nil {
			return 0, r.failure
		}

		if r.isEnded {
			return 0, io.EOF
		}

		r.failure = r.advance()
	}

	copied := copy(p, r.pending)
	r.pending = r.pending[copied:]

	return copied, nil
}

// advance reads the next record. A DUMP record becomes pending; END is checked against the dump
// read so far and against the end of the stream.
func (r *Reader) advance() error {
	recordType, payload, err := r.readRecord(max(maxDumpRecordSize, endRecordLength), false)
	if err != nil {
		return err
	}

	switch recordType {
	case recordTypeDump:
		if len(payload) == 0 {
			return fmt.Errorf("%w: an empty dump record", ErrArchiveCorrupted)
		}

		_, _ = r.digest.Write(payload)
		r.dumpLength += uint64(len(payload))
		r.pending = payload

		return nil
	case recordTypeEnd:
		if err := r.checkEnd(payload); err != nil {
			return err
		}

		r.isEnded = true

		return nil
	default:
		return fmt.Errorf("%w: an unexpected record", ErrArchiveCorrupted)
	}
}

func (r *Reader) checkEnd(payload []byte) error {
	if len(payload) != endRecordLength {
		return fmt.Errorf("%w: the end record has a wrong length", ErrArchiveCorrupted)
	}

	backupID := uuid.UUID(payload[:16])
	dumpLength := binary.BigEndian.Uint64(payload[16:24])
	dumpSha256 := payload[24:]

	if backupID != r.manifest.BackupID || dumpLength != r.dumpLength || !bytes.Equal(dumpSha256, r.digest.Sum(nil)) {
		return fmt.Errorf("%w: the dump does not match its end record", ErrArchiveCorrupted)
	}

	trailing := make([]byte, 1)

	count, err := r.decryptor.Read(trailing)
	if count > 0 {
		return fmt.Errorf("%w: data follows the end record", ErrArchiveCorrupted)
	}

	if !errors.Is(err, io.EOF) {
		return classifyStreamError(err, false)
	}

	return nil
}

func (r *Reader) readRecord(maxLength int, isFirst bool) (byte, []byte, error) {
	header := make([]byte, recordHeaderLength)

	if _, err := io.ReadFull(r.decryptor, header); err != nil {
		return 0, nil, classifyStreamError(err, isFirst)
	}

	length := int(binary.BigEndian.Uint32(header[1:]))
	if length > maxLength {
		return 0, nil, fmt.Errorf("%w: a record is longer than allowed", ErrArchiveCorrupted)
	}

	payload := make([]byte, length)

	if _, err := io.ReadFull(r.decryptor, payload); err != nil {
		return 0, nil, classifyStreamError(err, isFirst)
	}

	return header[0], payload, nil
}

func readOuterHeader(input io.Reader) (uuid.UUID, error) {
	header := make([]byte, outerHeaderLength)

	if _, err := io.ReadFull(input, header); err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", ErrNotDatabasusBackup, err)
	}

	if string(header[:len(outerMagic)]) != outerMagic {
		return uuid.Nil, ErrNotDatabasusBackup
	}

	versionStart := len(outerMagic)
	if binary.BigEndian.Uint16(header[versionStart:versionStart+2]) != formatVersion {
		return uuid.Nil, fmt.Errorf("%w: an unknown format version", ErrNotDatabasusBackup)
	}

	return uuid.UUID(header[versionStart+2:]), nil
}

// classifyStreamError tells a wrong key from damage: only the first chunk failing authentication
// is a key mismatch, a stream that stops early is incomplete, and anything else is corruption.
func classifyStreamError(err error, isFirst bool) error {
	switch {
	case errors.Is(err, crypto.ErrStreamChunkAuthentication) && isFirst:
		return fmt.Errorf("%w: %w", ErrMasterKeyMismatch, err)
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return fmt.Errorf("%w: %w", ErrArchiveIncomplete, err)
	default:
		return fmt.Errorf("%w: %w", ErrArchiveCorrupted, err)
	}
}
