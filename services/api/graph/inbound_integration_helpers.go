package graph

import (
	"encoding/json"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func inboundIntegrationDefinitions() []*model.IntegrationPluginDefinition {
	plugins := integrations.List()
	definitions := make([]*model.IntegrationPluginDefinition, 0, len(plugins))
	for _, plugin := range plugins {
		var schema map[string]any
		if err := json.Unmarshal(plugin.ConfigSchema(), &schema); err != nil {
			continue
		}
		definitions = append(definitions, &model.IntegrationPluginDefinition{
			Name:         plugin.Name(),
			ConfigSchema: schema,
		})
	}
	return definitions
}
