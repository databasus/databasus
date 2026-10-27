package physical

import (
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type Codec string

const (
	CodecZstd Codec = "ZSTD"
	CodecGzip Codec = "GZIP"
	CodecNone Codec = "NONE"
)

func GetCodecFallbackOrder() []Codec {
	return []Codec{CodecZstd, CodecGzip, CodecNone}
}

func (c Codec) GetCompressArgument() string {
	switch c {
	case CodecGzip:
		return "server-gzip:6"
	case CodecNone:
		return "none"
	case CodecZstd:
		return "server-zstd:5"
	default:
		return "server-zstd:5"
	}
}

func isCompressionUnsupportedError(stderr []byte) bool {
	return strings.Contains(string(stderr), "does not support compression with")
}

func newCodecDecompressor(
	r io.Reader,
	codec Codec,
) (io.Reader, func() error, error) {
	switch codec {
	case CodecZstd:
		decoder, err := zstd.NewReader(r)
		if err != nil {
			return nil, nil, fmt.Errorf("open zstd reader: %w", err)
		}

		return decoder, func() error { decoder.Close(); return nil }, nil

	case CodecGzip:
		reader, err := gzip.NewReader(r)
		if err != nil {
			return nil, nil, fmt.Errorf("open gzip reader: %w", err)
		}

		return reader, reader.Close, nil

	case CodecNone:
		return r, func() error { return nil }, nil

	default:
		return nil, nil, fmt.Errorf("unknown compression codec %q", codec)
	}
}
