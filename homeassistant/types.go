package homeassistant

type SensorState struct {
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type MQTTDiscoveryConfig struct {
	Name              string         `json:"name"`
	UniqueID          string         `json:"unique_id"`
	StateTopic        string         `json:"state_topic"`
	UnitOfMeasurement string         `json:"unit_of_measurement,omitempty"`
	DeviceClass       string         `json:"device_class,omitempty"`
	StateClass        string         `json:"state_class,omitempty"`
	ValueTemplate     string         `json:"value_template,omitempty"`
	JSONAttributesTopic string       `json:"json_attributes_topic,omitempty"`
	Device            MQTTDevice     `json:"device"`
}

type MQTTDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

type mqttPublishPayload struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
	Retain  bool   `json:"retain"`
}
