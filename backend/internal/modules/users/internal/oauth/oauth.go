// Package oauth exchanges the authorization code of GitHub or Google for the provider's account:
// its ID, its verified email address and its name.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const requestTimeout = 10 * time.Second

type Provider string

const (
	ProviderGitHub Provider = "GitHub"
	ProviderGoogle Provider = "Google"
)

// Authorization is what the provider's authorization page handed back to the browser.
type Authorization struct {
	Code        string
	RedirectURI string
}

// Account is the provider's account behind an authorization.
type Account struct {
	ID    string
	Email string
	Name  string
}

func getJSON(ctx context.Context, client *http.Client, url string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build the request to %s: %w", url, err)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send the request to %s: %w", url, err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s answered status %d", ErrUnexpectedAnswer, url, response.StatusCode)
	}

	err = json.NewDecoder(response.Body).Decode(destination)
	if err != nil {
		return fmt.Errorf("decode the answer of %s: %w", url, err)
	}

	return nil
}
