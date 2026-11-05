package aws_auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	aws_credentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	minio_credentials "github.com/minio/minio-go/v7/pkg/credentials"
)

type AuthType string

const (
	AuthTypeKeys    AuthType = "KEYS"
	AuthTypeIAMRole AuthType = "IAM_ROLE"
)

func (a AuthType) IsValid() bool {
	return a == AuthTypeKeys || a == AuthTypeIAMRole
}

type Config struct {
	AuthType   AuthType
	Region     string
	AccessKey  string
	SecretKey  string
	RoleARN    string
	ExternalID string
	Endpoint   string
}

func BuildAWSConfig(ctx context.Context, config Config) (aws.Config, error) {
	configOptions := make([]func(*aws_config.LoadOptions) error, 0, 4)

	if config.Region != "" {
		configOptions = append(configOptions, aws_config.WithRegion(config.Region))
	}

	if config.AuthType == AuthTypeKeys {
		if config.AccessKey == "" {
			return aws.Config{}, errors.New("AWS access key is required for KEYS authentication")
		}

		if config.SecretKey == "" {
			return aws.Config{}, errors.New("AWS secret key is required for KEYS authentication")
		}

		staticCredentialsProvider := aws_credentials.NewStaticCredentialsProvider(
			config.AccessKey,
			config.SecretKey,
			"",
		)
		configOptions = append(configOptions, aws_config.WithCredentialsProvider(staticCredentialsProvider))
	}

	awsConfig, err := aws_config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	if config.AuthType == AuthTypeIAMRole && config.RoleARN != "" {
		stsClient := sts.NewFromConfig(awsConfig)

		assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, config.RoleARN, func(options *stscreds.AssumeRoleOptions) {
			options.RoleSessionName = "databasus-session"

			if config.ExternalID != "" {
				options.ExternalID = aws.String(config.ExternalID)
			}
		})

		awsConfig.Credentials = aws.NewCredentialsCache(assumeRoleProvider)
	}

	return awsConfig, nil
}

type MinioCredentialsBridge struct {
	credentialsProvider aws.CredentialsProvider
	cachedCredentials   aws.Credentials
}

func (b *MinioCredentialsBridge) RetrieveWithCredContext(_ *minio_credentials.CredContext) (minio_credentials.Value, error) {
	return b.Retrieve()
}

func (b *MinioCredentialsBridge) Retrieve() (minio_credentials.Value, error) {
	retrievedCredentials, err := b.credentialsProvider.Retrieve(context.Background())
	if err != nil {
		return minio_credentials.Value{}, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	b.cachedCredentials = retrievedCredentials

	return minio_credentials.Value{
		AccessKeyID:     retrievedCredentials.AccessKeyID,
		SecretAccessKey: retrievedCredentials.SecretAccessKey,
		SessionToken:    retrievedCredentials.SessionToken,
		SignerType:      minio_credentials.SignatureV4,
	}, nil
}

func (b *MinioCredentialsBridge) IsExpired() bool {
	if !b.cachedCredentials.CanExpire {
		return false
	}

	return time.Now().After(b.cachedCredentials.Expires.Add(-time.Minute))
}

func BuildMinioCredentials(ctx context.Context, config Config) (*minio_credentials.Credentials, error) {
	if config.AuthType == AuthTypeKeys {
		return minio_credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""), nil
	}

	awsConfig, err := BuildAWSConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	bridge := &MinioCredentialsBridge{
		credentialsProvider: awsConfig.Credentials,
	}

	return minio_credentials.New(bridge), nil
}
