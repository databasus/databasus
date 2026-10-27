package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

// An encrypted stream starts with a 64-byte header: the magic bytes, the salt, the nonce and reserved
// zero bytes. Chunks of up to streamChunkSize plain bytes follow, each sealed with AES-256-GCM and
// prefixed with its sealed length as a big-endian uint32. Stored backup files of every earlier
// release use this format, so it never changes.
const (
	streamMagicBytes       = "PGRSUS01"
	streamSaltLen          = 32
	streamNonceLen         = 12
	streamReservedLen      = 12
	streamHeaderLen        = len(streamMagicBytes) + streamSaltLen + streamNonceLen + streamReservedLen
	streamChunkSize        = 1 * 1024 * 1024
	streamChunkLengthLen   = 4
	streamKeyLen           = 32
	streamPBKDF2Iterations = 100000
)

// StreamHeader holds the random values an encrypted stream was sealed with. The caller stores them
// beside the stream and passes them back to decrypt it, so a stream swapped for another one is refused.
type StreamHeader struct {
	Salt  []byte
	Nonce []byte
}

// newStreamCipher derives the key of one stream from the master key, the stream's own file ID and its
// salt, so no two streams share a key.
func newStreamCipher(masterKey *MasterKey, fileID uuid.UUID, salt []byte) (cipher.AEAD, error) {
	if masterKey == nil || masterKey.GetSecret() == "" {
		return nil, ErrEmptyMasterKey
	}

	keyMaterial := []byte(masterKey.GetSecret() + fileID.String())
	derivedKey := pbkdf2.Key(keyMaterial, salt, streamPBKDF2Iterations, streamKeyLen, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("create the stream cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create the stream GCM: %w", err)
	}

	return aead, nil
}

func newRandomStreamHeader() (StreamHeader, error) {
	header := StreamHeader{Salt: make([]byte, streamSaltLen), Nonce: make([]byte, streamNonceLen)}

	_, err := rand.Read(header.Salt)
	if err != nil {
		return StreamHeader{}, fmt.Errorf("generate the stream salt: %w", err)
	}

	_, err = rand.Read(header.Nonce)
	if err != nil {
		return StreamHeader{}, fmt.Errorf("generate the stream nonce: %w", err)
	}

	return header, nil
}

func (h StreamHeader) validateLengths() error {
	if len(h.Salt) != streamSaltLen {
		return fmt.Errorf("salt must be %d bytes, got %d: %w", streamSaltLen, len(h.Salt), ErrMalformedStreamHeader)
	}

	if len(h.Nonce) != streamNonceLen {
		return fmt.Errorf("nonce must be %d bytes, got %d: %w", streamNonceLen, len(h.Nonce), ErrMalformedStreamHeader)
	}

	return nil
}

func (h StreamHeader) marshal() []byte {
	header := make([]byte, streamHeaderLen)
	saltStart := len(streamMagicBytes)
	nonceStart := saltStart + streamSaltLen

	copy(header, streamMagicBytes)
	copy(header[saltStart:nonceStart], h.Salt)
	copy(header[nonceStart:nonceStart+streamNonceLen], h.Nonce)

	return header
}

func unmarshalStreamHeader(header []byte) (StreamHeader, error) {
	saltStart := len(streamMagicBytes)
	nonceStart := saltStart + streamSaltLen

	magic := string(header[:saltStart])
	if magic != streamMagicBytes {
		return StreamHeader{}, fmt.Errorf(
			"expected %s, got %s: %w", streamMagicBytes, magic, ErrInvalidStreamMagicBytes,
		)
	}

	return StreamHeader{
		Salt:  header[saltStart:nonceStart],
		Nonce: header[nonceStart : nonceStart+streamNonceLen],
	}, nil
}

// getChunkNonce puts the chunk's index into the last 8 bytes of the stream nonce, so every chunk is
// sealed under a nonce of its own and chunks cannot be reordered.
func getChunkNonce(streamNonce []byte, chunkIndex uint64) []byte {
	chunkNonce := make([]byte, streamNonceLen)
	copy(chunkNonce, streamNonce)
	binary.BigEndian.PutUint64(chunkNonce[streamNonceLen-8:], chunkIndex)

	return chunkNonce
}
