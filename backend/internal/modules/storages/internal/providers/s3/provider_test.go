package s3

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	test_utils "databasus-backend/internal/lib/httpclienttest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

// testInnerPartSize is the S3 minimum non-final part size (5 MiB). Combined with a tiny per-object
// part count it forces object rollover on a few tens of MiB instead of the production ~15.6 GiB.
const testInnerPartSize = 5 * 1024 * 1024

const testBucketName = "test-bucket"

func TestOpenFile(t *testing.T) {
	t.Run("returns the content of a file stored as one object without a manifest", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(4096)

		fileName := uuid.NewString()
		putRawObject(t, rawClient, testedBucket.settings.Bucket, fileName, persistedBytes)

		reassembledBytes := readWholeFile(t, testedBucket, fileName)

		assert.Equal(t, persistedBytes, reassembledBytes)
	})

	t.Run("fails the read when a part of a chunked file is missing", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(23 * 1024 * 1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		require.NoError(
			t,
			rawClient.RemoveObject(
				t.Context(),
				testedBucket.settings.Bucket,
				fileName+".part000002",
				minio.RemoveObjectOptions{},
			),
		)

		reader, err := testedBucket.provider.OpenFile(t.Context(), testedBucket.access(fileName))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		_, err = io.ReadAll(reader)
		assert.Error(t, err, "a missing chunk must surface as a read error, never a silently truncated stream")
	})

	t.Run("returns a checksum error when a part of a chunked file is corrupted", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(23 * 1024 * 1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		// Overwrite a committed 10 MiB part with same-length but different bytes: the size still matches
		// the manifest, so only the per-part sha256 can catch the corruption.
		corruptedSameSize := bytes.Repeat([]byte{0xAB}, 10*1024*1024)
		putRawObject(t, rawClient, testedBucket.settings.Bucket, fileName+".part000002", corruptedSameSize)

		reader, err := testedBucket.provider.OpenFile(t.Context(), testedBucket.access(fileName))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		_, err = io.ReadAll(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("returns a size error when a part of a chunked file is truncated", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(23 * 1024 * 1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		// Replace a committed 10 MiB part with a shorter object: the manifest expects 10 MiB, so the
		// length check must reject it before its bytes reach the restore.
		putRawObject(t, rawClient, testedBucket.settings.Bucket, fileName+".part000002", generateBytes(4*1024*1024))

		reader, err := testedBucket.provider.OpenFile(t.Context(), testedBucket.access(fileName))
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		_, err = io.ReadAll(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "size mismatch")
	})

	t.Run("fails when the manifest cannot be parsed", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(23 * 1024 * 1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		// A manifest that exists but cannot be parsed must fail loudly, not fall through to the
		// single-object path and silently read nothing.
		putRawObject(
			t,
			rawClient,
			testedBucket.settings.Bucket,
			fileName+manifestSuffix,
			[]byte("not a valid manifest{"),
		)

		_, err := testedBucket.provider.OpenFile(t.Context(), testedBucket.access(fileName))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "manifest")
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("removes a file stored as one object without a manifest", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)

		fileName := uuid.NewString()
		putRawObject(t, rawClient, testedBucket.settings.Bucket, fileName, generateBytes(4096))

		require.NoError(t, testedBucket.provider.DeleteFile(t.Context(), testedBucket.access(fileName)))

		assert.Empty(t, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName))

		_, err := testedBucket.provider.OpenFile(t.Context(), testedBucket.access(fileName))
		assert.Error(t, err)
	})

	t.Run("removes every part and the manifest of a chunked file", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(23 * 1024 * 1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)
		require.NotEmpty(t, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName))

		require.NoError(t, testedBucket.provider.DeleteFile(t.Context(), testedBucket.access(fileName)))

		assert.Empty(t, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName))
	})

	// Issue #827: Garage before v2.4.0 answers a batch delete of an absent key with NoSuchKey, which
	// kept every deletion pending forever.
	t.Run("succeeds when the server reports the keys as missing", func(t *testing.T) {
		testedBucket := newStubBucket(t, startS3BatchDeleteServer(t, s3ErrCodeNoSuchKey))

		assert.NoError(t, testedBucket.provider.DeleteFile(t.Context(), testedBucket.access("backup")))
	})

	t.Run("names the key when the server denies its deletion", func(t *testing.T) {
		testedBucket := newStubBucket(t, startS3BatchDeleteServer(t, s3ErrCodeAccessDenied))

		err := testedBucket.provider.DeleteFile(t.Context(), testedBucket.access("backup"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "object backup")
		assert.Contains(t, err.Error(), s3ErrCodeAccessDenied)
	})

	t.Run("removes the parts when they have no manifest", func(t *testing.T) {
		testedBucket, client, bucketName := startS3CleanupTarget(t)
		fileName := "orphan-chunks"

		for index := 1; index <= 3; index++ {
			putObject(t, client, bucketName, fmt.Sprintf("%s.part%06d", fileName, index), "chunk")
		}

		require.NoError(t, deleteThroughProvider(t, testedBucket, fileName))

		for index := 1; index <= 3; index++ {
			key := fmt.Sprintf("%s.part%06d", fileName, index)
			assert.False(t, objectExists(t, client, bucketName, key), "chunk %s survived cleanup", key)
		}
	})

	t.Run("leaves a file alone when its name only starts with the deleted one", func(t *testing.T) {
		testedBucket, client, bucketName := startS3CleanupTarget(t)

		putObject(t, client, bucketName, "backup-1", "one")
		putObject(t, client, bucketName, "backup-1.part000001", "one chunk")
		putObject(t, client, bucketName, "backup-10", "ten")
		putObject(t, client, bucketName, "backup-10.part000001", "ten chunk")
		putObject(t, client, bucketName, "backup-1.metadata", "sidecar")

		require.NoError(t, deleteThroughProvider(t, testedBucket, "backup-1"))

		assert.False(t, objectExists(t, client, bucketName, "backup-1"))
		assert.False(t, objectExists(t, client, bucketName, "backup-1.part000001"))
		assert.True(t, objectExists(t, client, bucketName, "backup-10"), "a similarly named object must survive")
		assert.True(t, objectExists(t, client, bucketName, "backup-10.part000001"))
		assert.True(t, objectExists(t, client, bucketName, "backup-1.metadata"), "a sidecar is its own logical file")
	})

	t.Run("aborts the multipart upload when the upload was interrupted", func(t *testing.T) {
		testedBucket, client, bucketName := startS3CleanupTarget(t)
		fileName := "interrupted-upload"

		coreClient := minio.Core{Client: client}

		uploadID, err := coreClient.NewMultipartUpload(t.Context(), bucketName, fileName, minio.PutObjectOptions{})
		require.NoError(t, err)

		_, err = coreClient.PutObjectPart(
			t.Context(), bucketName, fileName, uploadID, 1,
			bytes.NewReader(make([]byte, 5*1024*1024)), 5*1024*1024, minio.PutObjectPartOptions{},
		)
		require.NoError(t, err)

		require.True(t, hasIncompleteUpload(t, client, bucketName, fileName))

		require.NoError(t, deleteThroughProvider(t, testedBucket, fileName))

		assert.False(t, hasIncompleteUpload(t, client, bucketName, fileName),
			"an interrupted upload keeps consuming storage until it is aborted")
	})

	t.Run("succeeds when called twice", func(t *testing.T) {
		testedBucket, client, bucketName := startS3CleanupTarget(t)
		fileName := "twice"

		putObject(t, client, bucketName, fileName, "payload")

		require.NoError(t, deleteThroughProvider(t, testedBucket, fileName))
		assert.NoError(t, deleteThroughProvider(t, testedBucket, fileName))
	})

	t.Run("removes the parts and the manifest of a published chunked file", func(t *testing.T) {
		testedBucket, client, bucketName := startS3CleanupTarget(t)
		fileName := "chunked-" + uuid.New().String()

		payload := strings.Repeat("x", 6*1024*1024)
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), strings.NewReader(payload)),
		)

		require.NoError(t, deleteThroughProvider(t, testedBucket, fileName))

		assert.False(t, objectExists(t, client, bucketName, fileName))
		assert.False(t, objectExists(t, client, bucketName, fileName+".parts"))
		assert.False(t, objectExists(t, client, bucketName, fileName+".part000001"))
	})
}

func TestSaveFile(t *testing.T) {
	t.Run("writes one object without a manifest when the stream fits one part", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		assert.ElementsMatch(
			t,
			[]string{fileName},
			listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName),
		)

		reassembledBytes := readWholeFile(t, testedBucket, fileName)
		assert.Equal(t, persistedBytes, reassembledBytes)
	})

	t.Run("writes one object without a manifest when the stream is exactly one part", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		persistedBytes := generateBytes(testInnerPartSize)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		assert.ElementsMatch(
			t,
			[]string{fileName},
			listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName),
		)

		reassembledBytes := readWholeFile(t, testedBucket, fileName)
		assert.Equal(t, persistedBytes, reassembledBytes)
	})

	t.Run("splits the file into objects and reads it back when the stream exceeds one object", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)

		// 23 MiB with 5 MiB parts and 2 parts/object: object1 = parts 1-2, object2 = parts 3-4,
		// object3 = the trailing 3 MiB part, exercising multi-object rollover and a sub-part final chunk.
		persistedBytes := generateBytes(23 * 1024 * 1024)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		assert.ElementsMatch(t, []string{
			fileName + ".part000001",
			fileName + ".part000002",
			fileName + ".part000003",
			fileName + manifestSuffix,
		}, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName))

		reassembledBytes := readWholeFile(t, testedBucket, fileName)
		assert.Equal(t, persistedBytes, reassembledBytes)
	})

	t.Run("leaves no part behind when the upload is cancelled midway", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)

		ctx, cancel := context.WithCancel(t.Context())

		// 5 MiB parts, 2 parts/object => 10 MiB objects. Deliver 30 MiB but trip cancellation after
		// ~12 MiB: object 1 (parts 1-2) is committed, then cancellation lands while object 2 is in
		// flight. Object 1 is the orphan that cleanup must remove even though ctx is already cancelled.
		source := &cancelAfterReader{
			data:     generateBytes(30 * 1024 * 1024),
			cancelAt: 12 * 1024 * 1024,
			cancel:   cancel,
		}

		fileName := uuid.NewString()
		err := testedBucket.provider.SaveFile(ctx, testedBucket.access(fileName), source)
		require.Error(t, err, "a cancelled upload must fail")

		assert.Empty(
			t,
			listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName),
			"cancellation mid-upload must leave no orphan part objects behind",
		)
	})

	t.Run("writes, reads and deletes a chunked file under the prefix", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)
		testedBucket.settings.Prefix = "team-a/nested"

		persistedBytes := generateBytes(23 * 1024 * 1024)
		fileName := uuid.NewString()

		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		// Parts and manifest land under the prefix, and the manifest must reference the prefixed keys so
		// read and delete reconstruct them correctly.
		assert.ElementsMatch(t, []string{
			"team-a/nested/" + fileName + ".part000001",
			"team-a/nested/" + fileName + ".part000002",
			"team-a/nested/" + fileName + ".part000003",
			"team-a/nested/" + fileName + manifestSuffix,
		}, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, "team-a/"))

		reassembledBytes := readWholeFile(t, testedBucket, fileName)
		assert.Equal(t, persistedBytes, reassembledBytes)

		require.NoError(t, testedBucket.provider.DeleteFile(t.Context(), testedBucket.access(fileName)))
		assert.Empty(t, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, "team-a/"))
	})

	t.Run("writes one empty object without a manifest when the stream is empty", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)

		fileName := uuid.NewString()
		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(nil)),
		)

		assert.ElementsMatch(
			t,
			[]string{fileName},
			listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName),
		)

		assert.Empty(t, readWholeFile(t, testedBucket, fileName))
	})

	t.Run("writes no trailing empty object when the stream fills the objects exactly", func(t *testing.T) {
		rawClient, testedBucket := setupChunkedBucket(t)

		// 20 MiB is exactly two 10 MiB objects (5 MiB part x 2). The boundary must not commit an empty
		// trailing object: the bucket holds two parts plus the manifest, not three.
		persistedBytes := generateBytes(20 * 1024 * 1024)
		fileName := uuid.NewString()

		require.NoError(
			t,
			testedBucket.provider.SaveFile(t.Context(), testedBucket.access(fileName), bytes.NewReader(persistedBytes)),
		)

		assert.ElementsMatch(t, []string{
			fileName + ".part000001",
			fileName + ".part000002",
			fileName + manifestSuffix,
		}, listObjectKeys(t, rawClient, testedBucket.settings.Bucket, fileName))

		reassembledBytes := readWholeFile(t, testedBucket, fileName)
		assert.Equal(t, persistedBytes, reassembledBytes)
	})
}

func TestTestConnection(t *testing.T) {
	t.Parallel()

	t.Run("succeeds when the delete of the probe is denied", func(t *testing.T) {
		t.Parallel()

		testedBucket := newStubBucket(t, startS3ProbeServer(t, s3ProbeStub{deleteErrorCode: s3ErrCodeAccessDenied}))

		assert.NoError(t, testedBucket.provider.TestConnection(t.Context(), testedBucket.location()))
	})

	t.Run("names the missing bucket when the bucket does not exist", func(t *testing.T) {
		t.Parallel()

		testedBucket := newStubBucket(t, startS3ProbeServer(t, s3ProbeStub{putErrorCode: s3ErrCodeNoSuchBucket}))

		err := testedBucket.provider.TestConnection(t.Context(), testedBucket.location())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("reports the denied write when the credentials cannot write", func(t *testing.T) {
		t.Parallel()

		testedBucket := newStubBucket(t, startS3ProbeServer(t, s3ProbeStub{putErrorCode: s3ErrCodeAccessDenied}))

		err := testedBucket.provider.TestConnection(t.Context(), testedBucket.location())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot write to")
	})

	t.Run("reports the denied read when the credentials cannot read back", func(t *testing.T) {
		t.Parallel()

		testedBucket := newStubBucket(t, startS3ProbeServer(t, s3ProbeStub{getErrorCode: s3ErrCodeAccessDenied}))

		err := testedBucket.provider.TestConnection(t.Context(), testedBucket.location())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot read back from")
	})

	t.Run("reports the mismatch when the read returns different content", func(t *testing.T) {
		t.Parallel()

		testedBucket := newStubBucket(t, startS3ProbeServer(t, s3ProbeStub{servedContent: "corrupted"}))

		err := testedBucket.provider.TestConnection(t.Context(), testedBucket.location())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "different content")
	})
}

func setupChunkedBucket(t *testing.T) (*minio.Client, *testBucket) {
	t.Helper()

	endpoint := containers.StartMinio(t)
	address := providertest.GetAddress(endpoint)
	rawClient := newRawClient(t, address)
	createTestBucket(t, rawClient)

	testedBucket := newTestBucket(t, Settings{
		Bucket:   testBucketName,
		Region:   containers.MinioRegion,
		Endpoint: "http://" + address,
	})
	testedBucket.provider.partLayout = partLayout{innerPartSize: testInnerPartSize, maxPartsPerObject: 2}

	return rawClient, testedBucket
}

func newStubBucket(t *testing.T, server *httptest.Server) *testBucket {
	t.Helper()

	return newTestBucket(t, Settings{Bucket: testBucketName, Region: containers.MinioRegion, Endpoint: server.URL})
}

// testBucket is a provider with the settings of one storage, holding encrypted keys as a repository
// returns them.
type testBucket struct {
	provider  *Provider
	settings  Settings
	storageID uuid.UUID
}

func newTestBucket(t *testing.T, settings Settings) *testBucket {
	t.Helper()

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	settings.AccessKey = providertest.Encrypt(t, fieldEncryptor, containers.MinioRootUser)
	settings.SecretKey = providertest.Encrypt(t, fieldEncryptor, containers.MinioRootPassword)

	return &testBucket{
		provider:  New(Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: fieldEncryptor}),
		settings:  settings,
		storageID: uuid.New(),
	}
}

func (b *testBucket) location() providers.Location {
	return providers.Location{StorageID: b.storageID, Settings: b.settings}
}

func (b *testBucket) access(fileName string) providers.FileAccess {
	return providers.FileAccess{Location: b.location(), FileName: fileName, Logger: providertest.NewDiscardLogger()}
}

func newRawClient(t *testing.T, address string) *minio.Client {
	t.Helper()

	rawClient, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Secure: false,
		Region: containers.MinioRegion,
	})
	require.NoError(t, err, "failed to create raw minio client")

	return rawClient
}

func createTestBucket(t *testing.T, rawClient *minio.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	require.NoError(
		t,
		rawClient.MakeBucket(ctx, testBucketName, minio.MakeBucketOptions{Region: containers.MinioRegion}),
	)
}

// A real MinIO container cannot express these branches: its root credentials ignore bucket
// policies, so a denied write, read or delete is not reproducible against it.
type s3ProbeStub struct {
	putErrorCode    string
	getErrorCode    string
	deleteErrorCode string
	servedContent   string
}

func startS3ProbeServer(t *testing.T, stub s3ProbeStub) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if stub.putErrorCode != "" {
				writeS3ErrorResponse(w, stub.putErrorCode)
				return
			}

			w.Header().Set("ETag", `"probe"`)
		case http.MethodGet:
			if stub.getErrorCode != "" {
				writeS3ErrorResponse(w, stub.getErrorCode)
				return
			}

			content := stub.servedContent
			if content == "" {
				content = connectionTestObjectContent
			}

			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
			_, _ = w.Write([]byte(content))
		case http.MethodDelete:
			if stub.deleteErrorCode != "" {
				writeS3ErrorResponse(w, stub.deleteErrorCode)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

// Serves an empty bucket whose batch delete fails every requested key with objectErrorCode, which
// MinIO cannot reproduce: it reports a missing key as deleted.
func startS3BatchDeleteServer(t *testing.T, objectErrorCode string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		switch {
		case r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodGet && query.Has("uploads"):
			writeXMLResponse(
				w,
				http.StatusOK,
				`<ListMultipartUploadsResult><IsTruncated>false</IsTruncated></ListMultipartUploadsResult>`,
			)
		case r.Method == http.MethodGet:
			writeXMLResponse(w, http.StatusOK, `<ListBucketResult><IsTruncated>false</IsTruncated></ListBucketResult>`)
		case r.Method == http.MethodPost && query.Has("delete"):
			var request struct {
				Objects []struct {
					Key string `xml:"Key"`
				} `xml:"Object"`
			}

			if err := xml.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var result strings.Builder
			result.WriteString("<DeleteResult>")

			for _, object := range request.Objects {
				fmt.Fprintf(&result, "<Error><Key>%s</Key><Code>%s</Code><Message>%s</Message></Error>",
					object.Key, objectErrorCode, objectErrorCode)
			}

			result.WriteString("</DeleteResult>")
			writeXMLResponse(w, http.StatusOK, result.String())
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func writeXMLResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + body))
}

func writeS3ErrorResponse(w http.ResponseWriter, code string) {
	statusByCode := map[string]int{
		s3ErrCodeNoSuchBucket: http.StatusNotFound,
		s3ErrCodeAccessDenied: http.StatusForbidden,
	}

	status, isKnownCode := statusByCode[code]
	if !isKnownCode {
		status = http.StatusForbidden
	}

	writeXMLResponse(w, status, fmt.Sprintf("<Error><Code>%s</Code><Message>%s</Message></Error>", code, code))
}

func putRawObject(t *testing.T, client *minio.Client, bucketName, key string, data []byte) {
	t.Helper()

	_, err := client.PutObject(
		t.Context(),
		bucketName,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{},
	)
	require.NoError(t, err, "failed to write raw object %s", key)
}

func readWholeFile(t *testing.T, testedBucket *testBucket, fileName string) []byte {
	t.Helper()

	reader, err := testedBucket.provider.OpenFile(t.Context(), testedBucket.access(fileName))
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)

	return content
}

func listObjectKeys(t *testing.T, client *minio.Client, bucketName, prefix string) []string {
	t.Helper()

	var keys []string
	for object := range client.ListObjects(t.Context(), bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		require.NoError(t, object.Err)
		keys = append(keys, object.Key)
	}

	return keys
}

// generateBytes returns deterministic, position-dependent bytes so a round trip can be compared
// exactly without holding a second random source.
func generateBytes(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i*7 + 13)
	}

	return data
}

// cancelAfterReader serves data and cancels its context once cancelAt bytes have been read, to
// simulate a backup cancelled partway through a chunked upload.
type cancelAfterReader struct {
	data     []byte
	pos      int
	cancelAt int
	cancel   context.CancelFunc
}

func (r *cancelAfterReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.pos:])
	r.pos += n

	if r.cancel != nil && r.pos >= r.cancelAt {
		r.cancel()
		r.cancel = nil
	}

	return n, nil
}

func TestTransport(t *testing.T) {
	t.Parallel()

	t.Run("takes the proxy from the environment", func(t *testing.T) {
		t.Parallel()

		transport := buildTestS3Transport(t, false)

		require.NotNil(t, transport.Proxy)
		assert.Equal(t,
			reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
			reflect.ValueOf(transport.Proxy).Pointer(),
		)
	})

	t.Run("follows the setting that skips the TLS check", func(t *testing.T) {
		t.Parallel()

		assert.False(t, buildTestS3Transport(t, false).TLSClientConfig.InsecureSkipVerify)
		assert.True(t, buildTestS3Transport(t, true).TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("tunnels through the proxy and rejects an untrusted certificate when the check is on", func(t *testing.T) {
		t.Parallel()

		exchange := test_utils.SendThroughTunnelProxy(t, buildTestS3Transport(t, false), "AWS4-HMAC-SHA256 signature")

		test_utils.AssertProxyReceivedOnlyTunnelRequest(t, exchange)
		test_utils.AssertCertificateRejected(t, exchange)
	})

	t.Run("sends the credentials only to the endpoint when the check is skipped", func(t *testing.T) {
		t.Parallel()

		exchange := test_utils.SendThroughTunnelProxy(t, buildTestS3Transport(t, true), "AWS4-HMAC-SHA256 signature")

		require.NoError(t, exchange.Err)
		test_utils.AssertProxyReceivedOnlyTunnelRequest(t, exchange)
		require.Len(t, exchange.EndpointRequests, 1)
		assert.Equal(t, "AWS4-HMAC-SHA256 signature", exchange.EndpointRequests[0].Header.Get("Authorization"))
	})
}

func buildTestS3Transport(t *testing.T, isSkipTLSVerify bool) *http.Transport {
	t.Helper()

	testedBucket := newTestBucket(t, Settings{
		Bucket:          testBucketName,
		Region:          "us-east-1",
		Endpoint:        "https://s3.example.com",
		IsSkipTLSVerify: isSkipTLSVerify,
	})

	params, err := (&bucket{provider: testedBucket.provider, settings: testedBucket.settings}).getClientParams()
	require.NoError(t, err)

	return params.transport
}

func startS3CleanupTarget(t *testing.T) (*testBucket, *minio.Client, string) {
	t.Helper()

	address := providertest.GetAddress(containers.StartMinio(t))
	rawClient := newRawClient(t, address)
	createTestBucket(t, rawClient)

	testedBucket := newTestBucket(t, Settings{
		Bucket:   testBucketName,
		Region:   containers.MinioRegion,
		Endpoint: "http://" + address,
	})

	return testedBucket, rawClient, testBucketName
}

func objectExists(t *testing.T, client *minio.Client, bucketName, key string) bool {
	t.Helper()

	_, err := client.StatObject(t.Context(), bucketName, key, minio.StatObjectOptions{})

	return err == nil
}

func putObject(t *testing.T, client *minio.Client, bucketName, key, body string) {
	t.Helper()

	_, err := client.PutObject(
		t.Context(), bucketName, key, strings.NewReader(body), int64(len(body)), minio.PutObjectOptions{},
	)
	require.NoError(t, err)
}

func deleteThroughProvider(t *testing.T, testedBucket *testBucket, fileName string) error {
	t.Helper()

	return testedBucket.provider.DeleteFile(t.Context(), testedBucket.access(fileName))
}

func hasIncompleteUpload(t *testing.T, client *minio.Client, bucketName, key string) bool {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	for upload := range client.ListIncompleteUploads(ctx, bucketName, key, true) {
		require.NoError(t, upload.Err)

		if upload.Key == key {
			return true
		}
	}

	return false
}
