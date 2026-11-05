package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
)

// chunkedManifest lists, in order, the objects a single logical file was split into, with each
// object's size and sha256. It is written LAST, after every part object is durably committed.
//
// Why a manifest instead of just probing {fileName}.part1, .part2, ... until a key 404s:
//
//  1. The bytes we store are an encrypted stream with no end-of-stream marker.
//     Probe-until-404 cannot tell a complete backup from one whose final part
//     failed to upload, was truncated, or was reaped by lifecycle rules: it would stop at the gap
//     and hand the restore a silently short, corrupt stream. TotalSize plus the per-part list and
//     sha256 let GetFile reject a missing/short/corrupted part BEFORE the restore starts, turning
//     silent data loss into a hard error.
//  2. Because the manifest is written last, its presence is the atomic "backup is complete" marker.
//     A crash mid-upload leaves part objects but no manifest, which GetFile treats as a hard error
//     (it finds neither manifest nor single object) rather than a valid N-part backup.
//  3. It immunizes reads against S3 eventual consistency: a transient 404 on a middle part can no
//     longer be misread as the end of the stream.
type chunkedManifest struct {
	Version   int          `json:"version"`
	TotalSize int64        `json:"totalSize"`
	Parts     []objectSpan `json:"parts"`
}

// objectSpan is one stored object with the length and sha256 a read must verify it against. The
// manifest lists them for a chunked file; a file stored before chunking is described by a single
// span with no recorded checksum.
type objectSpan struct {
	Key    string `json:"key"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func partObjectKey(baseKey string, objectIndex int) string {
	return fmt.Sprintf("%s%s%0*d", baseKey, partSuffix, partIndexDigits, objectIndex)
}

func isPartObjectKey(baseKey, candidate string) bool {
	index, found := strings.CutPrefix(candidate, baseKey+partSuffix)
	if !found || len(index) != partIndexDigits {
		return false
	}

	for _, digit := range index {
		if digit < '0' || digit > '9' {
			return false
		}
	}

	return true
}

func manifestObjectKey(baseKey string) string {
	return baseKey + manifestSuffix
}

func (b *bucket) writeManifest(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
	manifest chunkedManifest,
) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk manifest: %w", err)
	}

	opts := b.putObjectOptions()
	opts.SendContentMd5 = true
	opts.ContentType = "application/json"

	_, err = client.PutObject(
		ctx,
		b.settings.Bucket,
		manifestObjectKey(baseKey),
		bytes.NewReader(body),
		int64(len(body)),
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to write chunk manifest: %w", err)
	}

	return nil
}

func (b *bucket) readManifest(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) (chunkedManifest, bool, error) {
	var manifest chunkedManifest

	manifestKey := manifestObjectKey(baseKey)

	_, err := client.StatObject(ctx, b.settings.Bucket, manifestKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return manifest, false, nil
		}

		return manifest, false, fmt.Errorf("failed to stat chunk manifest: %w", err)
	}

	object, err := client.GetObject(ctx, b.settings.Bucket, manifestKey, minio.GetObjectOptions{})
	if err != nil {
		return manifest, false, fmt.Errorf("failed to read chunk manifest: %w", err)
	}
	defer func() { _ = object.Close() }()

	body, err := io.ReadAll(object)
	if err != nil {
		return manifest, false, fmt.Errorf("failed to read chunk manifest: %w", err)
	}

	if err := json.Unmarshal(body, &manifest); err != nil {
		return manifest, false, fmt.Errorf("failed to parse chunk manifest: %w", err)
	}

	return manifest, true, nil
}
