package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf16"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/redaction"
)

// maxTextUTF16Units is the longest text the Bot API accepts in one message.
const maxTextUTF16Units = 4096

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// OfficialBotAPIURL is the Bot API of a notifier that names none.
	OfficialBotAPIURL string
}

type Provider struct {
	repository        *settingsRepository
	fieldEncryptor    *crypto.FieldEncryptor
	officialBotAPIURL string
	directClient      *http.Client
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:        deps.Repositories.settings,
		fieldEncryptor:    deps.FieldEncryptor,
		officialBotAPIURL: deps.OfficialBotAPIURL,
		directClient:      providers.NewHTTPClient(nil),
	}
}

func (p *Provider) GetType() notification.NotifierType {
	return notification.NotifierTypeTelegram
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	telegramSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if telegramSettings.BotToken == "" {
		return providers.NewInvalidSettingError("botToken", "is required")
	}

	if telegramSettings.TargetChatID == "" {
		return providers.NewInvalidSettingError("targetChatId", "is required")
	}

	if telegramSettings.IsProxyEnabled {
		if telegramSettings.ProxyURL == "" {
			return providers.NewInvalidSettingError("proxyUrl", "is required while the proxy is on")
		}

		proxyURL, err := p.fieldEncryptor.Decrypt(telegramSettings.ProxyURL)
		if err != nil {
			return fmt.Errorf("decrypt the proxy URL: %w", err)
		}

		if _, err := parseProxyURL(proxyURL); err != nil {
			return providers.NewInvalidSettingError("proxyUrl", err.Error())
		}
	}

	if telegramSettings.BotAPIURL != "" {
		if reason, isValid := validateBotAPIURL(telegramSettings.BotAPIURL); !isValid {
			return providers.NewInvalidSettingError("botApiUrl", reason)
		}
	}

	return nil
}

// MergeStoredSecrets keeps the stored proxy URL only while the proxy stays on, so turning it off
// clears the URL.
func (p *Provider) MergeStoredSecrets(merge providers.SecretMerge) (providers.Settings, error) {
	incomingSettings, err := providers.CastSettings[Settings](merge.Incoming)
	if err != nil {
		return nil, err
	}

	storedSettings, err := providers.CastSettings[Settings](merge.Stored)
	if err != nil {
		return nil, err
	}

	// The stored proxy URL is encrypted, and it is never compared with one in the request, so that an
	// answer cannot tell a caller whether a guessed URL matches it.
	isAddressChanged := incomingSettings.BotAPIURL != storedSettings.BotAPIURL ||
		incomingSettings.IsProxyEnabled != storedSettings.IsProxyEnabled ||
		incomingSettings.ProxyURL != ""

	if incomingSettings.BotToken == "" && !isAddressChanged {
		incomingSettings.BotToken = storedSettings.BotToken
	}

	if !incomingSettings.IsProxyEnabled {
		incomingSettings.ProxyURL = ""
	} else if incomingSettings.ProxyURL == "" {
		incomingSettings.ProxyURL = storedSettings.ProxyURL
	}

	return incomingSettings, nil
}

func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	telegramSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	botToken, err := p.fieldEncryptor.Decrypt(telegramSettings.BotToken)
	if err != nil {
		return fmt.Errorf("decrypt the bot token: %w", err)
	}

	botAPIURL := p.officialBotAPIURL
	if telegramSettings.BotAPIURL != "" {
		botAPIURL = telegramSettings.BotAPIURL
	}

	form := url.Values{}
	form.Set("chat_id", telegramSettings.TargetChatID)
	form.Set("text", buildText(message))

	if telegramSettings.ThreadID != nil && *telegramSettings.ThreadID != 0 {
		form.Set("message_thread_id", strconv.FormatInt(*telegramSettings.ThreadID, 10))
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, buildSendMessageURL(botAPIURL, botToken), strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("create the request: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client, err := p.getClient(telegramSettings)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send the telegram message: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)

		return fmt.Errorf(
			"telegram API returned non-OK status: %s. Error: %s",
			response.Status,
			redaction.RedactSecretsInText(string(responseBody), botToken),
		)
	}

	return nil
}

func (p *Provider) getClient(telegramSettings Settings) (*http.Client, error) {
	if !telegramSettings.IsProxyEnabled {
		return p.directClient, nil
	}

	rawProxyURL, err := p.fieldEncryptor.Decrypt(telegramSettings.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("decrypt the proxy URL: %w", err)
	}

	proxyURL, err := parseProxyURL(rawProxyURL)
	if err != nil {
		return nil, err
	}

	transport, err := newProxyTransport(proxyURL)
	if err != nil {
		return nil, err
	}

	return providers.NewHTTPClient(transport), nil
}

// buildText cuts the text at the Bot API limit, which counts UTF-16 code units, so a character
// outside the Basic Multilingual Plane, such as most emoji, takes two of them.
func buildText(message notification.Notification) string {
	text := message.Heading
	if message.Message != "" {
		text = fmt.Sprintf("%s\n\n%s", message.Heading, message.Message)
	}

	textUTF16Units := 0
	for byteOffset, textRune := range text {
		textUTF16Units += utf16.RuneLen(textRune)
		if textUTF16Units > maxTextUTF16Units {
			return text[:byteOffset]
		}
	}

	return text
}
