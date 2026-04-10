package homeassistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	baseTopic     = "tibber_cli"
	discoveryBase = "homeassistant/sensor"
)

var tibberDevice = MQTTDevice{
	Identifiers:  []string{"tibber_cli"},
	Name:         "Tibber CLI",
	Manufacturer: "Tibber",
	Model:        "tibber-cli",
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

// NewTestClient creates a client with a custom HTTP client, for testing.
func NewTestClient(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: httpClient,
	}
}

// PublishSensor registers a sensor via MQTT discovery and publishes its state.
func (c *Client) PublishSensor(id string, config MQTTDiscoveryConfig, state SensorState) error {
	// Set topics based on id
	stateTopic := fmt.Sprintf("%s/%s/state", baseTopic, id)
	attrTopic := fmt.Sprintf("%s/%s/attributes", baseTopic, id)
	config.StateTopic = stateTopic
	config.ValueTemplate = "{{ value_json.state }}"
	config.JSONAttributesTopic = attrTopic
	config.Device = tibberDevice

	// Publish discovery config (retained)
	discoveryTopic := fmt.Sprintf("%s/%s/config", discoveryBase, id)
	if err := c.mqttPublish(discoveryTopic, config, true); err != nil {
		return fmt.Errorf("publish discovery for %s: %w", id, err)
	}

	// Publish state (retained)
	if err := c.mqttPublish(stateTopic, state, true); err != nil {
		return fmt.Errorf("publish state for %s: %w", id, err)
	}

	// Publish attributes (retained)
	if state.Attributes != nil {
		if err := c.mqttPublish(attrTopic, state.Attributes, true); err != nil {
			return fmt.Errorf("publish attributes for %s: %w", id, err)
		}
	}

	return nil
}

func (c *Client) mqttPublish(topic string, payload any, retain bool) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	body, err := json.Marshal(mqttPublishPayload{
		Topic:   topic,
		Payload: string(payloadJSON),
		Retain:  retain,
	})
	if err != nil {
		return fmt.Errorf("marshal mqtt publish: %w", err)
	}

	url := fmt.Sprintf("%s/api/services/mqtt/publish", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HA API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SetState is kept for backward compatibility but deprecated in favor of PublishSensor.
func (c *Client) SetState(entityID string, sensor SensorState) error {
	body, err := json.Marshal(sensor)
	if err != nil {
		return fmt.Errorf("marshal sensor state: %w", err)
	}

	url := fmt.Sprintf("%s/api/states/%s", c.baseURL, entityID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HA API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
