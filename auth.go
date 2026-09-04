package weheat

import (
	"context"
	"sync"

	"golang.org/x/oauth2"
)

// Keycloak endpoints for the WeHeat realm.
const (
	AuthURL  = "https://auth.weheat.nl/auth/realms/Weheat/protocol/openid-connect/auth"
	TokenURL = "https://auth.weheat.nl/auth/realms/Weheat/protocol/openid-connect/token"
)

// CommunityClientID is the public Keycloak client WeHeat exposes for third-party tools.
// It has no secret and supports the password grant. The confidential HomeAssistantAPI client
// (ID and secret published in the WeHeat knowledge base) also works; pass its secret.
const CommunityClientID = "WeheatCommunityAPI"

// Scopes requested from Keycloak. offline_access yields a refresh token.
var Scopes = []string{"openid", "offline_access"}

// OAuthConfig returns the oauth2 config for the WeHeat Keycloak realm.
// clientSecret is empty for public clients such as CommunityClientID.
func OAuthConfig(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     oauth2.Endpoint{AuthURL: AuthURL, TokenURL: TokenURL, AuthStyle: oauth2.AuthStyleInParams},
		Scopes:       Scopes,
	}
}

// PasswordTokenSource logs in with the password grant and returns a self-refreshing token source.
// Community tools use this grant with CommunityClientID.
func PasswordTokenSource(ctx context.Context, cfg *oauth2.Config, username, password string) (oauth2.TokenSource, error) {
	tok, err := cfg.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return nil, err
	}
	return cfg.TokenSource(ctx, tok), nil
}

// RefreshTokenSource returns a self-refreshing token source seeded with a refresh token.
func RefreshTokenSource(ctx context.Context, cfg *oauth2.Config, refreshToken string) oauth2.TokenSource {
	return cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
}

// NotifyTokenSource calls onRefresh whenever ts hands out a token whose refresh token changed.
// Keycloak rotates refresh tokens, so callers that want to survive a restart must persist the new one.
func NotifyTokenSource(ts oauth2.TokenSource, onRefresh func(*oauth2.Token)) oauth2.TokenSource {
	return &notifyingSource{src: ts, onRefresh: onRefresh}
}

type notifyingSource struct {
	src       oauth2.TokenSource
	onRefresh func(*oauth2.Token)

	mu   sync.Mutex
	last string
}

func (n *notifyingSource) Token() (*oauth2.Token, error) {
	tok, err := n.src.Token()
	if err != nil {
		return nil, err
	}
	n.mu.Lock()
	changed := tok.RefreshToken != "" && tok.RefreshToken != n.last
	n.last = tok.RefreshToken
	n.mu.Unlock()
	if changed {
		n.onRefresh(tok)
	}
	return tok, nil
}
