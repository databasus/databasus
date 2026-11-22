package providers_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"databasus-backend/internal/lib/testreport"
)

const (
	objectPrefixMarker = "databasus_e2e_"
	// abandonedPrefixAge leaves a younger marked prefix alone, because another run may still use it.
	abandonedPrefixAge   = time.Hour
	objectCleanupTimeout = 2 * time.Minute
)

// objectPrefixPattern reads the creation time from a run's key prefix: the marker, the Unix time and
// a random part unique to the run.
var objectPrefixPattern = regexp.MustCompile(`^databasus_e2e_(\d+)_[0-9a-f]+/`)

// objectStorageRun owns one key prefix of a shared bucket. Concurrent runs never touch each other's
// prefixes, and a later run can sweep what a killed run left behind.
type objectStorageRun struct {
	target objectStorageTarget
	prefix string
}

// startObjectStorageRun registers the cleanup before the run writes anything, so the cleanup also
// runs after a failed test.
func startObjectStorageRun(
	reporter testreport.Reporter,
	target objectStorageTarget,
	startedAt time.Time,
) *objectStorageRun {
	reporter.Helper()

	randomPart := make([]byte, 6)
	_, _ = rand.Read(randomPart)

	run := &objectStorageRun{
		target: target,
		prefix: fmt.Sprintf("%s%d_%s/", objectPrefixMarker, startedAt.Unix(), hex.EncodeToString(randomPart)),
	}

	reporter.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(reporter.Context()), objectCleanupTimeout)
		defer cancel()

		for _, failure := range removeEverythingUnder(ctx, target, run.prefix) {
			reporter.Errorf("cleanup of run %s: %s", run.prefix, failure)
		}
	})

	return run
}

// reportLeftovers fails naming every object and incomplete upload left under the run's prefix after
// stage.
func (r *objectStorageRun) reportLeftovers(reporter testreport.Reporter, stage string) {
	reporter.Helper()

	leftovers, err := findLeftovers(reporter.Context(), r.target, r.prefix)
	if err != nil {
		reporter.Errorf("could not list what %s left in %s: %s",
			stage, r.target.describe(), r.target.redactor.redact(err.Error()))

		return
	}

	for _, leftover := range leftovers {
		reporter.Errorf("after %s, %s remains in %s", stage, leftover, r.target.describe())
	}
}

// sweepAbandonedObjectStoragePrefixes removes the marked prefixes older than abandonedPrefixAge,
// which a run killed before its cleanup left behind. It never touches a key outside a marked prefix.
// A failed removal is only logged, because another run's sweep may be removing the same object.
func sweepAbandonedObjectStoragePrefixes(reporter testreport.Reporter, target objectStorageTarget, now time.Time) {
	reporter.Helper()

	ctx := reporter.Context()
	abandonedPrefixes := map[string]struct{}{}

	for object := range target.rawClient.ListObjects(ctx, target.settings.Bucket, minio.ListObjectsOptions{
		Prefix:    objectPrefixMarker,
		Recursive: true,
	}) {
		if object.Err != nil {
			reporter.Logf("sweep could not list %s: %s", target.describe(), target.redactor.redact(object.Err.Error()))

			return
		}

		if prefix, isAbandoned := getAbandonedPrefix(object.Key, now); isAbandoned {
			abandonedPrefixes[prefix] = struct{}{}
		}
	}

	uploads, err := listIncompleteUploadsUnder(ctx, target, objectPrefixMarker)
	if err != nil {
		reporter.Logf("sweep could not list uploads of %s: %s", target.describe(), target.redactor.redact(err.Error()))

		return
	}

	for _, upload := range uploads {
		if prefix, isAbandoned := getAbandonedPrefix(upload.Key, now); isAbandoned {
			abandonedPrefixes[prefix] = struct{}{}
		}
	}

	for prefix := range abandonedPrefixes {
		for _, failure := range removeEverythingUnder(ctx, target, prefix) {
			reporter.Logf("sweep: %s", failure)
		}
	}
}

func getAbandonedPrefix(key string, now time.Time) (string, bool) {
	match := objectPrefixPattern.FindStringSubmatch(key)
	if match == nil {
		return "", false
	}

	createdAtUnix, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return "", false
	}

	return match[0], now.Sub(time.Unix(createdAtUnix, 0)) > abandonedPrefixAge
}

// removeEverythingUnder removes every object and aborts every incomplete upload under prefix, and
// names each one it could not remove.
func removeEverythingUnder(ctx context.Context, target objectStorageTarget, prefix string) []string {
	bucket := target.settings.Bucket

	var failures []string

	objectsToRemove := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsToRemove)

		for object := range target.rawClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}) {
			if object.Err != nil {
				failures = append(failures, fmt.Sprintf("could not list objects under %s in %s: %s",
					prefix, target.describe(), target.redactor.redact(object.Err.Error())))

				return
			}

			objectsToRemove <- object
		}
	}()

	var removalFailures []string

	for removal := range target.rawClient.RemoveObjects(ctx, bucket, objectsToRemove, minio.RemoveObjectsOptions{}) {
		if removal.Err != nil {
			removalFailures = append(removalFailures, fmt.Sprintf("could not remove object %s from %s: %s",
				removal.ObjectName, target.describe(), target.redactor.redact(removal.Err.Error())))
		}
	}

	failures = append(failures, removalFailures...)

	coreClient := &minio.Core{Client: target.rawClient}

	uploads, err := listIncompleteUploadsUnder(ctx, target, prefix)
	if err != nil {
		failures = append(failures, fmt.Sprintf("could not list uploads under %s in %s: %s",
			prefix, target.describe(), target.redactor.redact(err.Error())))
	}

	for _, upload := range uploads {
		if err := coreClient.AbortMultipartUpload(ctx, bucket, upload.Key, upload.UploadID); err != nil {
			failures = append(failures, fmt.Sprintf("could not abort upload %s of %s in %s: %s",
				upload.UploadID, upload.Key, target.describe(), target.redactor.redact(err.Error())))
		}
	}

	return failures
}

func findLeftovers(ctx context.Context, target objectStorageTarget, prefix string) ([]string, error) {
	var leftovers []string

	for object := range target.rawClient.ListObjects(ctx, target.settings.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}

		leftovers = append(leftovers, "object "+object.Key)
	}

	uploads, err := listIncompleteUploadsUnder(ctx, target, prefix)
	if err != nil {
		return nil, err
	}

	for _, upload := range uploads {
		leftovers = append(leftovers, fmt.Sprintf("incomplete upload %s of %s", upload.UploadID, upload.Key))
	}

	return leftovers, nil
}

// listIncompleteUploadsUnder lists every incomplete upload of the bucket and keeps those under prefix,
// because MinIO answers a listing by prefix only for an exact object key.
func listIncompleteUploadsUnder(
	ctx context.Context,
	target objectStorageTarget,
	prefix string,
) ([]minio.ObjectMultipartInfo, error) {
	var uploads []minio.ObjectMultipartInfo

	for upload := range target.rawClient.ListIncompleteUploads(ctx, target.settings.Bucket, "", true) {
		if upload.Err != nil {
			return nil, upload.Err
		}

		if strings.HasPrefix(upload.Key, prefix) {
			uploads = append(uploads, upload)
		}
	}

	return uploads, nil
}
