package homeassistant

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(srv *httptest.Server, token string) *Client {
	return &Client{
		baseURL:    srv.URL,
		token:      token,
		httpClient: srv.Client(),
	}
}

type mqttCall struct {
	Path string
	Body mqttPublishPayload
}

func TestPublishSensorSendsDiscoveryAndState(t *testing.T) {
	var calls []mqttCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body mqttPublishPayload
		json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, mqttCall{Path: r.URL.Path, Body: body})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, "tok")
	config := MQTTDiscoveryConfig{
		Name:              "Test Sensor",
		UniqueID:          "tibber_test",
		UnitOfMeasurement: "kWh",
		DeviceClass:       "energy",
	}
	state := SensorState{
		State:      "1.234",
		Attributes: map[string]any{"from": "2026-01-01"},
	}

	err := c.PublishSensor("tibber_test", config, state)
	if err != nil {
		t.Fatalf("PublishSensor failed: %v", err)
	}

	// Expect 3 calls: discovery config, state, attributes
	if len(calls) != 3 {
		t.Fatalf("got %d MQTT publishes, want 3", len(calls))
	}

	// All should go to mqtt/publish
	for _, call := range calls {
		if call.Path != "/api/services/mqtt/publish" {
			t.Errorf("path = %q, want /api/services/mqtt/publish", call.Path)
		}
	}

	// Check discovery topic
	if !strings.Contains(calls[0].Body.Topic, "homeassistant/sensor/tibber_test/config") {
		t.Errorf("discovery topic = %q", calls[0].Body.Topic)
	}
	if !calls[0].Body.Retain {
		t.Error("discovery should be retained")
	}

	// Check discovery payload has unique_id and device
	var disc MQTTDiscoveryConfig
	json.Unmarshal([]byte(calls[0].Body.Payload), &disc)
	if disc.UniqueID != "tibber_test" {
		t.Errorf("unique_id = %q, want tibber_test", disc.UniqueID)
	}
	if disc.Device.Name != "Tibber CLI" {
		t.Errorf("device name = %q, want Tibber CLI", disc.Device.Name)
	}
	if disc.StateTopic != "tibber_cli/tibber_test/state" {
		t.Errorf("state_topic = %q", disc.StateTopic)
	}

	// Check state topic
	if calls[1].Body.Topic != "tibber_cli/tibber_test/state" {
		t.Errorf("state topic = %q", calls[1].Body.Topic)
	}

	// Check attributes topic
	if calls[2].Body.Topic != "tibber_cli/tibber_test/attributes" {
		t.Errorf("attributes topic = %q", calls[2].Body.Topic)
	}
}

func TestPublishSensorAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, "ha-token-123")
	c.PublishSensor("test", MQTTDiscoveryConfig{
		Name:     "Test",
		UniqueID: "test",
	}, SensorState{State: "1"})

	if gotAuth != "Bearer ha-token-123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer ha-token-123")
	}
}

func TestPublishSensorHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	c := newTestClient(srv, "bad-token")
	err := c.PublishSensor("test", MQTTDiscoveryConfig{
		Name:     "Test",
		UniqueID: "test",
	}, SensorState{State: "1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
