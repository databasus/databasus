package oauth

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type tokenExchange struct {
	config     *oauth2.Config
	httpClient *http.Client
}

func newTokenExchange(config *oauth2.Config) tokenExchange {
	return tokenExchange{config: config, httpClient: &http.Client{Timeout: requestTimeout}}
}

// authorize exchanges the code for a token and returns a client that sends it.
func (e tokenExchange) authorize(ctx context.Context, authorization Authorization) (*http.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, e.httpClient)

	config := *e.config
	config.RedirectURL = authorization.RedirectURI

	token, err := config.Exchange(ctx, authorization.Code)
	if err != nil {
		return nil, fmt.Errorf("exchange the authorization code: %w", err)
	}

	return config.Client(ctx, token), nil
}
