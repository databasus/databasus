package aws_auth

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BuildAWSConfig_KeysMode_MissingAccessKeyReturnsError(t *testing.T) {
	_, err := BuildAWSConfig(context.Background(), Config{
		AuthType:  AuthTypeKeys,
		AccessKey: "",
		SecretKey: "secret",
		Region:    "us-east-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access key is required")
}

func Test_BuildAWSConfig_KeysMode_MissingSecretKeyReturnsError(t *testing.T) {
	_, err := BuildAWSConfig(context.Background(), Config{
		AuthType:  AuthTypeKeys,
		AccessKey: "access",
		SecretKey: "",
		Region:    "us-east-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret key is required")
}

func Test_BuildAWSConfig_KeysMode_ValidCredentialsConfiguresProvider(t *testing.T) {
	awsConfig, err := BuildAWSConfig(context.Background(), Config{
		AuthType:  AuthTypeKeys,
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region:    "us-west-2",
	})

	require.NoError(t, err)
	assert.Equal(t, "us-west-2", awsConfig.Region)

	retrievedCredentials, err := awsConfig.Credentials.Retrieve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", retrievedCredentials.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", retrievedCredentials.SecretAccessKey)
}

func Test_BuildMinioCredentials_KeysMode_ReturnsStaticCredentials(t *testing.T) {
	minioCreds, err := BuildMinioCredentials(context.Background(), Config{
		AuthType:  AuthTypeKeys,
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Region:    "us-east-1",
	})

	require.NoError(t, err)
	require.NotNil(t, minioCreds)

	val, err := minioCreds.Get()
	require.NoError(t, err)
	assert.Equal(t, "test-access-key", val.AccessKeyID)
	assert.Equal(t, "test-secret-key", val.SecretAccessKey)
}

type staticTestCredentialsProvider struct {
	credentials aws.Credentials
}

func (s *staticTestCredentialsProvider) Retrieve(_ context.Context) (aws.Credentials, error) {
	return s.credentials, nil
}

func Test_MinioCredentialsBridge_RetrieveAndIsExpired(t *testing.T) {
	expiresAt := time.Now().Add(10 * time.Minute)
	testCreds := aws.Credentials{
		AccessKeyID:     "ASIAEXAMPLE",
		SecretAccessKey: "secretExample",
		SessionToken:    "tokenExample",
		CanExpire:       true,
		Expires:         expiresAt,
	}

	bridge := &MinioCredentialsBridge{
		credentialsProvider: &staticTestCredentialsProvider{credentials: testCreds},
	}

	val, err := bridge.Retrieve()
	require.NoError(t, err)
	assert.Equal(t, "ASIAEXAMPLE", val.AccessKeyID)
	assert.Equal(t, "secretExample", val.SecretAccessKey)
	assert.Equal(t, "tokenExample", val.SessionToken)
	assert.False(t, bridge.IsExpired())

	// Test expiration when expires is in past
	bridge.cachedCredentials.Expires = time.Now().Add(-5 * time.Minute)
	assert.True(t, bridge.IsExpired())
}
