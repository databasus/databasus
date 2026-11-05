package ses_notifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
)

func Test_SESNotifier_Validate_IAMRoleMode(t *testing.T) {
	encryptor := encryption.GetFieldEncryptor()

	notifierWithoutFrom := &SESNotifier{
		TargetEmail: "admin@example.com",
		Region:      "us-east-1",
		AuthType:    SESAuthTypeIAMRole,
	}
	require.Error(t, notifierWithoutFrom.Validate(encryptor))

	notifierWithInvalidFrom := &SESNotifier{
		FromEmail:   "not-an-email",
		TargetEmail: "admin@example.com",
		Region:      "us-east-1",
		AuthType:    SESAuthTypeIAMRole,
	}
	require.Error(t, notifierWithInvalidFrom.Validate(encryptor))

	notifierWithoutTarget := &SESNotifier{
		FromEmail: "sender@example.com",
		Region:    "us-east-1",
		AuthType:  SESAuthTypeIAMRole,
	}
	require.Error(t, notifierWithoutTarget.Validate(encryptor))

	notifierWithoutRegion := &SESNotifier{
		FromEmail:   "sender@example.com",
		TargetEmail: "admin@example.com",
		AuthType:    SESAuthTypeIAMRole,
	}
	require.Error(t, notifierWithoutRegion.Validate(encryptor))

	validNotifier := &SESNotifier{
		FromEmail:   "sender@example.com",
		TargetEmail: "admin@example.com",
		Region:      "us-east-1",
		AuthType:    SESAuthTypeIAMRole,
	}
	require.NoError(t, validNotifier.Validate(encryptor))
}

func Test_SESNotifier_Validate_KeysMode(t *testing.T) {
	encryptor := encryption.GetFieldEncryptor()

	notifierWithoutKeys := &SESNotifier{
		FromEmail:   "sender@example.com",
		TargetEmail: "admin@example.com",
		Region:      "us-east-1",
		AuthType:    SESAuthTypeKeys,
	}
	require.Error(t, notifierWithoutKeys.Validate(encryptor))

	validKeysNotifier := &SESNotifier{
		FromEmail:   "sender@example.com",
		TargetEmail: "admin@example.com",
		Region:      "us-east-1",
		AuthType:    SESAuthTypeKeys,
		AccessKey:   "AKIAIOSFODNN7EXAMPLE",
		SecretKey:   "secret",
	}
	require.NoError(t, validKeysNotifier.Validate(encryptor))
}

func Test_SESNotifier_HideSensitiveData_ClearsKeysAndExternalID(t *testing.T) {
	notifier := &SESNotifier{
		AccessKey:  "access",
		SecretKey:  "secret",
		ExternalID: "ext-id",
	}

	notifier.HideSensitiveData()

	assert.Empty(t, notifier.AccessKey)
	assert.Empty(t, notifier.SecretKey)
	assert.Empty(t, notifier.ExternalID)
}

func Test_SESNotifier_Update(t *testing.T) {
	notifier := &SESNotifier{
		FromEmail:   "old@example.com",
		TargetEmail: "old-target@example.com",
		Region:      "us-east-1",
		AuthType:    SESAuthTypeKeys,
		AccessKey:   "old-key",
		SecretKey:   "old-secret",
	}

	incoming := &SESNotifier{
		FromEmail:   "new@example.com",
		TargetEmail: "new-target@example.com",
		Region:      "us-west-2",
		AuthType:    SESAuthTypeIAMRole,
		RoleARN:     "arn:aws:iam::123456789012:role/SESRole",
	}

	notifier.Update(incoming)

	assert.Equal(t, "new@example.com", notifier.FromEmail)
	assert.Equal(t, "new-target@example.com", notifier.TargetEmail)
	assert.Equal(t, "us-west-2", notifier.Region)
	assert.Equal(t, SESAuthTypeIAMRole, notifier.AuthType)
	assert.Equal(t, "arn:aws:iam::123456789012:role/SESRole", notifier.RoleARN)
	// Preserves old keys when incoming does not provide new keys
	assert.Equal(t, "old-key", notifier.AccessKey)
	assert.Equal(t, "old-secret", notifier.SecretKey)
}
