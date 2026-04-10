package homeassistant

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server, token string) *Client {
	return &Client{
		baseURL:    srv.URL,
		token:      token,
		httpClient: srv.Client(),
	}
}

func TestSetStateSendsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, "ha-token-123")
	err := c.SetState("sensor.test", SensorState{State: "42"})
	if err != nil {
		t.Fatalf("SetState failed: %v", err)
	}
	if gotAuth != "Bearer ha-token-123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer ha-token-123")
	}
}

func TestSetStateSendsCorrectBody(t *testing.T) {
	var gotBody SensorState
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, "tok")
	sensor := SensorState{
		State: "1.234",
		Attributes: map[string]any{
			"unit_of_measurement": "kWh",
			"friendly_name":      "Test Sensor",
		},
	}
	err := c.SetState("sensor.test", sensor)
	if err != nil {
		t.Fatalf("SetState failed: %v", err)
	}
	if gotBody.State != "1.234" {
		t.Errorf("state = %q, want %q", gotBody.State, "1.234")
	}
	if gotBody.Attributes["unit_of_measurement"] != "kWh" {
		t.Errorf("unit = %v, want kWh", gotBody.Attributes["unit_of_measurement"])
	}
}

func TestSetStateSendsToCorrectPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, "tok")
	c.SetState("sensor.tibber_price_current", SensorState{State: "0.50"})

	want := "/api/states/sensor.tibber_price_current"
	if gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestSetStateMethodIsPOST(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv, "tok")
	c.SetState("sensor.test", SensorState{State: "1"})

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

func TestSetStateHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	c := newTestClient(srv, "bad-token")
	err := c.SetState("sensor.test", SensorState{State: "1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
