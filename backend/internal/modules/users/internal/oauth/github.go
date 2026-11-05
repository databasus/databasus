package oauth

import (
	"context"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
)

type GitHubSettings struct {
	// ClientID and ClientSecret are empty when the instance offers no GitHub sign-in.
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserURL      string
	EmailsURL    string
}

type GitHub struct {
	settings GitHubSettings
	exchange tokenExchange
}

func NewGitHub(settings GitHubSettings) *GitHub {
	return &GitHub{
		settings: settings,
		exchange: newTokenExchange(&oauth2.Config{
			ClientID:     settings.ClientID,
			ClientSecret: settings.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:       settings.AuthURL,
				DeviceAuthURL: "",
				TokenURL:      settings.TokenURL,
				AuthStyle:     oauth2.AuthStyleAutoDetect,
			},
			RedirectURL: "",
			Scopes:      []string{"user:email"},
		}),
	}
}

func (g *GitHub) GetProvider() Provider {
	return ProviderGitHub
}

func (g *GitHub) IsConfigured() bool {
	return g.settings.ClientID != "" && g.settings.ClientSecret != ""
}

// FetchAccount takes the public address of the profile, which GitHub shows only once it is
// verified, and otherwise the primary verified address, then any verified one. An unverified
// address never signs anybody in: it would let a GitHub account take over the user who owns that
// address here.
func (g *GitHub) FetchAccount(ctx context.Context, authorization Authorization) (Account, error) {
	client, err := g.exchange.authorize(ctx, authorization)
	if err != nil {
		return Account{}, err
	}

	var profile struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Login string `json:"login"`
	}

	err = getJSON(ctx, client, g.settings.UserURL, &profile)
	if err != nil {
		return Account{}, err
	}

	email := profile.Email
	if email == "" {
		email, err = g.fetchVerifiedEmail(ctx, client)
		if err != nil {
			return Account{}, err
		}
	}

	name := profile.Name
	if name == "" {
		name = profile.Login
	}

	return Account{ID: strconv.FormatInt(profile.ID, 10), Email: email, Name: name}, nil
}

func (g *GitHub) fetchVerifiedEmail(ctx context.Context, client *http.Client) (string, error) {
	var listedEmails []struct {
		Email      string `json:"email"`
		IsPrimary  bool   `json:"primary"`
		IsVerified bool   `json:"verified"`
	}

	err := getJSON(ctx, client, g.settings.EmailsURL, &listedEmails)
	if err != nil {
		return "", err
	}

	for _, listedEmail := range listedEmails {
		if listedEmail.IsPrimary && listedEmail.IsVerified {
			return listedEmail.Email, nil
		}
	}

	for _, listedEmail := range listedEmails {
		if listedEmail.IsVerified {
			return listedEmail.Email, nil
		}
	}

	return "", ErrNoVerifiedEmail
}
