package saml

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/crewjam/saml"
)

// ProviderConfig holds runtime SAML service provider settings.
type ProviderConfig struct {
	PublicURL        string
	IDPEntityID      string
	IDPSSOURL        string
	IDPCertificatePEM string
	SPCertificatePEM string
	SPPrivateKeyPEM  string
}

// ServiceProvider is the crewjam SAML service provider type used by handlers.
type ServiceProvider = saml.ServiceProvider

// BuildServiceProvider constructs a crewjam/saml service provider from stored settings.
func BuildServiceProvider(cfg ProviderConfig) (*ServiceProvider, error) {
	publicURL := strings.TrimSuffix(strings.TrimSpace(cfg.PublicURL), "/")
	if publicURL == "" {
		return nil, fmt.Errorf("public url is required for saml")
	}

	idpCert, err := ParseCertificate(cfg.IDPCertificatePEM)
	if err != nil {
		return nil, fmt.Errorf("parse idp certificate: %w", err)
	}
	_ = idpCert

	spCert, err := ParseCertificate(cfg.SPCertificatePEM)
	if err != nil {
		return nil, fmt.Errorf("parse sp certificate: %w", err)
	}

	spKey, err := ParsePrivateKey(cfg.SPPrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse sp private key: %w", err)
	}

	metadataURL, err := url.Parse(publicURL + "/api/v1/auth/saml/metadata")
	if err != nil {
		return nil, fmt.Errorf("parse metadata url: %w", err)
	}
	acsURL, err := url.Parse(publicURL + "/api/v1/auth/saml/acs")
	if err != nil {
		return nil, fmt.Errorf("parse acs url: %w", err)
	}

	entityID := publicURL + "/api/v1/auth/saml/metadata"
	idpMetadata := &saml.EntityDescriptor{
		EntityID: strings.TrimSpace(cfg.IDPEntityID),
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
					},
				},
				SingleSignOnServices: []saml.Endpoint{
					{Binding: saml.HTTPRedirectBinding, Location: strings.TrimSpace(cfg.IDPSSOURL)},
					{Binding: saml.HTTPPostBinding, Location: strings.TrimSpace(cfg.IDPSSOURL)},
				},
			},
		},
	}

	idpCertPEM := strings.TrimSpace(cfg.IDPCertificatePEM)

	return &ServiceProvider{
		EntityID:          entityID,
		Key:               spKey,
		Certificate:       spCert,
		MetadataURL:       *metadataURL,
		AcsURL:            *acsURL,
		IDPMetadata:       idpMetadata,
		IDPCertificate:    &idpCertPEM,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
	}, nil
}

// LoginRedirectURL starts SP-initiated login and returns the IdP redirect URL and AuthnRequest ID.
func LoginRedirectURL(sp *ServiceProvider, relayState string) (*url.URL, string, error) {
	req, err := sp.MakeAuthenticationRequest(
		sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		return nil, "", fmt.Errorf("make authentication request: %w", err)
	}

	redirectURL, err := req.Redirect(relayState, sp)
	if err != nil {
		return nil, "", fmt.Errorf("redirect authentication request: %w", err)
	}
	return redirectURL, req.ID, nil
}

// ParseACSResponse validates a SAML response posted to the ACS endpoint.
func ParseACSResponse(sp *ServiceProvider, r *http.Request, requestID string) (*saml.Assertion, error) {
	possibleRequestIDs := []string{}
	if requestID != "" {
		possibleRequestIDs = append(possibleRequestIDs, requestID)
	}
	return sp.ParseResponse(r, possibleRequestIDs)
}

// EmailFromAssertion extracts the authenticated user's email from a SAML assertion.
func EmailFromAssertion(assertion *saml.Assertion) string {
	if assertion == nil {
		return ""
	}
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		if email := strings.TrimSpace(assertion.Subject.NameID.Value); email != "" {
			return email
		}
	}
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			name := strings.ToLower(attribute.Name)
			if name == "email" ||
				name == "mail" ||
				strings.HasSuffix(name, "/emailaddress") ||
				strings.EqualFold(attribute.FriendlyName, "email") {
				if len(attribute.Values) > 0 {
					if email := strings.TrimSpace(attribute.Values[0].Value); email != "" {
						return email
					}
				}
			}
		}
	}
	return ""
}
