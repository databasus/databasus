// Package turnstile asks Cloudflare Turnstile whether the answer to a human-verification
// challenge is valid.
package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"databasus-backend/internal/lib/httpserver"
)

const verificationTimeout = 10 * time.Second

type Deps struct {
	// SecretKey is empty when the instance has no challenge, and then every request passes.
	SecretKey string
	VerifyURL string
	Logger    *slog.Logger
}

type Client struct {
	secretKey  string
	verifyURL  string
	httpClient *http.Client
	logger     *slog.Logger
}

func New(deps Deps) *Client {
	return &Client{
		secretKey:  deps.SecretKey,
		verifyURL:  deps.VerifyURL,
		httpClient: &http.Client{Timeout: verificationTimeout},
		logger:     deps.Logger,
	}
}

// Verify refuses a missing or rejected answer with human_verification_failed. It sends the client
// IP that httpserver.RememberClientIP put into ctx.
func (c *Client) Verify(ctx context.Context, answer *string) error {
	if c.secretKey == "" {
		return nil
	}

	if answer == nil || *answer == "" {
		return ErrVerificationFailed
	}

	isValid, err := c.ask(ctx, *answer)
	if err != nil {
		c.logger.WarnContext(ctx, "human verification could not be checked", slog.Any("error", err))

		return ErrVerificationFailed
	}

	if !isValid {
		return ErrVerificationFailed
	}

	return nil
}

func (c *Client) ask(ctx context.Context, answer string) (bool, error) {
	form := url.Values{}
	form.Set("secret", c.secretKey)
	form.Set("response", answer)
	form.Set("remoteip", httpserver.GetClientIP(ctx))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, fmt.Errorf("build the verification request: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("send the verification request: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: status %d", ErrUnexpectedAnswer, response.StatusCode)
	}

	var verification struct {
		IsSuccess bool `json:"success"`
	}

	err = json.NewDecoder(response.Body).Decode(&verification)
	if err != nil {
		return false, fmt.Errorf("decode the verification answer: %w", err)
	}

	return verification.IsSuccess, nil
}
