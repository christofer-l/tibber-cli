package homeassistant

type SensorState struct {
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes"`
}
