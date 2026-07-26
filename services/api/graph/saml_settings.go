package graph

import (
	"strings"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	escalitesaml "github.com/mdg-labs/escalite/services/api/internal/saml"
)

func samlSettingsFromDB(settings *db.OrganizationSamlSetting, publicURL string) *model.SamlSettings {
	result := &model.SamlSettings{
		Configured: settings != nil,
		Enabled:    settings != nil && settings.Enabled,
	}
	if settings == nil {
		return result
	}

	entityID := settings.IdpEntityID
	result.IdpEntityID = &entityID
	hint := settings.CertificateHint
	result.CertificateHint = &hint
	loginURL := strings.TrimSuffix(strings.TrimSpace(publicURL), "/") + "/api/v1/auth/saml/login"
	result.SamlLoginURL = &loginURL
	return result
}

func encryptSamlPrivateKey(box *crypto.Box, privateKeyPEM string) ([]byte, string, error) {
	encrypted, err := box.Encrypt([]byte(privateKeyPEM))
	if err != nil {
		return nil, "", err
	}
	return encrypted.Ciphertext, encrypted.KeyID, nil
}

func prepareSamlSettingsSave(
	box *crypto.Box,
	existing *db.OrganizationSamlSetting,
	metadataXML string,
	enabled bool,
) (db.UpsertOrganizationSamlSettingsParams, error) {
	idp, err := escalitesaml.ParseIDPMetadata(metadataXML)
	if err != nil {
		return db.UpsertOrganizationSamlSettingsParams{}, err
	}

	spCertPEM := ""
	spKeyCipher := []byte{}
	spKeyID := ""
	if existing != nil {
		spCertPEM = existing.SpCertificatePem
		spKeyCipher = existing.SpPrivateKeyCiphertext
		spKeyID = existing.SpEncryptionKeyID
	}
	if spCertPEM == "" || len(spKeyCipher) == 0 {
		keyPair, err := escalitesaml.GenerateKeyPair()
		if err != nil {
			return db.UpsertOrganizationSamlSettingsParams{}, err
		}
		spCertPEM = keyPair.CertificatePEM
		ciphertext, keyID, err := encryptSamlPrivateKey(box, keyPair.PrivateKeyPEM)
		if err != nil {
			return db.UpsertOrganizationSamlSettingsParams{}, err
		}
		spKeyCipher = ciphertext
		spKeyID = keyID
	}

	return db.UpsertOrganizationSamlSettingsParams{
		Enabled:                enabled,
		IdpEntityID:            idp.EntityID,
		IdpSsoUrl:              idp.SSOURL,
		IdpCertificatePem:      idp.CertificatePEM,
		SpCertificatePem:       spCertPEM,
		SpPrivateKeyCiphertext: spKeyCipher,
		SpEncryptionKeyID:      spKeyID,
		CertificateHint:        escalitesaml.CertificateHint(idp.CertificatePEM),
	}, nil
}

var errSamlMetadataRequired = validationError("saml idp metadata xml is required")
