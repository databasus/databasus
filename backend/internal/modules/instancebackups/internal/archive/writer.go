package archive

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"io"

	"databasus-backend/internal/lib/crypto"
)

// Writer frames the dump into DUMP records inside the encrypted stream. Only Finish writes END, so
// a dump that failed or was abandoned leaves a file that recovery refuses as incomplete.
type Writer struct {
	encryptor  *crypto.EncryptingWriter
	manifest   Manifest
	buffer     []byte
	digest     hash.Hash
	dumpLength uint64
	isFinished bool
}

// NewWriter writes the plaintext header and the MANIFEST record.
func NewWriter(output io.Writer, masterKey *crypto.MasterKey, manifest Manifest) (*Writer, error) {
	manifest.FormatVersion = int(formatVersion)

	header := make([]byte, 0, outerHeaderLength)
	header = append(header, outerMagic...)
	header = binary.BigEndian.AppendUint16(header, formatVersion)
	header = append(header, manifest.BackupID[:]...)

	if _, err := output.Write(header); err != nil {
		return nil, fmt.Errorf("write the Databasus backup header: %w", err)
	}

	encryptor, _, err := crypto.NewEncryptingWriter(output, masterKey, manifest.BackupID)
	if err != nil {
		return nil, err
	}

	encodedManifest, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode the Databasus backup manifest: %w", err)
	}

	writer := &Writer{
		encryptor:  encryptor,
		manifest:   manifest,
		buffer:     make([]byte, 0, maxDumpRecordSize),
		digest:     sha256.New(),
		dumpLength: 0,
		isFinished: false,
	}

	if err := writer.writeRecord(recordTypeManifest, encodedManifest); err != nil {
		return nil, err
	}

	return writer, nil
}

// Write takes dump bytes.
func (w *Writer) Write(p []byte) (int, error) {
	if w.isFinished {
		return 0, errWriterFinished
	}

	written := len(p)

	for len(p) > 0 {
		taken := min(maxDumpRecordSize-len(w.buffer), len(p))
		w.buffer = append(w.buffer, p[:taken]...)
		p = p[taken:]

		if len(w.buffer) == maxDumpRecordSize {
			if err := w.flushDumpRecord(); err != nil {
				return 0, err
			}
		}
	}

	return written, nil
}

// Finish writes the last DUMP record and END, then seals the stream. Call it only after the dump
// tool exited successfully.
func (w *Writer) Finish() (Summary, error) {
	if w.isFinished {
		return Summary{}, errWriterFinished
	}

	if err := w.flushDumpRecord(); err != nil {
		return Summary{}, err
	}

	summary := Summary{DumpLength: w.dumpLength, DumpSha256: [32]byte(w.digest.Sum(nil))}

	end := make([]byte, 0, endRecordLength)
	end = append(end, w.manifest.BackupID[:]...)
	end = binary.BigEndian.AppendUint64(end, summary.DumpLength)
	end = append(end, summary.DumpSha256[:]...)

	if err := w.writeRecord(recordTypeEnd, end); err != nil {
		return Summary{}, err
	}

	w.isFinished = true

	if err := w.encryptor.Close(); err != nil {
		return Summary{}, fmt.Errorf("seal the Databasus backup: %w", err)
	}

	return summary, nil
}

func (w *Writer) flushDumpRecord() error {
	if len(w.buffer) == 0 {
		return nil
	}

	if err := w.writeRecord(recordTypeDump, w.buffer); err != nil {
		return err
	}

	_, _ = w.digest.Write(w.buffer)
	w.dumpLength += uint64(len(w.buffer))
	w.buffer = w.buffer[:0]

	return nil
}

func (w *Writer) writeRecord(recordType byte, payload []byte) error {
	header := make([]byte, 0, recordHeaderLength)
	header = append(header, recordType)
	header = binary.BigEndian.AppendUint32(header, uint32(len(payload)))

	if _, err := w.encryptor.Write(header); err != nil {
		return fmt.Errorf("write a Databasus backup record: %w", err)
	}

	if _, err := w.encryptor.Write(payload); err != nil {
		return fmt.Errorf("write a Databasus backup record: %w", err)
	}

	return nil
}
