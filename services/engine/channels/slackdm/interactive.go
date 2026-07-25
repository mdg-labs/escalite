package slackdm

import "encoding/json"

const (
	// ActionAck is the Slack block action id for acknowledging an alert.
	ActionAck = "escalite_ack"
	// ActionEscalate is the Slack block action id for re-escalating an alert.
	ActionEscalate = "escalite_escalate"
)

type buttonValue struct {
	AlertID string `json:"alert_id"`
}

// ButtonValue returns the JSON value stored on Slack interactive buttons.
func ButtonValue(alertID string) string {
	raw, err := json.Marshal(buttonValue{AlertID: alertID})
	if err != nil {
		return `{"alert_id":""}`
	}
	return string(raw)
}
