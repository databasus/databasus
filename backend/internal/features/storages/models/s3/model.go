package s3_storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"databasus-backend/internal/util/encryption"
)

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

type S3Storage struct {
	StorageID   uuid.UUID `json:"storageId"   gorm:"primaryKey;type:uuid;column:storage_id"`
	S3Bucket    string    `json:"s3Bucket"    gorm:"not null;type:text;column:s3_bucket"`
	S3Region    string    `json:"s3Region"    gorm:"not null;type:text;column:s3_region"`
	S3AccessKey string    `json:"s3AccessKey" gorm:"not null;type:text;column:s3_access_key"`
	S3SecretKey string    `json:"s3SecretKey" gorm:"not null;type:text;column:s3_secret_key"`
	S3Endpoint  string    `json:"s3Endpoint"  gorm:"type:text;column:s3_endpoint"`

	S3Prefix                string         `json:"s3Prefix"                gorm:"type:text;column:s3_prefix"`
	S3UseVirtualHostedStyle bool           `json:"s3UseVirtualHostedStyle" gorm:"default:false;column:s3_use_virtual_hosted_style"`
	SkipTLSVerify           bool           `json:"skipTLSVerify"           gorm:"default:false;column:skip_tls_verify"`
	S3StorageClass          S3StorageClass `json:"s3StorageClass"          gorm:"type:text;column:s3_storage_class;default:''"`

	// Test seams: rolling at the production 15.6 GiB object boundary would need ~15 GiB per case,
	// so integration tests shrink the inner part and per-object part count to exercise object
	// rollover on a few tens of MiB. Unexported, so gorm and JSON ignore them; zero means default.
	innerPartSizeOverride int
	maxPartsOverride      int
}

func (s *S3Storage) TableName() string {
	return "s3_storages"
}

func (s *S3Storage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("upload cancelled before start: %w", ctx.Err())
	default:
	}

	client, err := s.getClient(ctx, encryptor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to build the s3 client", "bucket", s.S3Bucket, "error", err)

		return err
	}

	baseKey := s.buildObjectKey(fileName)
	innerPartSize := s.innerPartSize()

	startedAt := time.Now().UTC()

	logger.DebugContext(ctx, "saving file to s3", "file_name", fileName,
		"bucket", s.S3Bucket, "object_key", baseKey)

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
		if err := s.uploadSingleObject(ctx, client, baseKey, firstPart[:firstPartLen]); err != nil {
			logger.ErrorContext(ctx, "failed to save file to s3", "file_name", fileName,
				"object_key", baseKey, "error", err)

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
		if err := s.uploadSingleObject(ctx, client, baseKey, firstPart); err != nil {
			logger.ErrorContext(ctx, "failed to save file to s3", "file_name", fileName,
				"object_key", baseKey, "error", err)

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
		"file_name", fileName, "object_key", baseKey)

	if err := s.uploadChunked(ctx, coreClient, client, baseKey, source); err != nil {
		logger.ErrorContext(ctx, "failed to save chunked file to s3", "file_name", fileName,
			"object_key", baseKey, "error", err)

		return err
	}

	logger.DebugContext(ctx, fmt.Sprintf("saved chunked file to s3 in %s", time.Since(startedAt)),
		"file_name", fileName, "object_key", baseKey)

	return nil
}

type savedObject struct {
	FileName  string
	ObjectKey string
	SizeBytes int64
	StartedAt time.Time
}

func reportSaveToS3(ctx context.Context, logger *slog.Logger, saved savedObject) {
	logger.DebugContext(ctx, fmt.Sprintf("saved file to s3: %.2f MB in %s",
		float64(saved.SizeBytes)/(1024*1024), time.Since(saved.StartedAt)),
		"file_name", saved.FileName, "object_key", saved.ObjectKey)
}

func (s *S3Storage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	client, err := s.getClient(ctx, encryptor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to build the s3 client", "bucket", s.S3Bucket, "error", err)

		return nil, err
	}

	coreClient := &minio.Core{Client: client}

	baseKey := s.buildObjectKey(fileName)

	logger.DebugContext(ctx, "reading file from s3", "file_name", fileName, "object_key", baseKey)

	manifest, hasManifest, err := s.readManifest(ctx, coreClient.Client, baseKey)
	if err != nil {
		return nil, err
	}

	if hasManifest {
		return newReassemblingReader(reassemblingReaderSpec{
			streamCtx:           ctx,
			logger:              logger,
			coreClient:          coreClient,
			bucket:              s.S3Bucket,
			parts:               manifest.Parts,
			hasManifestChecksum: true,
		}), nil
	}

	objectInfo, err := coreClient.StatObject(ctx, s.S3Bucket, baseKey, minio.StatObjectOptions{})
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
		bucket:     s.S3Bucket,
		parts:      []objectSpan{{Key: baseKey, Size: objectInfo.Size}},
	}), nil
}

// Chunks are found by listing rather than only through the manifest, because a
// write can complete numbered objects before it publishes one. The multipart abort
// matters too: an interrupted upload keeps consuming storage until it is aborted.
func (s *S3Storage) DeleteFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation carries its own deadline; the caller's ctx stays for log correlation only.
	deleteCtx, cancel := context.WithTimeout(context.Background(), s3DeleteTimeout)
	defer cancel()

	client, err := s.getClient(deleteCtx, encryptor)
	if err != nil {
		return err
	}

	baseKey := s.buildObjectKey(fileName)

	keys, err := s.derivedObjectKeys(deleteCtx, client, baseKey)
	if err != nil {
		logger.WarnContext(ctx, "failed to discover s3 objects to delete",
			"file_name", fileName, "object_key", baseKey, "error", err)

		return err
	}

	if err := s.removeObjects(deleteCtx, client, keys); err != nil {
		logger.WarnContext(ctx, "failed to delete file from s3",
			"file_name", fileName, "object_key", baseKey, "error", err)

		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	if err := s.abortDerivedMultipartUploads(deleteCtx, client, baseKey); err != nil {
		logger.WarnContext(ctx, "failed to abort incomplete s3 uploads",
			"file_name", fileName, "object_key", baseKey, "error", err)

		return err
	}

	logger.DebugContext(ctx, fmt.Sprintf("deleted file from s3 (%d objects)", len(keys)),
		"file_name", fileName, "object_key", baseKey)

	return nil
}

func (s *S3Storage) Validate(encryptor encryption.FieldEncryptor) error {
	if s.S3Bucket == "" {
		return errors.New("S3 bucket is required")
	}
	if s.S3AccessKey == "" {
		return errors.New("S3 access key is required")
	}
	if s.S3SecretKey == "" {
		return errors.New("S3 secret key is required")
	}

	return nil
}

// A backup credential may be scoped to the configured prefix and denied deletes on purpose, so the
// probe stays on a single key inside the prefix and never lets a refused delete fail the storage:
// writing and reading back is everything a backup needs from S3.
func (s *S3Storage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3ConnectionTestTimeout)
	defer cancel()

	client, err := s.getClient(ctx, encryptor)
	if err != nil {
		return err
	}

	probeKey := s.buildObjectKey(connectionTestObjectName)
	probeContent := []byte(connectionTestObjectContent)

	if err := s.writeConnectionProbe(client, probeKey, probeContent); err != nil {
		return err
	}

	if err := s.readAndVerifyConnectionProbe(client, probeKey, probeContent); err != nil {
		return err
	}

	_ = s.deleteConnectionProbe(client, probeKey)

	return nil
}

func (s *S3Storage) HideSensitiveData() {
	s.S3AccessKey = ""
	s.S3SecretKey = ""
}

func (s *S3Storage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.S3AccessKey != "" {
		s.S3AccessKey, err = encryptor.Encrypt(s.S3AccessKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt S3 access key: %w", err)
		}
	}

	if s.S3SecretKey != "" {
		s.S3SecretKey, err = encryptor.Encrypt(s.S3SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt S3 secret key: %w", err)
		}
	}

	return nil
}

func (s *S3Storage) Update(incoming *S3Storage) {
	s.S3Bucket = incoming.S3Bucket
	s.S3Region = incoming.S3Region
	s.S3Endpoint = incoming.S3Endpoint
	s.S3UseVirtualHostedStyle = incoming.S3UseVirtualHostedStyle
	s.SkipTLSVerify = incoming.SkipTLSVerify
	s.S3StorageClass = incoming.S3StorageClass

	if incoming.S3AccessKey != "" {
		s.S3AccessKey = incoming.S3AccessKey
	}

	if incoming.S3SecretKey != "" {
		s.S3SecretKey = incoming.S3SecretKey
	}

	// we do not allow to change the prefix after creation,
	// otherwise we will have to transfer all the data to the new prefix
}

// Listing matches the exact derived grammar, so deleting "backup-1" cannot reach
// "backup-10". The manifest key is removed unconditionally, so no caller may name a
// sidecar "<base>.parts".
func (s *S3Storage) derivedObjectKeys(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) ([]string, error) {
	keys := []string{baseKey, manifestObjectKey(baseKey)}

	manifest, hasManifest, err := s.readManifest(ctx, client, baseKey)
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

	for object := range client.ListObjects(ctx, s.S3Bucket, minio.ListObjectsOptions{
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

func (s *S3Storage) abortDerivedMultipartUploads(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) error {
	coreClient := &minio.Core{Client: client}

	for upload := range client.ListIncompleteUploads(ctx, s.S3Bucket, baseKey, true) {
		if upload.Err != nil {
			return fmt.Errorf("failed to list incomplete uploads: %w", upload.Err)
		}

		if upload.Key != baseKey && !isPartObjectKey(baseKey, upload.Key) {
			continue
		}

		if err := coreClient.AbortMultipartUpload(ctx, s.S3Bucket, upload.Key, upload.UploadID); err != nil {
			return fmt.Errorf("failed to abort incomplete upload for %s: %w", upload.Key, err)
		}
	}

	return nil
}

func (s *S3Storage) writeConnectionProbe(
	client *minio.Client,
	objectKey string,
	content []byte,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3ConnectionTestTimeout)
	defer cancel()

	// The probe carries the configured storage class so a policy that constrains
	// s3:x-amz-storage-class accepts or rejects it exactly as it would a real backup.
	options := s.putObjectOptions()
	options.SendContentMd5 = true

	_, err := client.PutObject(
		ctx,
		s.S3Bucket,
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
		return fmt.Errorf("bucket '%s' does not exist", s.S3Bucket)
	case s3ErrCodeAccessDenied:
		return fmt.Errorf("credentials cannot write to '%s' in bucket '%s'", objectKey, s.S3Bucket)
	}

	return fmt.Errorf("failed to upload test file to S3: %w", err)
}

func (s *S3Storage) readAndVerifyConnectionProbe(
	client *minio.Client,
	objectKey string,
	writtenContent []byte,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3ConnectionTestTimeout)
	defer cancel()

	object, err := client.GetObject(ctx, s.S3Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return s.describeConnectionProbeReadError(err, objectKey)
	}

	defer func() {
		_ = object.Close()
	}()

	storedContent, err := io.ReadAll(object)
	if err != nil {
		return s.describeConnectionProbeReadError(err, objectKey)
	}

	if !bytes.Equal(storedContent, writtenContent) {
		return fmt.Errorf("storage returned different content than was written to '%s'", objectKey)
	}

	return nil
}

func (s *S3Storage) describeConnectionProbeReadError(err error, objectKey string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("failed to connect to the bucket. Please check params")
	}

	if minio.ToErrorResponse(err).Code == s3ErrCodeAccessDenied {
		return fmt.Errorf("credentials cannot read back from '%s', so restores would fail", objectKey)
	}

	return fmt.Errorf("failed to read test file from S3: %w", err)
}

func (s *S3Storage) deleteConnectionProbe(client *minio.Client, objectKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3ConnectionTestTimeout)
	defer cancel()

	return client.RemoveObject(ctx, s.S3Bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *S3Storage) innerPartSize() int {
	if s.innerPartSizeOverride > 0 {
		return s.innerPartSizeOverride
	}

	return defaultInnerPartSize
}

func (s *S3Storage) maxPartsPerObject() int {
	if s.maxPartsOverride > 0 {
		return s.maxPartsOverride
	}

	return defaultMaxPartsPerObject
}

func (s *S3Storage) uploadSingleObject(
	ctx context.Context,
	client *minio.Client,
	objectKey string,
	data []byte,
) error {
	opts := s.putObjectOptions()
	opts.SendContentMd5 = true

	_, err := client.PutObject(ctx, s.S3Bucket, objectKey, bytes.NewReader(data), int64(len(data)), opts)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) uploadChunked(
	ctx context.Context,
	coreClient *minio.Core,
	client *minio.Client,
	baseKey string,
	source io.Reader,
) error {
	innerPartSize := s.innerPartSize()
	maxParts := s.maxPartsPerObject()
	buf := make([]byte, innerPartSize)

	var objectSpans []objectSpan
	var uploadedKeys []string
	var totalSize int64

	// abandon undoes a failed chunked upload: it aborts the in-flight multipart (when uploadID is set)
	// and deletes every already-completed part object. It uses a fresh context detached from ctx
	// because the usual reason we land here is ctx cancellation, and cleanup issued on a cancelled
	// context is a no-op — which would leave completed part objects with no manifest as dead weight in
	// the bucket until retention or an overwrite reclaims them.
	abandon := func(objectKey, uploadID string) {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), s3DeleteTimeout)
		defer cancel()

		if uploadID != "" {
			_ = coreClient.AbortMultipartUpload(cleanupCtx, s.S3Bucket, objectKey, uploadID)
		}

		if len(uploadedKeys) > 0 {
			_ = s.removeObjects(cleanupCtx, client, uploadedKeys)
		}
	}

	objectIndex := 1
	sourceEnded := false

	for !sourceEnded {
		objectKey := partObjectKey(baseKey, objectIndex)

		uploadID, err := coreClient.NewMultipartUpload(ctx, s.S3Bucket, objectKey, s.putObjectOptions())
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

			if err := s.putInnerPart(
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
			_ = coreClient.AbortMultipartUpload(ctx, s.S3Bucket, objectKey, uploadID)
			break
		}

		if _, err := coreClient.CompleteMultipartUpload(
			ctx,
			s.S3Bucket,
			objectKey,
			uploadID,
			parts,
			s.putObjectOptions(),
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

	if err := s.writeManifest(ctx, client, baseKey, manifest); err != nil {
		abandon("", "")
		return err
	}

	return nil
}

func (s *S3Storage) putInnerPart(
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
		s.S3Bucket,
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

func (s *S3Storage) putObjectOptions() minio.PutObjectOptions {
	return minio.PutObjectOptions{
		StorageClass: string(s.S3StorageClass),
	}
}

func (s *S3Storage) buildObjectKey(fileName string) string {
	if s.S3Prefix == "" {
		return fileName
	}

	prefix := s.S3Prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return prefix + fileName
}

func (s *S3Storage) getClient(ctx context.Context, encryptor encryption.FieldEncryptor) (*minio.Client, error) {
	params, err := s.getClientParams(encryptor)
	if err != nil {
		return nil, err
	}

	region := s.S3Region
	if region == "" {
		region = getCachedBucketRegion(ctx, params, s.S3Bucket)
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

func (s *S3Storage) getClientParams(encryptor encryption.FieldEncryptor) (clientParams, error) {
	endpoint := s.S3Endpoint
	useSSL := true

	if after, ok := strings.CutPrefix(endpoint, "http://"); ok {
		useSSL = false
		endpoint = after
	} else if after, ok := strings.CutPrefix(endpoint, "https://"); ok {
		endpoint = after
	}

	if endpoint == "" {
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", s.S3Region)
	}

	accessKey, err := encryptor.Decrypt(s.S3AccessKey)
	if err != nil {
		return clientParams{}, fmt.Errorf("failed to decrypt S3 access key: %w", err)
	}

	secretKey, err := encryptor.Decrypt(s.S3SecretKey)
	if err != nil {
		return clientParams{}, fmt.Errorf("failed to decrypt S3 secret key: %w", err)
	}

	bucketLookup := minio.BucketLookupAuto
	if s.S3UseVirtualHostedStyle {
		bucketLookup = minio.BucketLookupDNS
	}

	return clientParams{
		endpoint:     endpoint,
		useSSL:       useSSL,
		accessKey:    accessKey,
		secretKey:    secretKey,
		bucketLookup: bucketLookup,
		transport:    getSharedTransport(s.SkipTLSVerify),
	}, nil
}

// chunkedManifest lists, in order, the objects a single logical file was split into, with each
// object's size and sha256. It is written LAST, after every part object is durably committed.
//
// Why a manifest instead of just probing {fileName}.part1, .part2, ... until a key 404s:
//
//  1. The bytes we store are an encrypted stream with no end-of-stream marker (see
//     util/encryption). Probe-until-404 cannot tell a complete backup from one whose final part
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

func (s *S3Storage) writeManifest(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
	manifest chunkedManifest,
) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk manifest: %w", err)
	}

	opts := s.putObjectOptions()
	opts.SendContentMd5 = true
	opts.ContentType = "application/json"

	_, err = client.PutObject(
		ctx,
		s.S3Bucket,
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

func (s *S3Storage) readManifest(
	ctx context.Context,
	client *minio.Client,
	baseKey string,
) (chunkedManifest, bool, error) {
	var manifest chunkedManifest

	manifestKey := manifestObjectKey(baseKey)

	_, err := client.StatObject(ctx, s.S3Bucket, manifestKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return manifest, false, nil
		}

		return manifest, false, fmt.Errorf("failed to stat chunk manifest: %w", err)
	}

	object, err := client.GetObject(ctx, s.S3Bucket, manifestKey, minio.GetObjectOptions{})
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

func (s *S3Storage) removeObjects(ctx context.Context, client *minio.Client, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(keys))
	for _, key := range keys {
		objectsCh <- minio.ObjectInfo{Key: key}
	}
	close(objectsCh)

	var firstErr error
	for removeResult := range client.RemoveObjects(ctx, s.S3Bucket, objectsCh, minio.RemoveObjectsOptions{}) {
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
