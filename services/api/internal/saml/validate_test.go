package saml_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/saml"
)

func TestParseACSResponseRejectsInvalidPayload(t *testing.T) {
	idp, err := saml.ParseIDPMetadata(testIDPMetadataXML)
	require.NoError(t, err)

	spKeyPair, err := saml.GenerateKeyPair()
	require.NoError(t, err)

	sp, err := saml.BuildServiceProvider(saml.ProviderConfig{
		PublicURL:         "http://example.com",
		IDPEntityID:       idp.EntityID,
		IDPSSOURL:         idp.SSOURL,
		IDPCertificatePEM: idp.CertificatePEM,
		SPCertificatePEM:  spKeyPair.CertificatePEM,
		SPPrivateKeyPEM:   spKeyPair.PrivateKeyPEM,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/saml/acs", strings.NewReader("SAMLResponse=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err = saml.ParseACSResponse(sp, req, "test-request-id")
	require.Error(t, err)
}
