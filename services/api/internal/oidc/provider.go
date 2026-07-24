package oidc

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config holds OIDC client settings loaded from ESCALITE_OIDC_* env vars.
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// UserInfo is the authenticated identity returned after a successful OIDC exchange.
type UserInfo struct {
	Subject string
	Email   string
}

// Provider wraps go-oidc and oauth2 for the authorization code flow.
type Provider struct {
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// NewProvider discovers the IdP and returns a ready OIDC client.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	issuerURL := strings.TrimSpace(cfg.IssuerURL)
	if issuerURL == "" {
		return nil, fmt.Errorf("oidc issuer URL is required")
	}

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	return &Provider{
		oauth2Config: oauth2Config,
		verifier:     verifier,
	}, nil
}

// AuthCodeURL returns the IdP authorization URL for the given CSRF state.
func (p *Provider) AuthCodeURL(state string) string {
	return p.oauth2Config.AuthCodeURL(state)
}

// Exchange validates the authorization code and returns verified user identity claims.
func (p *Provider) Exchange(ctx context.Context, code string) (UserInfo, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return UserInfo{}, fmt.Errorf("exchange authorization code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return UserInfo{}, fmt.Errorf("id_token missing from token response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return UserInfo{}, fmt.Errorf("verify id token: %w", err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return UserInfo{}, fmt.Errorf("parse id token claims: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return UserInfo{}, fmt.Errorf("email claim is required")
	}

	return UserInfo{
		Subject: idToken.Subject,
		Email:   email,
	}, nil
}
