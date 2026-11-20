package recovery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"os"
	"syscall"
	"unicode/utf8"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/diskusage"
)

const (
	maxMasterKeyBytes = 64 << 10
	// diskCheckIntervalBytes is how much of the archive is written between two free-space checks.
	diskCheckIntervalBytes = 64 << 20
	uploadBufferBytes      = 1 << 20
)

const (
	archiveField   = "archive"
	masterKeyField = "masterKey"
)

type uploadReceiverSpec struct {
	body            *multipart.Reader
	archivePath     string
	maxArchiveBytes int64
	inspectDisk     func(string) (diskusage.Usage, error)
	diskPath        string
}

// receivedUpload is what an upload staged: the fingerprint of the archive copy and the exact bytes
// of the entered key.
type receivedUpload struct {
	archiveSha256 string
	keyContent    []byte
}

// uploadReceiver reads the multipart body of a recovery upload. It streams the archive into a
// private file with bounded memory and keeps the key, at most 64 KiB, in memory.
type uploadReceiver struct {
	spec                uploadReceiverSpec
	isArchiveReceived   bool
	isMasterKeyReceived bool
	archiveSha256       string
	keyContent          []byte
}

func newUploadReceiver(spec uploadReceiverSpec) *uploadReceiver {
	return &uploadReceiver{
		spec:                spec,
		isArchiveReceived:   false,
		isMasterKeyReceived: false,
		archiveSha256:       "",
		keyContent:          nil,
	}
}

// receive leaves a partial archive file behind on failure; the caller removes the operation
// directory that holds it.
func (r *uploadReceiver) receive() (receivedUpload, error) {
	for {
		part, err := r.spec.body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return receivedUpload{}, newUploadIncompleteError(err)
		}

		err = r.receivePart(part)
		_ = part.Close()

		if err != nil {
			return receivedUpload{}, err
		}
	}

	if err := r.checkComplete(); err != nil {
		return receivedUpload{}, err
	}

	return receivedUpload{archiveSha256: r.archiveSha256, keyContent: r.keyContent}, nil
}

func (r *uploadReceiver) receivePart(part *multipart.Part) error {
	switch part.FormName() {
	case masterKeyField:
		return r.receiveMasterKey(part)
	case archiveField:
		return r.receiveArchive(part)
	default:
		return nil
	}
}

func (r *uploadReceiver) receiveMasterKey(part io.Reader) error {
	value, isWithinLimit, err := readLimited(part, maxMasterKeyBytes)
	if err != nil {
		return newUploadIncompleteError(err)
	}

	if !isWithinLimit {
		return newFieldError(masterKeyField, "must be at most 64 KiB")
	}

	r.keyContent = value
	r.isMasterKeyReceived = true

	return nil
}

func (r *uploadReceiver) receiveArchive(part io.Reader) error {
	if r.isArchiveReceived {
		return newFieldError(archiveField, "must be sent once")
	}

	r.isArchiveReceived = true

	if err := r.checkDiskSpace(); err != nil {
		return err
	}

	archiveSha256, err := r.writeArchive(part)
	if err != nil {
		return err
	}

	r.archiveSha256 = archiveSha256

	return nil
}

func (r *uploadReceiver) checkComplete() error {
	if !r.isArchiveReceived {
		return newFieldError(archiveField, "is required")
	}

	if !r.isMasterKeyReceived || len(r.keyContent) == 0 {
		return newFieldError(masterKeyField, "is required")
	}

	if !utf8.Valid(r.keyContent) {
		return newFieldError(masterKeyField, "must be UTF-8 text")
	}

	if _, err := crypto.NewMasterKey(r.keyContent); err != nil {
		return newFieldError(masterKeyField, "is shorter than a Databasus master key")
	}

	return nil
}

// writeArchive copies the archive part into a new private file while hashing it, and stops at the
// first byte beyond the limit, whatever length the client declared.
func (r *uploadReceiver) writeArchive(part io.Reader) (string, error) {
	file, err := os.OpenFile(r.spec.archivePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateFileMode)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errStagingWrite, err)
	}

	digest := sha256.New()
	copyErr := r.copyArchive(file, digest, part)
	syncErr := file.Sync()
	closeErr := file.Close()

	if copyErr != nil {
		return "", copyErr
	}

	if err := errors.Join(syncErr, closeErr); err != nil {
		return "", newStagingWriteError(err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

func (r *uploadReceiver) copyArchive(file *os.File, digest hash.Hash, part io.Reader) error {
	buffer := make([]byte, uploadBufferBytes)

	var writtenBytes, uncheckedBytes int64

	for {
		readCount, readErr := part.Read(buffer)
		if readCount > 0 {
			writtenBytes += int64(readCount)
			if writtenBytes > r.spec.maxArchiveBytes {
				return ErrArchiveTooLarge
			}

			if _, err := file.Write(buffer[:readCount]); err != nil {
				return newStagingWriteError(err)
			}

			digest.Write(buffer[:readCount])

			uncheckedBytes += int64(readCount)
			if uncheckedBytes >= diskCheckIntervalBytes {
				uncheckedBytes = 0

				if err := r.checkDiskSpace(); err != nil {
					return err
				}
			}
		}

		if errors.Is(readErr, io.EOF) {
			return nil
		}

		if readErr != nil {
			return newUploadIncompleteError(readErr)
		}
	}
}

// checkDiskSpace keeps the headroom the journal and logs need, so a large upload never fills the
// data volume.
func (r *uploadReceiver) checkDiskSpace() error {
	usage, err := r.spec.inspectDisk(r.spec.diskPath)
	if err != nil {
		return fmt.Errorf("check free space for the staged recovery: %w", err)
	}

	if usage.FreeBytes < stagingHeadroomBytes {
		return ErrInsufficientDiskSpace
	}

	return nil
}

// readLimited reads at most limit bytes and reports whether the part ended within them.
func readLimited(part io.Reader, limit int64) ([]byte, bool, error) {
	var buffer bytes.Buffer

	if _, err := io.Copy(&buffer, io.LimitReader(part, limit+1)); err != nil {
		return nil, false, err
	}

	if int64(buffer.Len()) > limit {
		return nil, false, nil
	}

	return buffer.Bytes(), true, nil
}

func newStagingWriteError(err error) error {
	if errors.Is(err, syscall.ENOSPC) {
		return ErrInsufficientDiskSpace
	}

	return fmt.Errorf("%w: %w", errStagingWrite, err)
}

func newUploadIncompleteError(err error) error {
	return fmt.Errorf("%w: %w",
		apierror.NewInvalidFieldsProblem([]apierror.FieldError{
			{Field: archiveField, Detail: "the upload ended before the request was complete"},
		}),
		err)
}

func newFieldError(field, detail string) error {
	return apierror.NewInvalidFieldsProblem([]apierror.FieldError{{Field: field, Detail: detail}})
}
