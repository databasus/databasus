package crypto

import (
	"bytes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// DecryptingReader opens a stream that EncryptingWriter sealed. It authenticates every chunk before
// returning any of its bytes, so a changed or cut chunk ends the read with an error.
type DecryptingReader struct {
	input       io.Reader
	cipher      cipher.AEAD
	streamNonce []byte
	buffer      []byte
	chunkIndex  uint64
	isEOF       bool
}

// NewDecryptingReader reads the stream's header at once and refuses a stream whose salt or nonce
// differs from header, which is what a stream swapped for another file looks like.
func NewDecryptingReader(
	input io.Reader,
	masterKey *MasterKey,
	fileID uuid.UUID,
	header StreamHeader,
) (*DecryptingReader, error) {
	err := header.validateLengths()
	if err != nil {
		return nil, err
	}

	aead, err := newStreamCipher(masterKey, fileID, header.Salt)
	if err != nil {
		return nil, err
	}

	err = readAndCheckStreamHeader(input, header)
	if err != nil {
		return nil, err
	}

	return &DecryptingReader{
		input:       input,
		cipher:      aead,
		streamNonce: header.Nonce,
		buffer:      nil,
		chunkIndex:  0,
		isEOF:       false,
	}, nil
}

func (r *DecryptingReader) Read(p []byte) (int, error) {
	for len(r.buffer) < len(p) && !r.isEOF {
		err := r.readAndOpenChunk()
		if errors.Is(err, io.EOF) {
			r.isEOF = true

			break
		}

		if err != nil {
			return 0, err
		}
	}

	if len(r.buffer) == 0 {
		return 0, io.EOF
	}

	n := copy(p, r.buffer)
	r.buffer = r.buffer[n:]

	return n, nil
}

func readAndCheckStreamHeader(input io.Reader, expectedHeader StreamHeader) error {
	rawHeader := make([]byte, streamHeaderLen)

	_, err := io.ReadFull(input, rawHeader)
	if err != nil {
		return fmt.Errorf("read the stream header: %w", err)
	}

	storedHeader, err := unmarshalStreamHeader(rawHeader)
	if err != nil {
		return err
	}

	if !bytes.Equal(storedHeader.Salt, expectedHeader.Salt) {
		return fmt.Errorf("salt: %w", ErrStreamHeaderMismatch)
	}

	if !bytes.Equal(storedHeader.Nonce, expectedHeader.Nonce) {
		return fmt.Errorf("nonce: %w", ErrStreamHeaderMismatch)
	}

	return nil
}

// readAndOpenChunk returns io.EOF only when the stream ends exactly before a chunk's length prefix.
func (r *DecryptingReader) readAndOpenChunk() error {
	lengthPrefix := make([]byte, streamChunkLengthLen)

	_, err := io.ReadFull(r.input, lengthPrefix)
	if err != nil {
		return err
	}

	sealedChunkLen := binary.BigEndian.Uint32(lengthPrefix)
	if sealedChunkLen == 0 || sealedChunkLen > streamChunkSize+uint32(r.cipher.Overhead()) {
		return fmt.Errorf("chunk length %d: %w", sealedChunkLen, ErrInvalidStreamChunkLength)
	}

	sealedChunk := make([]byte, sealedChunkLen)

	_, err = io.ReadFull(r.input, sealedChunk)
	if err != nil {
		return fmt.Errorf("read the encrypted chunk: %w", err)
	}

	chunk, err := r.cipher.Open(nil, getChunkNonce(r.streamNonce, r.chunkIndex), sealedChunk, nil)
	if err != nil {
		return fmt.Errorf("decrypt chunk %d: %w: %w", r.chunkIndex, ErrStreamChunkAuthentication, err)
	}

	r.buffer = append(r.buffer, chunk...)
	r.chunkIndex++

	return nil
}
