package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

const (
	githubAuthURL  = "https://github.com/login/oauth/authorize"
	githubTokenURL = "https://github.com/login/oauth/access_token"
)

type githubProvider struct {
	clientID     string
	clientSecret string
	// apiBase is the GitHub API root, overridden by tests.
	apiBase string
	// endpoint is the OAuth endpoint pair, overridden by tests.
	endpoint oauth2.Endpoint
	client   *http.Client
}

// NewGithubProvider builds the GitHub identity provider for an OAuth App.
func NewGithubProvider(clientID, clientSecret string) *githubProvider {
	return &githubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		apiBase:      "https://api.github.com",
		endpoint:     oauth2.Endpoint{AuthURL: githubAuthURL, TokenURL: githubTokenURL},
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *githubProvider) ID() string          { return "github" }
func (g *githubProvider) DisplayName() string { return "GitHub" }
func (g *githubProvider) Icon() string        { return "mdi:github" }

// oauth2Config ignores callbackURI. GitHub lets a request omit redirect_uri and
// falls back to the URL registered on the OAuth app, which is one less thing to
// keep in sync and one less thing a forged Host header can influence.
func (g *githubProvider) oauth2Config(callbackURI string) (*oauth2.Config, error) {
	return &oauth2.Config{
		ClientID:     g.clientID,
		ClientSecret: g.clientSecret,
		Endpoint:     g.endpoint,
		// user:email is what makes /user/emails readable; without it the primary
		// address stays hidden and the profile has no verified one to show.
		Scopes: []string{"read:user", "user:email"},
		// RedirectURL is intentionally empty. See LoginHandler.
	}, nil
}

// match resolves on the GitHub login rather than the email. A login is stable
// and always present; an email can be private, unverified, or changed, so
// matching on one would either lock people out or key the account on something
// its owner can move.
func (g *githubProvider) match(users userLookup, id externalIdentity) (User, bool) {
	if id.Login == "" {
		return User{}, false
	}

	return users.byGithubLogin(id.Login)
}

type githubUser struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *githubProvider) identity(ctx context.Context, token *oauth2.Token) (externalIdentity, error) {
	// The error is unreachable: GitHub's endpoints are constants.
	config, _ := g.oauth2Config("")
	client := config.Client(ctx, token)
	if g.client != nil {
		client.Timeout = g.client.Timeout
	}

	var profile githubUser
	if err := g.get(ctx, client, "/user", &profile); err != nil {
		return externalIdentity{}, err
	}

	if profile.Login == "" {
		return externalIdentity{}, fmt.Errorf("github returned a profile with no login")
	}

	identity := externalIdentity{
		Sub:   fmt.Sprintf("%d", profile.ID),
		Login: profile.Login,
		Name:  profile.Name,
	}

	// Display only, and only ever the primary verified address. The profile's
	// public email is self-asserted and often empty, so it is never used.
	var emails []githubEmail
	if err := g.get(ctx, client, "/user/emails", &emails); err == nil {
		for _, e := range emails {
			if e.Primary && e.Verified {
				identity.Email = e.Email
				break
			}
		}
	}

	return identity, nil
}

func (g *githubProvider) get(ctx context.Context, client *http.Client, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiBase+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github %s returned %s", path, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
