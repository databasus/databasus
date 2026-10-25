package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// googleFallbackName names a user whose Google account has no name.
const googleFallbackName = "User"

type GoogleSettings struct {
	// ClientID and ClientSecret are empty when the instance offers no Google sign-in.
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

type Google struct {
	settings GoogleSettings
	exchange tokenExchange
}

func NewGoogle(settings GoogleSettings) *Google {
	return &Google{
		settings: settings,
		exchange: newTokenExchange(&oauth2.Config{
			ClientID:     settings.ClientID,
			ClientSecret: settings.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:       settings.AuthURL,
				DeviceAuthURL: "",
				TokenURL:      settings.TokenURL,
				AuthStyle:     oauth2.AuthStyleInParams,
			},
			RedirectURL: "",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		}),
	}
}

func (g *Google) GetProvider() Provider {
	return ProviderGoogle
}

func (g *Google) IsConfigured() bool {
	return g.settings.ClientID != "" && g.settings.ClientSecret != ""
}

// FetchAccount refuses an unverified address, which would let a Google account take over the user
// who owns that address here.
func (g *Google) FetchAccount(ctx context.Context, authorization Authorization) (Account, error) {
	client, err := g.exchange.authorize(ctx, authorization)
	if err != nil {
		return Account{}, err
	}

	var userInfo struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		IsVerified bool   `json:"verified_email"`
		Name       string `json:"name"`
	}

	err = getJSON(ctx, client, g.settings.UserInfoURL, &userInfo)
	if err != nil {
		return Account{}, err
	}

	if userInfo.Email == "" || !userInfo.IsVerified {
		return Account{}, ErrNoVerifiedEmail
	}

	name := userInfo.Name
	if name == "" {
		name = googleFallbackName
	}

	return Account{ID: userInfo.ID, Email: userInfo.Email, Name: name}, nil
}
