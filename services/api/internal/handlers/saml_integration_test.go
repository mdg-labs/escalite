package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/saml"
)

const testIDPMetadataXML = `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com/metadata">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIIB7zCCAVgCCQDFzbKIp7b3MTANBgkqhkiG9w0BAQUFADA8MQswCQYDVQQGEwJVUzELMAkGA1UECAwCR0ExDDAKBgNVBAoMA2ZvbzESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTEzMTAwMjAwMDg1MVoXDTE0MTAwMjAwMDg1MVowPDELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkdBMQwwCgYDVQQKDANmb28xEjAQBgNVBAMMCWxvY2FsaG9zdDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1PMHYmhZj308kWLhZVT4vOulqx/9ibm5B86fPWwUKKQ2i12MYtz07tzukPymisTDhQaqyJ8Kqb/6JjhmeMnEOdTvSPmHO8m1ZVveJU6NoKRn/mP/BD7FW52WhbrUXLSeHVSKfWkNk6S4hk9MV9TswTvyRIKvRsw0X/gfnqkroJcCAwEAATANBgkqhkiG9w0BAQUFAAOBgQCMMlIO+GNcGekevKgkakpMdAqJfs24maGb90DvTLbRZRD7Xvn1MnVBBS9hzlXiFLYOInXACMW5gcoRFfeTQLSouMM8o57h0uKjfTmuoWHLQLi6hnF+cvCsEFiJZ4AbF+DgmO6TarJ8O05t8zvnOwJlNCASPZRH/JmF8tX0hoHuAQ==</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"></SingleSignOnService>
  </IDPSSODescriptor>
</EntityDescriptor>`

func TestSAMLLoginDisabledReturnsNotFound(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/saml/login", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusNotFound, rec.Code)
	})
}

func TestGraphQLLoginOptionsWithoutSAML(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		rec := postGraphQL(t, handler, `{ loginOptions { samlEnabled oidcEnabled } }`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				LoginOptions struct {
					SamlEnabled bool `json:"samlEnabled"`
					OidcEnabled bool `json:"oidcEnabled"`
				} `json:"loginOptions"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.False(a, resp.Data.LoginOptions.SamlEnabled)
		require.False(a, resp.Data.LoginOptions.OidcEnabled)
	})
}

func TestSAMLLoginEnabledRedirectsToIdP(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)
		seedEnabledSAMLSettings(t, pool)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/saml/login", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusFound, rec.Code)
		require.Contains(a, rec.Header().Get("Location"), "https://idp.example.com/sso")
	})
}

func TestGraphQLSaveSamlSettingsAndLoginOptions(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		saveRec := postGraphQL(t, handler, `mutation {
		saveSamlSettings(input: { metadataXml: """`+testIDPMetadataXML+`""", enabled: true }) {
			configured
			enabled
			idpEntityId
			certificateHint
		}
	}`, cookie)
		require.Equal(a, 200, saveRec.Code, saveRec.Body.String())

		var saveResp struct {
			Data struct {
				SaveSamlSettings struct {
					Configured      bool   `json:"configured"`
					Enabled         bool   `json:"enabled"`
					IdpEntityID     string `json:"idpEntityId"`
					CertificateHint string `json:"certificateHint"`
				} `json:"saveSamlSettings"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(saveRec.Body.Bytes(), &saveResp))
		require.Empty(a, saveResp.Errors)
		require.True(a, saveResp.Data.SaveSamlSettings.Configured)
		require.True(a, saveResp.Data.SaveSamlSettings.Enabled)
		require.Equal(a, "https://idp.example.com/metadata", saveResp.Data.SaveSamlSettings.IdpEntityID)
		require.NotEmpty(a, saveResp.Data.SaveSamlSettings.CertificateHint)
		require.NotContains(a, saveRec.Body.String(), "BEGIN CERTIFICATE")

		optionsRec := postGraphQL(t, handler, `{ loginOptions { samlEnabled samlLoginUrl } }`, nil)
		require.Equal(a, 200, optionsRec.Code)

		var optionsResp struct {
			Data struct {
				LoginOptions struct {
					SamlEnabled  bool   `json:"samlEnabled"`
					SamlLoginURL string `json:"samlLoginUrl"`
				} `json:"loginOptions"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(optionsRec.Body.Bytes(), &optionsResp))
		require.True(a, optionsResp.Data.LoginOptions.SamlEnabled)
		require.Contains(a, optionsResp.Data.LoginOptions.SamlLoginURL, "/api/v1/auth/saml/login")
	})
}

func seedEnabledSAMLSettings(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	queries := db.New(pool)
	org, err := queries.GetFirstOrganization(context.Background())
	require.NoError(t, err)

	secrets := testSecretsBox(t)
	idp, err := saml.ParseIDPMetadata(testIDPMetadataXML)
	require.NoError(t, err)

	keyPair, err := saml.GenerateKeyPair()
	require.NoError(t, err)

	encrypted, err := secrets.Encrypt([]byte(keyPair.PrivateKeyPEM))
	require.NoError(t, err)

	_, err = queries.UpsertOrganizationSamlSettings(context.Background(), db.UpsertOrganizationSamlSettingsParams{
		OrganizationID:         org.ID,
		Enabled:                true,
		IdpEntityID:            idp.EntityID,
		IdpSsoUrl:              idp.SSOURL,
		IdpCertificatePem:      idp.CertificatePEM,
		SpCertificatePem:       keyPair.CertificatePEM,
		SpPrivateKeyCiphertext: encrypted.Ciphertext,
		SpEncryptionKeyID:      encrypted.KeyID,
		CertificateHint:        saml.CertificateHint(idp.CertificatePEM),
	})
	require.NoError(t, err)
}
