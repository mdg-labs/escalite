package graph

import (
	"encoding/json"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/engine/channels"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
)

func notificationChannelDefinitions() []*model.NotificationChannelDefinition {
	plugins := channels.List()
	definitions := make([]*model.NotificationChannelDefinition, 0, len(plugins))
	for _, plugin := range plugins {
		var schema map[string]any
		if err := json.Unmarshal(plugin.ConfigSchema(), &schema); err != nil {
			continue
		}
		definitions = append(definitions, &model.NotificationChannelDefinition{
			Name:         plugin.Name(),
			ConfigSchema: schema,
		})
	}
	return definitions
}
