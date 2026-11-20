package crypto

import (
	"bytes"
	"fmt"
	"io"
)

// ReadStreamHeader reads the header an encrypted stream starts with, for a self-contained file that
// stores no header beside it. The returned reader yields the whole stream from its first byte, so
// NewDecryptingReader still checks the header it reads against the one returned here.
func ReadStreamHeader(input io.Reader) (StreamHeader, io.Reader, error) {
	headerBytes := make([]byte, streamHeaderLen)

	if _, err := io.ReadFull(input, headerBytes); err != nil {
		return StreamHeader{}, nil, fmt.Errorf("%w: %w", ErrMalformedStreamHeader, err)
	}

	header, err := unmarshalStreamHeader(headerBytes)
	if err != nil {
		return StreamHeader{}, nil, err
	}

	ownedHeader := StreamHeader{Salt: bytes.Clone(header.Salt), Nonce: bytes.Clone(header.Nonce)}

	return ownedHeader, io.MultiReader(bytes.NewReader(headerBytes), input), nil
}
