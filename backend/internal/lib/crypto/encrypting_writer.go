package crypto

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// EncryptingWriter seals everything written to it into the stream format of stream_format.go. Close
// seals the last partial chunk, and it writes the header when nothing was written.
type EncryptingWriter struct {
	output          io.Writer
	cipher          cipher.AEAD
	header          StreamHeader
	buffer          []byte
	chunkIndex      uint64
	isHeaderWritten bool
}

// NewEncryptingWriter returns the writer and the header it seals with. The caller stores the header
// beside the stream, because NewDecryptingReader needs it back.
func NewEncryptingWriter(
	output io.Writer,
	masterKey *MasterKey,
	fileID uuid.UUID,
) (*EncryptingWriter, StreamHeader, error) {
	header, err := newRandomStreamHeader()
	if err != nil {
		return nil, StreamHeader{}, err
	}

	aead, err := newStreamCipher(masterKey, fileID, header.Salt)
	if err != nil {
		return nil, StreamHeader{}, err
	}

	return &EncryptingWriter{
		output:          output,
		cipher:          aead,
		header:          header,
		buffer:          make([]byte, 0, streamChunkSize),
		chunkIndex:      0,
		isHeaderWritten: false,
	}, header, nil
}

func (w *EncryptingWriter) Write(p []byte) (int, error) {
	err := w.writeHeaderOnce()
	if err != nil {
		return 0, err
	}

	w.buffer = append(w.buffer, p...)

	for len(w.buffer) >= streamChunkSize {
		err = w.sealAndWriteChunk(w.buffer[:streamChunkSize])
		if err != nil {
			return 0, err
		}

		w.buffer = w.buffer[streamChunkSize:]
	}

	return len(p), nil
}

func (w *EncryptingWriter) Close() error {
	err := w.writeHeaderOnce()
	if err != nil {
		return err
	}

	if len(w.buffer) == 0 {
		return nil
	}

	err = w.sealAndWriteChunk(w.buffer)
	w.buffer = nil

	return err
}

func (w *EncryptingWriter) writeHeaderOnce() error {
	if w.isHeaderWritten {
		return nil
	}

	_, err := w.output.Write(w.header.marshal())
	if err != nil {
		return fmt.Errorf("write the stream header: %w", err)
	}

	w.isHeaderWritten = true

	return nil
}

func (w *EncryptingWriter) sealAndWriteChunk(chunk []byte) error {
	sealedChunk := w.cipher.Seal(nil, getChunkNonce(w.header.Nonce, w.chunkIndex), chunk, nil)

	sealedChunkLen := uint32(len(sealedChunk)) // a sealed chunk is at most 1 MiB and a 16-byte tag
	lengthPrefix := make([]byte, streamChunkLengthLen)
	binary.BigEndian.PutUint32(lengthPrefix, sealedChunkLen)

	_, err := w.output.Write(lengthPrefix)
	if err != nil {
		return fmt.Errorf("write the chunk length: %w", err)
	}

	_, err = w.output.Write(sealedChunk)
	if err != nil {
		return fmt.Errorf("write the encrypted chunk: %w", err)
	}

	w.chunkIndex++

	return nil
}
