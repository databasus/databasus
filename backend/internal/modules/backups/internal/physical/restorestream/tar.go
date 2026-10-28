package restorestream

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"time"
)

// streamSizedTarEntry writes a regular-file entry whose length is known up front,
// streaming the body straight through without buffering it. Used for the compressed
// backup blobs, which can be many gigabytes.
func streamSizedTarEntry(
	tarWriter *tar.Writer,
	name string,
	mode, size int64,
	reader io.Reader,
	checksums *checksumLedger,
) error {
	header := &tar.Header{
		Name:     name,
		Mode:     mode,
		Size:     size,
		Typeflag: tar.TypeReg,
		ModTime:  time.Unix(0, 0).UTC(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("write tar header %q: %w", name, err)
	}

	hasher := checksums.begin(name)
	if _, err := io.Copy(io.MultiWriter(tarWriter, hasher), reader); err != nil {
		return fmt.Errorf("write tar body %q: %w", name, err)
	}

	checksums.commit(name, hasher)

	return nil
}

// streamTarEntry writes a single regular-file entry of unknown length: the tar
// header needs the size up front, so the (small, 16 MB-or-less) body is read
// fully into memory first. Used for manifests, WAL segments and history files,
// never for arbitrarily large backup blobs (those go through streamSizedTarEntry).
func streamTarEntry(
	tarWriter *tar.Writer,
	name string,
	mode int64,
	reader io.Reader,
	checksums *checksumLedger,
) error {
	body, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read %q body: %w", name, err)
	}

	return writeTarBytes(tarWriter, name, mode, body, checksums)
}

func writeTarBytes(
	tarWriter *tar.Writer,
	name string,
	mode int64,
	body []byte,
	checksums *checksumLedger,
) error {
	return streamSizedTarEntry(tarWriter, name, mode, int64(len(body)), bytes.NewReader(body), checksums)
}
