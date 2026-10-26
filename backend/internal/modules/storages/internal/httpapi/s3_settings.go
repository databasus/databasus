package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/s3"

func toS3Settings(body S3StorageSettings) s3.Settings {
	storageClass := s3.StorageClassDefault
	if body.S3StorageClass != nil {
		storageClass = s3.StorageClass(*body.S3StorageClass)
	}

	return s3.Settings{
		Bucket:               body.S3Bucket,
		Region:               getStringOrEmpty(body.S3Region),
		AccessKey:            getStringOrEmpty(body.S3AccessKey),
		SecretKey:            getStringOrEmpty(body.S3SecretKey),
		Endpoint:             getStringOrEmpty(body.S3Endpoint),
		Prefix:               getStringOrEmpty(body.S3Prefix),
		IsVirtualHostedStyle: body.IsVirtualHostedStyle,
		IsSkipTLSVerify:      body.IsSkipTlsVerify,
		StorageClass:         storageClass,
	}
}

func fromS3Settings(settings s3.Settings) S3StorageSettings {
	var storageClass *S3StorageClass
	if settings.StorageClass != s3.StorageClassDefault {
		bodyStorageClass := S3StorageClass(settings.StorageClass)
		storageClass = &bodyStorageClass
	}

	return S3StorageSettings{
		Type:                 S3StorageSettingsTypeS3,
		S3Bucket:             settings.Bucket,
		S3Region:             toOptionalString(settings.Region),
		S3AccessKey:          nil,
		S3SecretKey:          nil,
		S3Endpoint:           toOptionalString(settings.Endpoint),
		S3Prefix:             toOptionalString(settings.Prefix),
		IsVirtualHostedStyle: settings.IsVirtualHostedStyle,
		IsSkipTlsVerify:      settings.IsSkipTLSVerify,
		S3StorageClass:       storageClass,
	}
}
