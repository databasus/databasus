package sns_notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/aws_auth"
	"databasus-backend/internal/util/encryption"
)

type SNSAuthType string

const (
	SNSAuthTypeKeys    SNSAuthType = "KEYS"
	SNSAuthTypeIAMRole SNSAuthType = "IAM_ROLE"
)

const maxSNSSubjectLength = 100

type SNSNotifier struct {
	NotifierID uuid.UUID   `json:"notifierId" gorm:"primaryKey;type:uuid;column:notifier_id"`
	TopicARN   string      `json:"topicArn"   gorm:"not null;type:text;column:topic_arn"`
	Region     string      `json:"region"     gorm:"not null;type:text;column:region"`
	AuthType   SNSAuthType `json:"authType"   gorm:"not null;type:text;column:auth_type;default:'KEYS'"`
	AccessKey  string      `json:"accessKey"  gorm:"type:text;column:access_key;default:''"`
	SecretKey  string      `json:"secretKey"  gorm:"type:text;column:secret_key;default:''"`
	RoleARN    string      `json:"roleArn"    gorm:"type:text;column:role_arn;default:''"`
	ExternalID string      `json:"externalId" gorm:"type:text;column:external_id;default:''"`
}

func (s *SNSNotifier) TableName() string {
	return "sns_notifiers"
}

func (s *SNSNotifier) FillDefaults() {
	if s.Region == "" && s.TopicARN != "" {
		s.Region = extractRegionFromTopicARN(s.TopicARN)
	}
}

func extractRegionFromTopicARN(topicARN string) string {
	parts := strings.Split(topicARN, ":")
	if len(parts) >= 6 && parts[0] == "arn" && strings.HasPrefix(parts[1], "aws") && parts[2] == "sns" {
		return parts[3]
	}

	return ""
}

func (s *SNSNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if s.TopicARN == "" {
		return errors.New("SNS topic ARN is required")
	}

	if !strings.HasPrefix(s.TopicARN, "arn:aws") || !strings.Contains(s.TopicARN, ":sns:") {
		return errors.New("invalid SNS topic ARN format (must start with arn:aws and contain :sns:)")
	}

	if s.Region == "" {
		s.Region = extractRegionFromTopicARN(s.TopicARN)
	}

	if s.Region == "" {
		return errors.New("AWS region is required")
	}

	authType := s.AuthType
	if authType == "" {
		authType = SNSAuthTypeKeys
	}

	switch authType {
	case SNSAuthTypeKeys:
		if s.AccessKey == "" {
			return errors.New("AWS access key is required for KEYS authentication")
		}

		if s.SecretKey == "" {
			return errors.New("AWS secret key is required for KEYS authentication")
		}
	case SNSAuthTypeIAMRole:
		// RoleARN and ExternalID are optional in IAM role mode (uses ambient credentials when empty)
	default:
		return fmt.Errorf("unknown SNS auth type: %s", authType)
	}

	return nil
}

func (s *SNSNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	var accessKey, secretKey, externalID string
	var err error

	if s.AccessKey != "" {
		accessKey, err = encryptor.Decrypt(s.AccessKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SNS access key: %w", err)
		}
	}

	if s.SecretKey != "" {
		secretKey, err = encryptor.Decrypt(s.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SNS secret key: %w", err)
		}
	}

	if s.ExternalID != "" {
		externalID, err = encryptor.Decrypt(s.ExternalID)
		if err != nil {
			return fmt.Errorf("failed to decrypt SNS external ID: %w", err)
		}
	}

	authType := s.AuthType
	if authType == "" {
		authType = SNSAuthTypeKeys
	}

	region := s.Region
	if region == "" {
		region = extractRegionFromTopicARN(s.TopicARN)
	}

	awsConfig, err := aws_auth.BuildAWSConfig(context.Background(), aws_auth.Config{
		AuthType:   aws_auth.AuthType(authType),
		Region:     region,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		RoleARN:    s.RoleARN,
		ExternalID: externalID,
	})
	if err != nil {
		return fmt.Errorf("failed to configure AWS for SNS: %w", err)
	}

	client := sns.NewFromConfig(awsConfig)

	subject := formatSNSSubject(notification.Heading)

	messageContent := notification.Heading
	if notification.Message != "" {
		messageContent += "\n\n" + notification.Message
	}

	publishInput := &sns.PublishInput{
		TopicArn: aws.String(s.TopicARN),
		Subject:  aws.String(subject),
		Message:  aws.String(messageContent),
	}

	_, err = client.Publish(context.Background(), publishInput)
	if err != nil {
		logger.ErrorContext(context.Background(), "failed to publish notification to SNS",
			"topic_arn", s.TopicARN, "region", region, "error", err)

		return fmt.Errorf("failed to publish to SNS topic: %w", err)
	}

	return nil
}

func formatSNSSubject(rawHeading string) string {
	cleaned := strings.ReplaceAll(rawHeading, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.TrimSpace(cleaned)

	if len(cleaned) > maxSNSSubjectLength {
		return cleaned[:maxSNSSubjectLength-3] + "..."
	}

	if cleaned == "" {
		return "Databasus Notification"
	}

	return cleaned
}

func (s *SNSNotifier) HideSensitiveData() {
	s.AccessKey = ""
	s.SecretKey = ""
	s.ExternalID = ""
}

func (s *SNSNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.AccessKey != "" {
		s.AccessKey, err = encryptor.Encrypt(s.AccessKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt SNS access key: %w", err)
		}
	}

	if s.SecretKey != "" {
		s.SecretKey, err = encryptor.Encrypt(s.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt SNS secret key: %w", err)
		}
	}

	if s.ExternalID != "" {
		s.ExternalID, err = encryptor.Encrypt(s.ExternalID)
		if err != nil {
			return fmt.Errorf("failed to encrypt SNS external ID: %w", err)
		}
	}

	return nil
}

func (s *SNSNotifier) Update(incoming *SNSNotifier) {
	s.TopicARN = incoming.TopicARN
	s.Region = incoming.Region
	s.AuthType = incoming.AuthType
	s.RoleARN = incoming.RoleARN

	if incoming.AccessKey != "" {
		s.AccessKey = incoming.AccessKey
	}

	if incoming.SecretKey != "" {
		s.SecretKey = incoming.SecretKey
	}

	if incoming.ExternalID != "" {
		s.ExternalID = incoming.ExternalID
	}
}
