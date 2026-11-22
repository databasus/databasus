package providers_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/minio/minio-go/v7"

	"databasus-backend/internal/lib/testreport"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

const (
	singleObjectFileSize = 1024
	// multiPieceFileSize spans two 16 MiB multipart pieces of one object, the second one partial.
	multiPieceFileSize = 20 * 1024 * 1024
	// The cancelled save is stopped while its second piece is read, after the first one was uploaded.
	cancelledFileSize     = 40 * 1024 * 1024
	cancelledAfterBytes   = 20 * 1024 * 1024
	smallFileName         = "small-file"
	multiPieceFileName    = "multi-piece-file"
	cancelledSaveFileName = "cancelled-file"
)

// runObjectStorageFileCycle sends the bucket every request a backup sends: the connection test, a
// single object, a multipart object, ranged reads, deletes and the abort of a cancelled multipart
// upload, and checks that nothing of the run remains.
func runObjectStorageFileCycle(reporter testreport.Reporter, run *objectStorageRun) {
	reporter.Helper()

	target := run.target
	location := target.getLocation(run.prefix)

	if err := target.provider.TestConnection(reporter.Context(), location); err != nil {
		reporter.Fatalf("connection test of %s failed: %s", target.describe(), target.redactor.redact(err.Error()))
	}

	for _, savedFile := range []struct {
		name string
		size int
	}{
		{smallFileName, singleObjectFileSize},
		{multiPieceFileName, multiPieceFileSize},
	} {
		content := generateObjectStorageContent(savedFile.size)
		saveObjectStorageFile(reporter, run, savedFile.name, content)
		verifyReadBack(reporter, run, savedFile.name, content)
	}

	for _, fileName := range []string{smallFileName, multiPieceFileName} {
		if err := target.provider.DeleteFile(reporter.Context(), getFileAccess(location, fileName)); err != nil {
			reporter.Fatalf("delete of %s from %s failed: %s",
				fileName, target.describe(), target.redactor.redact(err.Error()))
		}
	}

	run.reportLeftovers(reporter, "deleting the files")

	cancelObjectStorageSave(reporter, run)
	run.reportLeftovers(reporter, "cancelling a save")
}

func saveObjectStorageFile(reporter testreport.Reporter, run *objectStorageRun, fileName string, content []byte) {
	reporter.Helper()

	target := run.target
	fileAccess := getFileAccess(target.getLocation(run.prefix), fileName)

	if err := target.provider.SaveFile(reporter.Context(), fileAccess, bytes.NewReader(content)); err != nil {
		reporter.Fatalf("save of %s (%d bytes) to %s failed: %s",
			fileName, len(content), target.describe(), target.redactor.redact(err.Error()))
	}
}

func verifyReadBack(reporter testreport.Reporter, run *objectStorageRun, fileName string, savedContent []byte) {
	reporter.Helper()

	target := run.target
	fileAccess := getFileAccess(target.getLocation(run.prefix), fileName)

	openedFile, err := target.provider.OpenFile(reporter.Context(), fileAccess)
	if err != nil {
		reporter.Fatalf(
			"open of %s from %s failed: %s",
			fileName,
			target.describe(),
			target.redactor.redact(err.Error()),
		)
	}

	defer func() { _ = openedFile.Close() }()

	readHasher := sha256.New()

	readSize, err := io.Copy(readHasher, openedFile)
	if err != nil {
		reporter.Fatalf("read of %s from %s failed after %d bytes: %s",
			fileName, target.describe(), readSize, target.redactor.redact(err.Error()))
	}

	savedDigest := sha256.Sum256(savedContent)
	readDigest := hex.EncodeToString(readHasher.Sum(nil))

	if readSize != int64(len(savedContent)) || readDigest != hex.EncodeToString(savedDigest[:]) {
		reporter.Errorf("file %s read back %d bytes with SHA-256 %s from %s, but %d bytes with SHA-256 %s were saved",
			fileName, readSize, readDigest, target.describe(), len(savedContent), hex.EncodeToString(savedDigest[:]))
	}
}

// cancelObjectStorageSave cancels a save after its first pieces reached the bucket, which leaves an
// open multipart upload that the provider has to abort.
func cancelObjectStorageSave(reporter testreport.Reporter, run *objectStorageRun) {
	reporter.Helper()

	target := run.target
	ctx, cancel := context.WithCancel(reporter.Context())
	defer cancel()

	fileAccess := getFileAccess(target.getLocation(run.prefix), cancelledSaveFileName)
	cancelledFileKey := run.prefix + cancelledSaveFileName
	isPartUploadedBeforeCancel := false

	var partListingErr error

	source := &cancellingReader{
		content:     generateObjectStorageContent(cancelledFileSize),
		offset:      0,
		cancelAfter: cancelledAfterBytes,
		isCancelled: false,
		cancel: func() {
			isPartUploadedBeforeCancel, partListingErr = hasUploadedPartFor(ctx, target, cancelledFileKey)
			cancel()
		},
	}

	if err := target.provider.SaveFile(ctx, fileAccess, source); err == nil {
		reporter.Errorf(
			"save of %s to %s succeeded although it was cancelled",
			cancelledSaveFileName,
			target.describe(),
		)
	}

	if partListingErr != nil {
		reporter.Errorf("could not list the uploaded parts of %s in %s before cancelling its save: %s",
			cancelledSaveFileName, target.describe(), target.redactor.redact(partListingErr.Error()))

		return
	}

	if !isPartUploadedBeforeCancel {
		reporter.Errorf(
			"no uploaded part of %s was in %s when its save was cancelled, so the abort went unchecked",
			cancelledSaveFileName,
			target.describe(),
		)
	}
}

// hasUploadedPartFor reports whether an open multipart upload of a part object of the file already
// holds an uploaded part, so the abort that follows the cancel has data to remove.
func hasUploadedPartFor(ctx context.Context, target objectStorageTarget, fileKey string) (bool, error) {
	uploads, err := listIncompleteUploadsUnder(ctx, target, fileKey)
	if err != nil {
		return false, err
	}

	coreClient := &minio.Core{Client: target.rawClient}

	for _, upload := range uploads {
		listedParts, err := coreClient.ListObjectParts(ctx, target.settings.Bucket, upload.Key, upload.UploadID, 0, 1)
		if err != nil {
			return false, err
		}

		if len(listedParts.ObjectParts) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func getFileAccess(location providers.Location, fileName string) providers.FileAccess {
	return providers.FileAccess{Location: location, FileName: fileName, Logger: providertest.NewDiscardLogger()}
}

func generateObjectStorageContent(size int) []byte {
	content := make([]byte, size)
	_, _ = rand.Read(content)

	return content
}

// cancellingReader cancels the save once cancelAfter bytes were read, while more content remains.
type cancellingReader struct {
	content     []byte
	offset      int
	cancelAfter int
	cancel      func()
	isCancelled bool
}

func (r *cancellingReader) Read(buffer []byte) (int, error) {
	if r.offset >= r.cancelAfter && !r.isCancelled {
		r.isCancelled = true
		r.cancel()
	}

	if r.offset >= len(r.content) {
		return 0, io.EOF
	}

	readCount := copy(buffer, r.content[r.offset:])
	r.offset += readCount

	return readCount, nil
}
