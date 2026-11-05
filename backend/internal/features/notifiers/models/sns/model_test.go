package sns_notifier

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
)

func Test_SNSNotifier_Validate_KeysMode(t *testing.T) {
	encryptor := encryption.GetFieldEncryptor()

	notifierWithoutTopic := &SNSNotifier{
		Region:   "us-east-1",
		AuthType: SNSAuthTypeKeys,
	}
	require.Error(t, notifierWithoutTopic.Validate(encryptor))

	notifierWithInvalidTopic := &SNSNotifier{
		TopicARN: "invalid-topic-arn",
		Region:   "us-east-1",
		AuthType: SNSAuthTypeKeys,
	}
	require.Error(t, notifierWithInvalidTopic.Validate(encryptor))

	notifierWithoutKeys := &SNSNotifier{
		TopicARN: "arn:aws:sns:us-east-1:123456789012:my-topic",
		Region:   "us-east-1",
		AuthType: SNSAuthTypeKeys,
	}
	require.Error(t, notifierWithoutKeys.Validate(encryptor))

	validKeysNotifier := &SNSNotifier{
		TopicARN:  "arn:aws:sns:us-east-1:123456789012:my-topic",
		Region:    "us-east-1",
		AuthType:  SNSAuthTypeKeys,
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "secret",
	}
	require.NoError(t, validKeysNotifier.Validate(encryptor))
}

func Test_SNSNotifier_Validate_IAMRoleMode(t *testing.T) {
	encryptor := encryption.GetFieldEncryptor()

	notifierWithoutRegionAndUnparseable := &SNSNotifier{
		TopicARN: "arn:aws:sns::123456789012:my-topic",
		AuthType: SNSAuthTypeIAMRole,
	}
	require.Error(t, notifierWithoutRegionAndUnparseable.Validate(encryptor))

	validIAMNotifier := &SNSNotifier{
		TopicARN: "arn:aws:sns:eu-west-1:123456789012:my-topic",
		AuthType: SNSAuthTypeIAMRole,
	}
	// Automatically extracts region from Topic ARN
	require.NoError(t, validIAMNotifier.Validate(encryptor))
	assert.Equal(t, "eu-west-1", validIAMNotifier.Region)
}

func Test_SNSNotifier_HideSensitiveData_ClearsKeysAndExternalID(t *testing.T) {
	notifier := &SNSNotifier{
		AccessKey:  "access",
		SecretKey:  "secret",
		ExternalID: "ext-id",
	}

	notifier.HideSensitiveData()

	assert.Empty(t, notifier.AccessKey)
	assert.Empty(t, notifier.SecretKey)
	assert.Empty(t, notifier.ExternalID)
}

func Test_FormatSNSSubject(t *testing.T) {
	subjectWithNewlines := formatSNSSubject("Backup\nFinished\rSuccessfully")
	assert.Equal(t, "Backup Finished Successfully", subjectWithNewlines)

	emptySubject := formatSNSSubject("   ")
	assert.Equal(t, "Databasus Notification", emptySubject)

	longRaw := strings.Repeat("A", 120)
	truncatedSubject := formatSNSSubject(longRaw)
	assert.LessOrEqual(t, len(truncatedSubject), maxSNSSubjectLength)
	assert.True(t, strings.HasSuffix(truncatedSubject, "..."))
}

func Test_SNSNotifier_Update(t *testing.T) {
	notifier := &SNSNotifier{
		TopicARN:  "arn:aws:sns:us-east-1:123456789012:old-topic",
		Region:    "us-east-1",
		AuthType:  SNSAuthTypeKeys,
		AccessKey: "old-key",
		SecretKey: "old-secret",
	}

	incoming := &SNSNotifier{
		TopicARN: "arn:aws:sns:us-west-2:123456789012:new-topic",
		Region:   "us-west-2",
		AuthType: SNSAuthTypeIAMRole,
		RoleARN:  "arn:aws:iam::123456789012:role/SNSRole",
	}

	notifier.Update(incoming)

	assert.Equal(t, "arn:aws:sns:us-west-2:123456789012:new-topic", notifier.TopicARN)
	assert.Equal(t, "us-west-2", notifier.Region)
	assert.Equal(t, SNSAuthTypeIAMRole, notifier.AuthType)
	assert.Equal(t, "arn:aws:iam::123456789012:role/SNSRole", notifier.RoleARN)
	assert.Equal(t, "old-key", notifier.AccessKey)
	assert.Equal(t, "old-secret", notifier.SecretKey)
}
