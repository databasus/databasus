package ses_notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2_types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/aws_auth"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/smtp_transport"
)

type SESAuthType string

const (
	SESAuthTypeKeys    SESAuthType = "KEYS"
	SESAuthTypeIAMRole SESAuthType = "IAM_ROLE"
)

type SESNotifier struct {
	NotifierID  uuid.UUID   `json:"notifierId"  gorm:"primaryKey;type:uuid;column:notifier_id"`
	FromEmail   string      `json:"fromEmail"   gorm:"not null;type:text;column:from_email"`
	TargetEmail string      `json:"targetEmail" gorm:"not null;type:text;column:target_email"`
	Region      string      `json:"region"      gorm:"not null;type:text;column:region"`
	AuthType    SESAuthType `json:"authType"    gorm:"not null;type:text;column:auth_type;default:'IAM_ROLE'"`
	AccessKey   string      `json:"accessKey"   gorm:"type:text;column:access_key;default:''"`
	SecretKey   string      `json:"secretKey"   gorm:"type:text;column:secret_key;default:''"`
	RoleARN     string      `json:"roleArn"     gorm:"type:text;column:role_arn;default:''"`
	ExternalID  string      `json:"externalId"  gorm:"type:text;column:external_id;default:''"`
}

func (s *SESNotifier) TableName() string {
	return "ses_notifiers"
}

func (s *SESNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if s.FromEmail == "" {
		return errors.New("from email is required")
	}

	if _, err := smtp_transport.ParseSender(s.FromEmail); err != nil {
		return fmt.Errorf("invalid from email: %w", err)
	}

	if s.TargetEmail == "" {
		return errors.New("target email is required")
	}

	if _, err := smtp_transport.ParseRecipient(s.TargetEmail); err != nil {
		return fmt.Errorf("invalid target email: %w", err)
	}

	if s.Region == "" {
		return errors.New("AWS region is required")
	}

	authType := s.AuthType
	if authType == "" {
		authType = SESAuthTypeIAMRole
	}

	switch authType {
	case SESAuthTypeKeys:
		if s.AccessKey == "" {
			return errors.New("AWS access key is required for KEYS authentication")
		}

		if s.SecretKey == "" {
			return errors.New("AWS secret key is required for KEYS authentication")
		}
	case SESAuthTypeIAMRole:
		// RoleARN and ExternalID are optional in IAM role mode (uses ambient credentials when empty)
	default:
		return fmt.Errorf("unknown SES auth type: %s", authType)
	}

	return nil
}

func (s *SESNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	var accessKey, secretKey, externalID string
	var err error

	if s.AccessKey != "" {
		accessKey, err = encryptor.Decrypt(s.AccessKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SES access key: %w", err)
		}
	}

	if s.SecretKey != "" {
		secretKey, err = encryptor.Decrypt(s.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SES secret key: %w", err)
		}
	}

	if s.ExternalID != "" {
		externalID, err = encryptor.Decrypt(s.ExternalID)
		if err != nil {
			return fmt.Errorf("failed to decrypt SES external ID: %w", err)
		}
	}

	authType := s.AuthType
	if authType == "" {
		authType = SESAuthTypeIAMRole
	}

	awsConfig, err := aws_auth.BuildAWSConfig(context.Background(), aws_auth.Config{
		AuthType:   aws_auth.AuthType(authType),
		Region:     s.Region,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		RoleARN:    s.RoleARN,
		ExternalID: externalID,
	})
	if err != nil {
		return fmt.Errorf("failed to configure AWS for SES: %w", err)
	}

	client := sesv2.NewFromConfig(awsConfig)

	emailInput := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.FromEmail),
		Destination: &sesv2_types.Destination{
			ToAddresses: []string{s.TargetEmail},
		},
		Content: &sesv2_types.EmailContent{
			Simple: &sesv2_types.Message{
				Subject: &sesv2_types.Content{
					Data:    aws.String(notification.Heading),
					Charset: aws.String("UTF-8"),
				},
				Body: &sesv2_types.Body{
					Html: &sesv2_types.Content{
						Data:    aws.String(notification.Message),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	_, err = client.SendEmail(context.Background(), emailInput)
	if err != nil {
		logger.ErrorContext(context.Background(), "failed to send email via SES",
			"from", s.FromEmail, "target", s.TargetEmail, "region", s.Region, "error", err)

		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	return nil
}

func (s *SESNotifier) HideSensitiveData() {
	s.AccessKey = ""
	s.SecretKey = ""
	s.ExternalID = ""
}

func (s *SESNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.AccessKey != "" {
		s.AccessKey, err = encryptor.Encrypt(s.AccessKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt SES access key: %w", err)
		}
	}

	if s.SecretKey != "" {
		s.SecretKey, err = encryptor.Encrypt(s.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt SES secret key: %w", err)
		}
	}

	if s.ExternalID != "" {
		s.ExternalID, err = encryptor.Encrypt(s.ExternalID)
		if err != nil {
			return fmt.Errorf("failed to encrypt SES external ID: %w", err)
		}
	}

	return nil
}

func (s *SESNotifier) Update(incoming *SESNotifier) {
	s.FromEmail = incoming.FromEmail
	s.TargetEmail = incoming.TargetEmail
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
