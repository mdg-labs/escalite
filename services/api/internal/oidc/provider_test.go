package oidc_test

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/oidc"
	"github.com/mdg-labs/escalite/services/api/internal/oidc/testutil"
)

func TestProviderExchangeReturnsVerifiedEmail(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
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
		require.NoError(a, err)

		userInfo, err := provider.Exchange(context.Background(), "test-auth-code")
		require.NoError(a, err)
		require.Equal(a, "oidc-user@example.com", userInfo.Email)
		require.Equal(a, "oidc-subject-oidc-user@example.com", userInfo.Subject)
	})
}
