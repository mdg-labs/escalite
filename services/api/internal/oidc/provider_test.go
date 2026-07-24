package oidc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/oidc"
	"github.com/mdg-labs/escalite/services/api/internal/oidc/testutil"
)

func TestProviderExchangeReturnsVerifiedEmail(t *testing.T) {
	const (
		clientID     = "test-client"
		clientSecret = "test-secret"
		redirectURL  = "http://localhost/api/v1/auth/oidc/callback"
	)

	mock := testutil.NewMockServer(t, clientID, clientSecret, redirectURL, "oidc-user@example.com")
	defer mock.Close()

	provider, err := oidc.NewProvider(context.Background(), oidc.Config{
		IssuerURL:    mock.URL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
	})
	require.NoError(t, err)

	userInfo, err := provider.Exchange(context.Background(), "test-auth-code")
	require.NoError(t, err)
	require.Equal(t, "oidc-user@example.com", userInfo.Email)
	require.Equal(t, "oidc-subject-oidc-user@example.com", userInfo.Subject)
}
