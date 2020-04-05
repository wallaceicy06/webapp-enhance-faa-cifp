package auth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"

	oidc "github.com/coreos/go-oidc"
)

type OIDC struct {
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func NewOIDC(ctx context.Context, clientID, clientSecret string) (*OIDC, error) {
	a := &OIDC{}

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("could not create OIDC provider: %v", err)
	}

	a.verifier = provider.Verifier(&oidc.Config{ClientID: clientID})

	// Configure an OpenID Connect aware OAuth2 client.
	a.config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		// RedirectURL:  "https://",

		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return a, nil
}

// VerifyEmail returns the email from the corresponding OIDC code. If the
// code is unable to be verified, then an error is returned.
func (o *OIDC) VerifyEmail(ctx context.Context, code string) (string, error) {
	token, err := o.config.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("could not exchange code: %v", err)
	}
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return "", errors.New("missing extra token")
	}

	// Parse and verify ID Token payload.
	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", fmt.Errorf("unable to verify token: %v", err)
	}

	// Extract custom claims
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("could not extract claims: %v", err)
	}
	return claims.Email, nil
}
