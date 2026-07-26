package graph

import (
	"strings"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func scimSettingsFromDB(settings *db.OrganizationScimSetting, publicURL string) *model.ScimSettings {
	result := &model.ScimSettings{
		Configured: settings != nil,
	}
	if settings == nil {
		return result
	}

	hint := settings.TokenPrefix
	result.TokenHint = &hint
	baseURL := strings.TrimSuffix(strings.TrimSpace(publicURL), "/") + "/scim/v2"
	result.ScimBaseURL = &baseURL
	return result
}
