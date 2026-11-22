package providers_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testreport"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

// TestNeonObjectStorage checks the S3 storage against a real managed object storage service, which
// MinIO cannot stand in for. It needs the TEST_NEON_S3_* settings of a dedicated bucket.
func TestNeonObjectStorage(t *testing.T) {
	t.Parallel()

	t.Run("saves, reads and deletes files with nothing left behind on Neon", func(t *testing.T) {
		t.Parallel()
		settings := requireNeonObjectStorageSettings(t)
		target := newNeonObjectStorageTarget(t, settings)
		sweepAbandonedObjectStoragePrefixes(t, target, time.Now())

		run := startObjectStorageRun(t, target, time.Now())

		runObjectStorageFileCycle(t, run)
	})
}

func TestNeonObjectStorageCycleOnContainer(t *testing.T) {
	t.Parallel()

	t.Run("saves, reads and deletes files with nothing left behind on MinIO", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		run := startObjectStorageRun(t, target, time.Now())

		runObjectStorageFileCycle(t, run)
	})

	t.Run("passes both runs and leaves nothing when two runs share a bucket", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		recorders := []*testreport.Recorder{testreport.NewRecorder(t), testreport.NewRecorder(t)}
		runs := make([]*objectStorageRun, len(recorders))

		var waitGroup sync.WaitGroup

		for runIndex, recorder := range recorders {
			waitGroup.Go(func() {
				recorder.RunWithCleanups(func(reporter testreport.Reporter) {
					runs[runIndex] = startObjectStorageRun(reporter, target, time.Now())
					runObjectStorageFileCycle(reporter, runs[runIndex])
				})
			})
		}

		waitGroup.Wait()

		require.NotEqual(t, runs[0].prefix, runs[1].prefix)

		for _, recorder := range recorders {
			assert.Empty(t, recorder.GetFailures())
		}

		for _, run := range runs {
			leftovers, err := findLeftovers(t.Context(), target, run.prefix)
			require.NoError(t, err)
			assert.Empty(t, leftovers)
		}
	})

	t.Run("fails naming the file and digest when content reads back different", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		run := startObjectStorageRun(t, target, time.Now())
		savedContent := generateObjectStorageContent(singleObjectFileSize)
		saveObjectStorageFile(t, run, smallFileName, savedContent)
		putRawObject(t, target, run.prefix+smallFileName, []byte("different content"))
		recorder := testreport.NewRecorder(t)

		recorder.RunWithCleanups(func(reporter testreport.Reporter) {
			verifyReadBack(reporter, run, smallFileName, savedContent)
		})

		failures := recorder.GetFailures()
		require.Len(t, failures, 1)
		assert.Contains(t, failures[0], smallFileName)
		assert.Contains(t, failures[0], "SHA-256")
	})

	t.Run("fails naming each object and upload left under the prefix", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		run := startObjectStorageRun(t, target, time.Now())
		putRawObject(t, target, run.prefix+"stray-object", []byte("stray"))
		strayUploadID := startRawMultipartUpload(t, target, run.prefix+"stray-upload")
		recorder := testreport.NewRecorder(t)

		recorder.RunWithCleanups(func(reporter testreport.Reporter) {
			run.reportLeftovers(reporter, "deleting the files")
		})

		failures := strings.Join(recorder.GetFailures(), "\n")
		assert.Contains(t, failures, run.prefix+"stray-object")
		assert.Contains(t, failures, strayUploadID)
		assert.Contains(t, failures, run.prefix+"stray-upload")
	})

	t.Run("fails naming the endpoint and bucket without either key when the secret key is refused", func(t *testing.T) {
		t.Parallel()
		const refusedSecretKey = "refused-secret-key"
		target := startMinioObjectStorageTarget(t, refusedSecretKey)
		recorder := testreport.NewRecorder(t)

		recorder.RunWithCleanups(func(reporter testreport.Reporter) {
			run := startObjectStorageRun(reporter, target, time.Now())
			runObjectStorageFileCycle(reporter, run)
		})

		failures := strings.Join(recorder.GetFailures(), "\n")
		require.NotEmpty(t, failures)
		assert.Contains(t, failures, target.settings.Endpoint)
		assert.Contains(t, failures, target.settings.Bucket)
		assert.NotContains(t, failures, refusedSecretKey)
		assert.NotContains(t, failures, containers.MinioRootUser)

		logLines := strings.Join(recorder.GetLogLines(), "\n")
		assert.NotContains(t, logLines, refusedSecretKey)
		assert.NotContains(t, logLines, containers.MinioRootUser)
	})

	t.Run(
		"fails naming the storage error when the uploaded parts cannot be listed before the cancel",
		func(t *testing.T) {
			t.Parallel()
			target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
			refusingTarget := target.withRawClientThrough(t, startUploadListingRefusingProxy(t, target))
			run := startObjectStorageRun(t, target, time.Now())
			refusingRun := &objectStorageRun{target: refusingTarget, prefix: run.prefix}
			recorder := testreport.NewRecorder(t)

			recorder.RunWithCleanups(func(reporter testreport.Reporter) {
				cancelObjectStorageSave(reporter, refusingRun)
			})

			failures := strings.Join(recorder.GetFailures(), "\n")
			assert.Contains(t, failures, "could not list the uploaded parts")
			assert.Contains(t, failures, refusingTarget.describe())
			assert.Contains(t, failures, "Access Denied")
			assert.NotContains(t, failures, "abort went unchecked")
		},
	)

	t.Run("fails naming the key when cleanup cannot remove it", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		refusingTarget := target.withRawClientThrough(t, startRemovalRefusingProxy(t, target))
		recorder := testreport.NewRecorder(t)

		var run *objectStorageRun

		recorder.RunWithCleanups(func(reporter testreport.Reporter) {
			run = startObjectStorageRun(reporter, refusingTarget, time.Now())
			putRawObject(reporter, target, run.prefix+"kept-object", []byte("kept"))
		})

		t.Cleanup(func() { removeEverythingUnder(context.WithoutCancel(t.Context()), target, run.prefix) })

		failures := strings.Join(recorder.GetFailures(), "\n")
		assert.Contains(t, failures, run.prefix+"kept-object")
	})
}

func TestSweepAbandonedNeonObjectStoragePrefixes(t *testing.T) {
	t.Parallel()

	t.Run("removes prefixes older than an hour and keeps younger and unmarked keys", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		now := time.Now()
		abandonedPrefix := getMarkedPrefix(now.Add(-2*time.Hour), "aa11")
		youngPrefix := getMarkedPrefix(now.Add(-5*time.Minute), "bb22")
		putRawObject(t, target, abandonedPrefix+"file", []byte("abandoned"))
		startRawMultipartUpload(t, target, abandonedPrefix+"upload")
		putRawObject(t, target, youngPrefix+"file", []byte("young"))
		putRawObject(t, target, "unmarked/file", []byte("unmarked"))

		sweepAbandonedObjectStoragePrefixes(t, target, now)

		abandonedLeftovers, err := findLeftovers(t.Context(), target, abandonedPrefix)
		require.NoError(t, err)
		assert.Empty(t, abandonedLeftovers)

		youngLeftovers, err := findLeftovers(t.Context(), target, youngPrefix)
		require.NoError(t, err)
		assert.Equal(t, []string{"object " + youngPrefix + "file"}, youngLeftovers)

		unmarkedLeftovers, err := findLeftovers(t.Context(), target, "unmarked/")
		require.NoError(t, err)
		assert.Equal(t, []string{"object unmarked/file"}, unmarkedLeftovers)
	})

	t.Run("logs a failed removal and lets the run continue", func(t *testing.T) {
		t.Parallel()
		target := startMinioObjectStorageTarget(t, containers.MinioRootPassword)
		refusingTarget := target.withRawClientThrough(t, startRemovalRefusingProxy(t, target))
		abandonedPrefix := getMarkedPrefix(time.Now().Add(-2*time.Hour), "cc33")
		putRawObject(t, target, abandonedPrefix+"file", []byte("abandoned"))
		recorder := testreport.NewRecorder(t)
		isRunContinued := false

		recorder.RunWithCleanups(func(reporter testreport.Reporter) {
			sweepAbandonedObjectStoragePrefixes(reporter, refusingTarget, time.Now())
			isRunContinued = true
		})

		assert.True(t, isRunContinued)
		assert.Empty(t, recorder.GetFailures())
		assert.Contains(t, strings.Join(recorder.GetLogLines(), "\n"), abandonedPrefix+"file")
	})
}

func getMarkedPrefix(createdAt time.Time, randomPart string) string {
	return fmt.Sprintf("%s%d_%s/", objectPrefixMarker, createdAt.Unix(), randomPart)
}

// putRawObject takes a reporter, so a body that a recorder runs on a goroutine of its own reports a
// failed write through the recorder instead of failing the test off its goroutine.
func putRawObject(reporter testreport.Reporter, target objectStorageTarget, key string, content []byte) {
	reporter.Helper()

	_, err := target.rawClient.PutObject(reporter.Context(), target.settings.Bucket, key,
		bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{})
	if err != nil {
		reporter.Fatalf("could not put %s into %s: %s", key, target.describe(), target.redactor.redact(err.Error()))
	}
}

func startRawMultipartUpload(t *testing.T, target objectStorageTarget, key string) string {
	t.Helper()

	coreClient := &minio.Core{Client: target.rawClient}

	uploadID, err := coreClient.NewMultipartUpload(t.Context(), target.settings.Bucket, key, minio.PutObjectOptions{})
	require.NoError(t, err)

	return uploadID
}

// startRemovalRefusingProxy answers every batch delete with AccessDenied, which a storage returns
// for a credential that may not delete, and passes every other request to the target.
func startRemovalRefusingProxy(t *testing.T, target objectStorageTarget) string {
	t.Helper()

	return startAccessDeniedProxy(t, target, func(r *http.Request) bool {
		return r.Method == http.MethodPost && r.URL.Query().Has("delete")
	})
}

// startUploadListingRefusingProxy answers every listing of multipart uploads and of their parts
// with AccessDenied, and passes every other request to the target.
func startUploadListingRefusingProxy(t *testing.T, target objectStorageTarget) string {
	t.Helper()

	return startAccessDeniedProxy(t, target, func(r *http.Request) bool {
		query := r.URL.Query()

		return r.Method == http.MethodGet && (query.Has("uploads") || query.Has("uploadId"))
	})
}

func startAccessDeniedProxy(t *testing.T, target objectStorageTarget, isRefusable func(r *http.Request) bool) string {
	t.Helper()

	proxy := providertest.StartRefusingProxy(t, providertest.RefusingProxySpec{
		TargetURL:    target.settings.Endpoint,
		IsRefusable:  isRefusable,
		RefusalCount: 1_000_000,
		Status:       http.StatusForbidden,
		ContentType:  "application/xml",
		Body: `<?xml version="1.0" encoding="UTF-8"?>` +
			`<Error><Code>AccessDenied</Code><Message>Access Denied.</Message></Error>`,
	})

	return strings.TrimPrefix(proxy.URL, "http://")
}
