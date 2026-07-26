package saml

import (
	"fmt"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

// IDPConfig is the identity provider configuration extracted from metadata.
type IDPConfig struct {
	EntityID      string
	SSOURL        string
	CertificatePEM string
}

// ParseIDPMetadata extracts IdP settings from SAML metadata XML.
func ParseIDPMetadata(metadataXML string) (IDPConfig, error) {
	metadataXML = strings.TrimSpace(metadataXML)
	if metadataXML == "" {
		return IDPConfig{}, fmt.Errorf("idp metadata xml is required")
	}

	entity, err := samlsp.ParseMetadata([]byte(metadataXML))
	if err != nil {
		return IDPConfig{}, fmt.Errorf("parse idp metadata: %w", err)
	}

	if len(entity.IDPSSODescriptors) == 0 {
		return IDPConfig{}, fmt.Errorf("idp metadata missing IDPSSODescriptor")
	}

	descriptor := entity.IDPSSODescriptors[0]
	ssoURL := bindingLocation(descriptor.SingleSignOnServices, saml.HTTPRedirectBinding)
	if ssoURL == "" {
		ssoURL = bindingLocation(descriptor.SingleSignOnServices, saml.HTTPPostBinding)
	}
	if ssoURL == "" {
		return IDPConfig{}, fmt.Errorf("idp metadata missing single sign-on endpoint")
	}

	certPEM, err := signingCertificatePEM(descriptor.KeyDescriptors)
	if err != nil {
		return IDPConfig{}, err
	}

	entityID := strings.TrimSpace(entity.EntityID)
	if entityID == "" {
		return IDPConfig{}, fmt.Errorf("idp metadata missing entity id")
	}

	return IDPConfig{
		EntityID:       entityID,
		SSOURL:         ssoURL,
		CertificatePEM: certPEM,
	}, nil
}

func bindingLocation(endpoints []saml.Endpoint, binding string) string {
	for _, endpoint := range endpoints {
		if endpoint.Binding == binding && strings.TrimSpace(endpoint.Location) != "" {
			return endpoint.Location
		}
	}
	return ""
}

func signingCertificatePEM(descriptors []saml.KeyDescriptor) (string, error) {
	for _, descriptor := range descriptors {
		if descriptor.Use != "" && descriptor.Use != "signing" {
			continue
		}
		for _, cert := range descriptor.KeyInfo.X509Data.X509Certificates {
			data := strings.TrimSpace(cert.Data)
			if data == "" {
				continue
			}
			return formatCertificatePEM(data), nil
		}
	}
	return "", fmt.Errorf("idp metadata missing signing certificate")
}

func formatCertificatePEM(data string) string {
	data = strings.ReplaceAll(data, "\n", "")
	data = strings.ReplaceAll(data, "\r", "")
	data = strings.ReplaceAll(data, " ", "")
	var b strings.Builder
	b.WriteString("-----BEGIN CERTIFICATE-----\n")
	for len(data) > 64 {
		b.WriteString(data[:64])
		b.WriteByte('\n')
		data = data[64:]
	}
	if data != "" {
		b.WriteString(data)
		b.WriteByte('\n')
	}
	b.WriteString("-----END CERTIFICATE-----\n")
	return b.String()
}
