package graph

import (
	"encoding/json"
	"strings"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func slackSettingsFromDB(settings *db.OrganizationSlackSetting) *model.SlackSettings {
	if settings == nil {
		return &model.SlackSettings{Configured: false}
	}
	hint := settings.TokenHint
	return &model.SlackSettings{
		Configured: true,
		TokenHint:  &hint,
	}
}

func userContactMethodFromDB(method db.UserContactMethod) (*model.UserContactMethod, error) {
	config := map[string]any{}
	if len(method.Config) > 0 {
		if err := json.Unmarshal(method.Config, &config); err != nil {
			return nil, err
		}
	}
	return &model.UserContactMethod{
		ID:        method.ID.String(),
		UserID:    method.UserID.String(),
		Channel:   method.Channel,
		Config:    config,
		CreatedAt: timeFromDB(method.CreatedAt),
		UpdatedAt: timeFromDB(method.UpdatedAt),
	}, nil
}

func validateSlackBotToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return errSlackBotTokenRequired
	}
	return nil
}

func encryptSlackBotToken(box *crypto.Box, token string) (db.UpsertOrganizationSlackSettingsParams, error) {
	encrypted, err := box.Encrypt([]byte(token))
	if err != nil {
		return db.UpsertOrganizationSlackSettingsParams{}, err
	}
	return db.UpsertOrganizationSlackSettingsParams{
		BotTokenCiphertext: encrypted.Ciphertext,
		EncryptionKeyID:    encrypted.KeyID,
		TokenHint:          crypto.SecretHint(token),
	}, nil
}

var errSlackBotTokenRequired = validationError("slack bot token is required")

func validationError(message string) error {
	return &validationErrorMessage{message: message}
}

type validationErrorMessage struct {
	message string
}

func (e *validationErrorMessage) Error() string {
	return e.message
}
