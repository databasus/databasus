package telegram_notifier

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/encryption"
)

type TelegramNotifier struct {
	NotifierID     uuid.UUID `json:"notifierId"     gorm:"primaryKey;column:notifier_id"`
	BotToken       string    `json:"botToken"       gorm:"not null;column:bot_token"`
	TargetChatID   string    `json:"targetChatId"   gorm:"not null;column:target_chat_id"`
	ThreadID       *int64    `json:"threadId"       gorm:"column:thread_id"`
	IsProxyEnabled bool      `json:"isProxyEnabled" gorm:"column:is_proxy_enabled;type:boolean;not null;default:false"`
	ProxyURL       string    `json:"proxyUrl"       gorm:"column:proxy_url;type:text"`
	BotAPIURL      string    `json:"botApiUrl"      gorm:"column:bot_api_url;type:text"`
}

func (t *TelegramNotifier) TableName() string {
	return "telegram_notifiers"
}

func (t *TelegramNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if t.BotToken == "" {
		return errors.New("bot token is required")
	}

	if t.TargetChatID == "" {
		return errors.New("target chat ID is required")
	}

	if t.IsProxyEnabled {
		if t.ProxyURL == "" {
			return errors.New("proxy URL is required")
		}

		proxyURL, err := encryptor.Decrypt(t.ProxyURL)
		if err != nil {
			return fmt.Errorf("failed to decrypt proxy URL: %w", err)
		}

		if _, err := parseProxyURL(proxyURL); err != nil {
			return err
		}
	}

	if t.BotAPIURL != "" {
		if err := validateBotAPIURL(t.BotAPIURL); err != nil {
			return err
		}
	}

	return nil
}

func (t *TelegramNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	botToken, err := encryptor.Decrypt(t.BotToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt bot token: %w", err)
	}

	sendMessageURL, err := t.getSendMessageURL(botToken)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("chat_id", t.TargetChatID)
	data.Set("text", buildAndTruncateText(notification))
	data.Set("parse_mode", "HTML")

	if t.ThreadID != nil && *t.ThreadID != 0 {
		data.Set("message_thread_id", strconv.FormatInt(*t.ThreadID, 10))
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		sendMessageURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client, err := t.buildHTTPClient(encryptor)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", notifier_models.ErrorWithoutWebhookURLCredentials(err))
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"telegram API returned non-OK status: %s. Error: %s",
			resp.Status,
			notifier_models.RedactSecretInText(string(bodyBytes), botToken),
		)
	}

	return nil
}

func buildAndTruncateText(notification notifier_models.Notification) string {
	text := notification.Heading
	if notification.Message != "" {
		text = fmt.Sprintf("%s\n\n%s", notification.Heading, notification.Message)
	}

	textRunes := []rune(text)
	if len(textRunes) > maxTextRunes {
		return string(textRunes[:maxTextRunes])
	}

	return text
}

func (t *TelegramNotifier) HideSensitiveData() {
	t.BotToken = ""
	t.ProxyURL = ""
}

func (t *TelegramNotifier) Update(incoming *TelegramNotifier) {
	t.TargetChatID = incoming.TargetChatID
	t.ThreadID = incoming.ThreadID
	t.IsProxyEnabled = incoming.IsProxyEnabled
	t.BotAPIURL = incoming.BotAPIURL

	if !incoming.IsProxyEnabled {
		t.ProxyURL = ""
	} else if incoming.ProxyURL != "" {
		t.ProxyURL = incoming.ProxyURL
	}

	if incoming.BotToken != "" {
		t.BotToken = incoming.BotToken
	}
}

func (t *TelegramNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if t.BotToken != "" {
		encrypted, err := encryptor.Encrypt(t.BotToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt bot token: %w", err)
		}
		t.BotToken = encrypted
	}

	if !t.IsProxyEnabled {
		t.ProxyURL = ""
	} else if t.ProxyURL != "" {
		encrypted, err := encryptor.Encrypt(t.ProxyURL)
		if err != nil {
			return fmt.Errorf("failed to encrypt proxy URL: %w", err)
		}
		t.ProxyURL = encrypted
	}

	return nil
}
