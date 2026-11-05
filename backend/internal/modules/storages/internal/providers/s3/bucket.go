package s3

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// bucket is one S3 storage while a call runs: the settings it was called with and the provider
// that keeps the connections.
type bucket struct {
	provider *Provider
	settings Settings
}

const (
	s3ConnectTimeout      = 30 * time.Second
	s3ResponseTimeout     = 30 * time.Second
	s3IdleConnTimeout     = 90 * time.Second
	s3TLSHandshakeTimeout = 30 * time.Second
	s3DeleteTimeout       = 30 * time.Second

	s3ConnectionTestTimeout = 10 * time.Second

	connectionTestObjectName    = "databasus-connection-test"
	connectionTestObjectContent = "databasus connection test"

	// defaultInnerPartSize is the multipart part we feed S3 for one object. 16 MiB balances
	// memory against request count and creates backpressure to pg_dump: we keep one part in
	// flight and wait for S3 to confirm it before reading the next, so pipeline memory does not
	// grow with the database size (docs/architecture.md, "Backup data flow").
	defaultInnerPartSize = 16 * 1024 * 1024

	// defaultMaxPartsPerObject caps inner parts per object; once reached we roll to the next
	// object. Effective object size = defaultMaxPartsPerObject * defaultInnerPartSize ~= 15.6 GiB.
	// S3 multipart is bounded by part_size * part_count_limit and that limit is provider-set
	// (AWS 10000, Scaleway and many S3-compatibles 1000). Because we stream dumps of unknown size,
	// we write blind to the smallest common limit so one backup can span any provider without
	// knowing its real cap; the object count, not memory, scales with the database size.
	defaultMaxPartsPerObject = 1000

	manifestSuffix  = ".parts"
	partSuffix      = ".part"
	partIndexDigits = 6
	manifestVersion = 1

	s3ErrCodeNoSuchBucket = "NoSuchBucket"
	s3ErrCodeAccessDenied = "AccessDenied"
	s3ErrCodeNoSuchKey    = "NoSuchKey"
)

func (b *bucket) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("upload cancelled before start: %w", ctx.Err())
	default:
	}

	client, err := b.getClient(ctx)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to build the s3 client",
			slog.Any("bucket", b.settings.Bucket),
			slog.Any("error", err),
		)

		return err
	}

	baseKey := b.buildObjectKey(fileName)
	innerPartSize := b.provider.partLayout.innerPartSize

	startedAt := time.Now().UTC()

	logger.DebugContext(ctx, "saving file to s3", slog.Any("file_name", fileName),
		slog.Any("bucket", b.settings.Bucket), slog.Any("object_key", baseKey))

	// Look ahead by one inner part to choose the layout. A stream that fits in a single inner part
	// (WAL segments, .metadata/.manifest sidecars, small dumps, empty files) is written as one
	// plain object {fileName} with no manifest, exactly as legacy backups are. Anything larger is
	// split across {fileName}.partNNNNNN objects described by a {fileName}.parts manifest.
	firstPart := make([]byte, innerPartSize)
	firstPartLen, readErr := io.ReadFull(file, firstPart)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return fmt.Errorf("read error: %w", readErr)
	}

	if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
		if err := b.uploadSingleObject(ctx, client, baseKey, firstPart[:firstPartLen]); err != nil {
			logger.ErrorContext(ctx, "failed to save file to s3", slog.Any("file_name", fileName),
				slog.Any("object_key", baseKey), slog.Any("error", err))

			return err
		}

		reportSaveToS3(
			ctx,
			logger,
			savedObject{FileName: fileName, ObjectKey: baseKey, SizeBytes: int64(firstPartLen), StartedAt: startedAt},
		)

		return nil
	}

	// The first part filled exactly: peek one more byte to tell "stream is exactly one inner part"
	// (still single object, e.g. a 16 MiB WAL segment) from "stream is larger" (chunked).
	peek := make([]byte, 1)
	peekLen, peekErr := io.ReadFull(file, peek)
	if peekErr != nil && peekErr != io.EOF {
		return fmt.Errorf("read error: %w", peekErr)
	}

	if peekLen == 0 {
		if err := b.uploadSingleObject(ctx, client, baseKey, firstPart); err != nil {
			logger.ErrorContext(ctx, "failed to save file to s3", slog.Any("file_name", fileName),
				slog.Any("object_key", baseKey), slog.Any("error", err))

			return err
		}

		reportSaveToS3(
			ctx,
			logger,
			savedObject{FileName: fileName, ObjectKey: baseKey, SizeBytes: int64(len(firstPart)), StartedAt: startedAt},
		)

		return nil
	}

	coreClient := &minio.Core{Client: client}

	source := io.MultiReader(bytes.NewReader(firstPart), bytes.NewReader(peek[:peekLen]), file)

	logger.DebugContext(ctx, "stream exceeds one part, saving to s3 as a chunked object",
		slog.Any("file_name", fileName), slog.Any("object_key", baseKey))

	if err := b.uploadChunked(ctx, coreClient, client, baseKey, source); err != nil {
		logger.ErrorContext(ctx, "failed to save chunked file to s3", slog.Any("file_name", fileName),
			slog.Any("object_key", baseKey), slog.Any("error", err))

		return err
	}

	logger.DebugContext(ctx, "saved chunked file to s3",
		slog.String("file_name", fileName),
		slog.String("object_key", baseKey),
		slog.Duration("duration", time.Since(startedAt)))

	return nil
}

type savedObject struct {
	FileName  string
	ObjectKey string
	SizeBytes int64
	StartedAt time.Time
}

func reportSaveToS3(ctx context.Context, logger *slog.Logger, saved savedObject) {
	logger.DebugContext(ctx, "saved file to s3",
		slog.String("file_name", saved.FileName),
		slog.String("object_key", saved.ObjectKey),
		slog.Float64("size_mb", float64(saved.SizeBytes)/(1024*1024)),
		slog.Duration("duration", time.Since(saved.StartedAt)))
}

func (b *bucket) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	client, err := b.getClient(ctx)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to build the s3 client",
			slog.Any("bucket", b.settings.Bucket),
			slog.Any("error", err),
		)

		return nil, err
	}

	coreClient := &minio.Core{Client: client}

	baseKey := b.buildObjectKey(fileName)

	logger.DebugContext(ctx, "reading file from s3", slog.Any("file_name", fileName), slog.Any("object_key", baseKey))

	manifest, hasManifest, err := b.readManifest(ctx, coreClient.Client, baseKey)
	if err != nil {
		return nil, err
	}

	if hasManifest {
		return newReassemblingReader(reassemblingReaderSpec{
			streamCtx:           ctx,
			logger:              logger,
			coreClient:          coreClient,
			bucket:              b.settings.Bucket,
			parts:               manifest.Parts,
			hasManifestChecksum: true,
		}), nil
	}

	objectInfo, err := coreClient.StatObject(ctx, b.settings.Bucket, baseKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("file does not exist in S3: %w", err)
		}

		return nil, fmt.Errorf("failed to stat file in S3: %w", err)
	}

	return newReassemblingReader(reassemblingReaderSpec{
		streamCtx:  ctx,
		logger:     logger,
		coreClient: coreClient,
		bucket:     b.settings.Bucket,
		parts:      []objectSpan{{Key: baseKey, Size: objectInfo.Size}},
	}), nil
}

// Chunks are found by listing rather than only through the manifest, because a
// write can complete numbered objects before it publishes one. The multipart abort
// matters too: an interrupted upload keeps consuming storage until it is aborted.
func (b *bucket) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s3DeleteTimeout)
	defer cancel()

	client, err := b.getClient(deleteCtx)
	if err != nil {
		return err
	}

	baseKey := b.buildObjectKey(fileName)

	keys, err := b.derivedObjectKeys(deleteCtx, client, baseKey)
	if err != nil {
		logger.WarnContext(ctx, "failed to discover s3 objects to delete",
			slog.Any("file_name", fileName), slog.Any("object_key", baseKey), slog.Any("error", err))

		return err
	}

	if err := b.removeObjects(deleteCtx, client, keys); err != nil {
		logger.WarnContext(ctx, "failed to delete file from s3",
			slog.Any("file_name", fileName), slog.Any("object_key", baseKey), slog.Any("error", err))

		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	if err := b.abortDerivedMultipartUploads(deleteCtx, client, baseKey); err != nil {
		logger.WarnContext(ctx, "failed to abort incomplete s3 uploads",
			slog.Any("file_name", fileName), slog.Any("object_key", baseKey), slog.Any("error", err))

		return err
	}

	logger.DebugContext(ctx, "deleted file from s3",
		slog.String("file_name", fileName),
		slog.String("object_key", baseKey),
		slog.Int("object_count", len(keys)))

	return nil
}

// A backup credential may be scoped to the configured prefix and denied deletes on purpose, so the
// probe stays on a single key inside the prefix and never lets a refused delete fail the storage:
// writing and reading back is everything a backup needs from S3.
func (b *bucket) testConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s3ConnectionTestTimeout)
	defer cancel()

	client, err := b.getClient(ctx)
	if err != nil {
		return err
	}

	probeKey := b.buildObjectKey(connectionTestObjectName)
	probeContent := []byte(connectionTestObjectContent)

	if err := b.writeConnectionProbe(ctx, client, probeKey, probeContent); err != nil {
		return err
	}

	if err := b.readAndVerifyConnectionProbe(ctx, client, probeKey, probeContent); err != nil {
		return err
	}

	_ = b.deleteConnectionProbe(ctx, client, probeKey)

	return nil
}

// Listing matches the exact derived grammar, so deleting "backup-1" cannot reach
// "backup-10". The manifest key is removed unconditionally, so no caller may name a
// sidecar "<base>.parts".
func (b *bucket) derivedObjectKeys(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) ([]string, error) {
	keys := []string{baseKey, manifestObjectKey(baseKey)}

	manifest, hasManifest, err := b.readManifest(ctx, client, baseKey)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{keys[0]: {}, keys[1]: {}}

	if hasManifest {
		for _, part := range manifest.Parts {
			if _, duplicate := seen[part.Key]; duplicate {
				continue
			}

			seen[part.Key] = struct{}{}
			keys = append(keys, part.Key)
		}
	}

	for object := range client.ListObjects(ctx, b.settings.Bucket, minio.ListObjectsOptions{
		Prefix:    baseKey + partSuffix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list chunk objects: %w", object.Err)
		}

		if !isPartObjectKey(baseKey, object.Key) {
			continue
		}

		if _, duplicate := seen[object.Key]; duplicate {
			continue
		}

		seen[object.Key] = struct{}{}
		keys = append(keys, object.Key)
	}

	return keys, nil
}

func (b *bucket) abortDerivedMultipartUploads(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) error {
	coreClient := &minio.Core{Client: client}

	for upload := range client.ListIncompleteUploads(ctx, b.settings.Bucket, baseKey, true) {
		if upload.Err != nil {
			return fmt.Errorf("failed to list incomplete uploads: %w", upload.Err)
		}

		if upload.Key != baseKey && !isPartObjectKey(baseKey, upload.Key) {
			continue
		}

		if err := coreClient.AbortMultipartUpload(ctx, b.settings.Bucket, upload.Key, upload.UploadID); err != nil {
			return fmt.Errorf("failed to abort incomplete upload for %s: %w", upload.Key, err)
		}
	}

	return nil
}

func (b *bucket) writeConnectionProbe(
	ctx context.Context,
	client *minio.Client,
	objectKey string,
	content []byte,
) error {
	ctx, cancel := context.WithTimeout(ctx, s3ConnectionTestTimeout)
	defer cancel()

	// The probe carries the configured storage class so a policy that constrains
	// s3:x-amz-storage-class accepts or rejects it exactly as it would a real backup.
	options := b.putObjectOptions()
	options.SendContentMd5 = true

	_, err := client.PutObject(
		ctx,
		b.settings.Bucket,
		objectKey,
		bytes.NewReader(content),
		int64(len(content)),
		options,
	)
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("failed to connect to the bucket. Please check params")
	}

	switch minio.ToErrorResponse(err).Code {
	case s3ErrCodeNoSuchBucket:
		return fmt.Errorf("bucket '%s' does not exist", b.settings.Bucket)
	case s3ErrCodeAccessDenied:
		return fmt.Errorf("credentials cannot write to '%s' in bucket '%s'", objectKey, b.settings.Bucket)
	}

	return fmt.Errorf("failed to upload test file to S3: %w", err)
}

func (b *bucket) readAndVerifyConnectionProbe(
	ctx context.Context,
	client *minio.Client,
	objectKey string,
	writtenContent []byte,
) error {
	ctx, cancel := context.WithTimeout(ctx, s3ConnectionTestTimeout)
	defer cancel()

	object, err := client.GetObject(ctx, b.settings.Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return b.describeConnectionProbeReadError(err, objectKey)
	}

	defer func() {
		_ = object.Close()
	}()

	storedContent, err := io.ReadAll(object)
	if err != nil {
		return b.describeConnectionProbeReadError(err, objectKey)
	}

	if !bytes.Equal(storedContent, writtenContent) {
		return fmt.Errorf("storage returned different content than was written to '%s'", objectKey)
	}

	return nil
}

func (b *bucket) describeConnectionProbeReadError(err error, objectKey string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("failed to connect to the bucket. Please check params")
	}

	if minio.ToErrorResponse(err).Code == s3ErrCodeAccessDenied {
		return fmt.Errorf("credentials cannot read back from '%s', so restores would fail", objectKey)
	}

	return fmt.Errorf("failed to read test file from S3: %w", err)
}

func (b *bucket) deleteConnectionProbe(ctx context.Context, client *minio.Client, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, s3ConnectionTestTimeout)
	defer cancel()

	return client.RemoveObject(ctx, b.settings.Bucket, objectKey, minio.RemoveObjectOptions{})
}

func (b *bucket) uploadSingleObject(
	ctx context.Context,
	client *minio.Client,
	objectKey string,
	data []byte,
) error {
	opts := b.putObjectOptions()
	opts.SendContentMd5 = true

	_, err := client.PutObject(ctx, b.settings.Bucket, objectKey, bytes.NewReader(data), int64(len(data)), opts)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (b *bucket) uploadChunked(
	ctx context.Context,
	coreClient *minio.Core,
	client *minio.Client,
	baseKey string,
	source io.Reader,
) error {
	innerPartSize := b.provider.partLayout.innerPartSize
	maxParts := b.provider.partLayout.maxPartsPerObject
	buf := make([]byte, innerPartSize)

	var objectSpans []objectSpan
	var uploadedKeys []string
	var totalSize int64

	// abandon undoes a failed chunked upload: it aborts the in-flight multipart (when uploadID is set)
	// and deletes every already-completed part object. Its context is detached from the cancellation
	// of ctx because the usual reason we land here is ctx cancellation, and cleanup issued on a cancelled
	// context is a no-op — which would leave completed part objects with no manifest as dead weight in
	// the bucket until retention or an overwrite reclaims them.
	abandon := func(objectKey, uploadID string) {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s3DeleteTimeout)
		defer cancel()

		if uploadID != "" {
			_ = coreClient.AbortMultipartUpload(cleanupCtx, b.settings.Bucket, objectKey, uploadID)
		}

		if len(uploadedKeys) > 0 {
			_ = b.removeObjects(cleanupCtx, client, uploadedKeys)
		}
	}

	objectIndex := 1
	sourceEnded := false

	for !sourceEnded {
		objectKey := partObjectKey(baseKey, objectIndex)

		uploadID, err := coreClient.NewMultipartUpload(ctx, b.settings.Bucket, objectKey, b.putObjectOptions())
		if err != nil {
			abandon("", "")
			return fmt.Errorf("failed to initiate multipart upload for %s: %w", objectKey, err)
		}

		var parts []minio.CompletePart
		objectHasher := sha256.New()
		var objectSize int64

		for partNumber := 1; partNumber <= maxParts; partNumber++ {
			select {
			case <-ctx.Done():
				abandon(objectKey, uploadID)
				return fmt.Errorf("upload cancelled: %w", ctx.Err())
			default:
			}

			partLen, readErr := io.ReadFull(source, buf)
			if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
				abandon(objectKey, uploadID)
				return fmt.Errorf("read error: %w", readErr)
			}

			if partLen == 0 && readErr == io.EOF {
				sourceEnded = true
				break
			}

			if err := b.putInnerPart(
				ctx,
				coreClient,
				objectKey,
				uploadID,
				partNumber,
				buf[:partLen],
				objectHasher,
				&parts,
			); err != nil {
				abandon(objectKey, uploadID)
				return err
			}

			objectSize += int64(partLen)

			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				sourceEnded = true
				break
			}
		}

		// The source ended exactly on the previous object's boundary, so this object got no data.
		if len(parts) == 0 {
			_ = coreClient.AbortMultipartUpload(ctx, b.settings.Bucket, objectKey, uploadID)
			break
		}

		if _, err := coreClient.CompleteMultipartUpload(
			ctx,
			b.settings.Bucket,
			objectKey,
			uploadID,
			parts,
			b.putObjectOptions(),
		); err != nil {
			abandon(objectKey, uploadID)
			return fmt.Errorf("failed to complete multipart upload for %s: %w", objectKey, err)
		}

		uploadedKeys = append(uploadedKeys, objectKey)
		objectSpans = append(objectSpans, objectSpan{
			Key:    objectKey,
			Size:   objectSize,
			SHA256: hex.EncodeToString(objectHasher.Sum(nil)),
		})
		totalSize += objectSize
		objectIndex++
	}

	manifest := chunkedManifest{
		Version:   manifestVersion,
		TotalSize: totalSize,
		Parts:     objectSpans,
	}

	if err := b.writeManifest(ctx, client, baseKey, manifest); err != nil {
		abandon("", "")
		return err
	}

	return nil
}

func (b *bucket) putInnerPart(
	ctx context.Context,
	coreClient *minio.Core,
	objectKey string,
	uploadID string,
	partNumber int,
	partData []byte,
	objectHasher hash.Hash,
	parts *[]minio.CompletePart,
) error {
	// S3 requires a Content-MD5 checksum for each part; it checks integrity, not authenticity.
	// nosemgrep: go.lang.security.audit.crypto.use_of_weak_crypto.use-of-md5
	partMd5 := md5.Sum(partData)
	md5Base64 := base64.StdEncoding.EncodeToString(partMd5[:])

	part, err := coreClient.PutObjectPart(
		ctx,
		b.settings.Bucket,
		objectKey,
		uploadID,
		partNumber,
		bytes.NewReader(partData),
		int64(len(partData)),
		minio.PutObjectPartOptions{Md5Base64: md5Base64},
	)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("upload cancelled: %w", ctx.Err())
		}

		return fmt.Errorf("failed to upload part %d of %s: %w", partNumber, objectKey, err)
	}

	objectHasher.Write(partData)
	*parts = append(*parts, minio.CompletePart{PartNumber: partNumber, ETag: part.ETag})

	return nil
}

func (b *bucket) putObjectOptions() minio.PutObjectOptions {
	return minio.PutObjectOptions{
		StorageClass: b.settings.StorageClass,
	}
}

func (b *bucket) buildObjectKey(fileName string) string {
	if b.settings.Prefix == "" {
		return fileName
	}

	prefix := b.settings.Prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return prefix + fileName
}

func (b *bucket) getClient(ctx context.Context) (*minio.Client, error) {
	params, err := b.getClientParams()
	if err != nil {
		return nil, err
	}

	region := b.settings.Region
	if region == "" {
		region = b.provider.bucketRegions.getRegion(ctx, params, b.settings.Bucket)
	}

	minioClient, err := newMinioClient(params, region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return minioClient, nil
}

type clientParams struct {
	endpoint     string
	useSSL       bool
	accessKey    string
	secretKey    string
	bucketLookup minio.BucketLookupType
	transport    *http.Transport
}

func newMinioClient(params clientParams, region string) (*minio.Client, error) {
	return minio.New(params.endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(params.accessKey, params.secretKey, ""),
		Secure:       params.useSSL,
		Region:       region,
		BucketLookup: params.bucketLookup,
		Transport:    params.transport,
	})
}

func (b *bucket) getClientParams() (clientParams, error) {
	endpoint := b.settings.Endpoint
	useSSL := true

	if after, ok := strings.CutPrefix(endpoint, "http://"); ok {
		useSSL = false
		endpoint = after
	} else if after, ok := strings.CutPrefix(endpoint, "https://"); ok {
		endpoint = after
	}

	if endpoint == "" {
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", b.settings.Region)
	}

	accessKey, err := b.provider.fieldEncryptor.Decrypt(b.settings.AccessKey)
	if err != nil {
		return clientParams{}, fmt.Errorf("failed to decrypt S3 access key: %w", err)
	}

	secretKey, err := b.provider.fieldEncryptor.Decrypt(b.settings.SecretKey)
	if err != nil {
		return clientParams{}, fmt.Errorf("failed to decrypt S3 secret key: %w", err)
	}

	bucketLookup := minio.BucketLookupAuto
	if b.settings.IsVirtualHostedStyle {
		bucketLookup = minio.BucketLookupDNS
	}

	return clientParams{
		endpoint:     endpoint,
		useSSL:       useSSL,
		accessKey:    accessKey,
		secretKey:    secretKey,
		bucketLookup: bucketLookup,
		transport:    b.provider.transports.get(b.settings.IsSkipTLSVerify),
	}, nil
}

func (b *bucket) removeObjects(ctx context.Context, client *minio.Client, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(keys))
	for _, key := range keys {
		objectsCh <- minio.ObjectInfo{Key: key}
	}
	close(objectsCh)

	var firstErr error
	for removeResult := range client.RemoveObjects(ctx, b.settings.Bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if removeResult.Err == nil || firstErr != nil {
			continue
		}

		// AWS reports a missing key as deleted, but some S3-compatible servers (Garage before
		// v2.4.0) return it as a per-object NoSuchKey. Either way the object is confirmed absent.
		if minio.ToErrorResponse(removeResult.Err).Code == s3ErrCodeNoSuchKey {
			continue
		}

		firstErr = fmt.Errorf("object %s: %w", removeResult.ObjectName, removeResult.Err)
	}

	return firstErr
}
