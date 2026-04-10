package hasync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/christofer-l/tibber-cli/homeassistant"
	"github.com/christofer-l/tibber-cli/tibber"
)

type mqttPublish struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
	Retain  bool   `json:"retain"`
}

func fakeTibberServer(t *testing.T, prices, consumption tibber.GraphQLResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req tibber.GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Variables != nil && req.Variables["last"] != nil {
			json.NewEncoder(w).Encode(consumption)
		} else {
			json.NewEncoder(w).Encode(prices)
		}
	}))
}

func fakeHAServer(t *testing.T, publishes *[]mqttPublish) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var pub mqttPublish
		json.NewDecoder(r.Body).Decode(&pub)
		*publishes = append(*publishes, pub)
		w.WriteHeader(http.StatusOK)
	}))
}

// discoveryTopics extracts unique sensor IDs from discovery config topics
func discoveryTopics(pubs []mqttPublish) map[string]bool {
	topics := map[string]bool{}
	for _, p := range pubs {
		if strings.HasPrefix(p.Topic, "homeassistant/sensor/") && strings.HasSuffix(p.Topic, "/config") {
			topics[p.Topic] = true
		}
	}
	return topics
}

func makePriceResponse() tibber.GraphQLResponse {
	today := make([]tibber.Price, 10)
	for i := range today {
		today[i] = tibber.Price{
			Total:    float64(i+1) * 0.10,
			Energy:   float64(i+1) * 0.07,
			Tax:      float64(i+1) * 0.03,
			Currency: "SEK",
			StartsAt: time.Date(2026, 4, 10, i, 0, 0, 0, time.UTC),
		}
	}
	tomorrow := []tibber.Price{
		{Total: 0.55, Energy: 0.40, Tax: 0.15, Currency: "SEK", StartsAt: time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)},
	}
	return tibber.GraphQLResponse{
		Data: tibber.Data{Viewer: tibber.Viewer{Homes: []tibber.Home{{
			CurrentSubscription: tibber.Subscription{PriceInfo: tibber.PriceInfo{
				Current: tibber.Price{
					Total:    0.90,
					Energy:   0.63,
					Tax:      0.27,
					Currency: "SEK",
					StartsAt: time.Date(2026, 4, 10, 8, 0, 0, 0, time.UTC),
				},
				Today:    today,
				Tomorrow: tomorrow,
			}},
		}}}},
	}
}

func makeConsumptionResponse() tibber.GraphQLResponse {
	return tibber.GraphQLResponse{
		Data: tibber.Data{Viewer: tibber.Viewer{Homes: []tibber.Home{{
			Consumption: tibber.Consumption{Nodes: []tibber.ConsumptionNode{
				{
					From:        time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC),
					To:          time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC),
					Consumption: 1.5,
					Cost:        0.75,
					UnitPrice:   0.50,
				},
				{
					From:        time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC),
					To:          time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC),
					Consumption: 2.3,
					Cost:        1.15,
					UnitPrice:   0.50,
				},
			}},
		}}}},
	}
}

func TestRunPublishesFourSensors(t *testing.T) {
	var pubs []mqttPublish
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &pubs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	n, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if n != 4 {
		t.Errorf("synced %d sensors, want 4", n)
	}

	topics := discoveryTopics(pubs)
	for _, want := range []string{
		"homeassistant/sensor/tibber_price_current/config",
		"homeassistant/sensor/tibber_price_level/config",
		"homeassistant/sensor/tibber_consumption_hourly/config",
		"homeassistant/sensor/tibber_cost_hourly/config",
	} {
		if !topics[want] {
			t.Errorf("missing discovery topic: %s", want)
		}
	}
}

func TestRunUsesLatestConsumptionNode(t *testing.T) {
	var pubs []mqttPublish
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &pubs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, p := range pubs {
		if p.Topic == "tibber_cli/tibber_consumption_hourly/state" {
			var state homeassistant.SensorState
			json.Unmarshal([]byte(p.Payload), &state)
			if state.State != "2.300" {
				t.Errorf("consumption state = %q, want %q", state.State, "2.300")
			}
			return
		}
	}
	t.Error("consumption state topic not found")
}

func TestRunNoConsumptionData(t *testing.T) {
	var pubs []mqttPublish
	emptyConsumption := tibber.GraphQLResponse{
		Data: tibber.Data{Viewer: tibber.Viewer{Homes: []tibber.Home{{
			Consumption: tibber.Consumption{Nodes: []tibber.ConsumptionNode{}},
		}}}},
	}
	tibberSrv := fakeTibberServer(t, makePriceResponse(), emptyConsumption)
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &pubs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	n, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if n != 2 {
		t.Errorf("synced %d sensors, want 2 (price + price_level only)", n)
	}
}

func TestRunPriceLevelPercentile(t *testing.T) {
	var pubs []mqttPublish
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &pubs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, p := range pubs {
		if p.Topic == "tibber_cli/tibber_price_level/state" {
			var state homeassistant.SensorState
			json.Unmarshal([]byte(p.Payload), &state)
			// Current price 0.90, prices are 0.10..1.00
			// 9 out of 10 are <= 0.90, percentile = 90
			if state.State != "90" {
				t.Errorf("price_level state = %q, want %q", state.State, "90")
			}
			return
		}
	}
	t.Error("price_level state topic not found")
}

func TestRunPriceLevelAttributes(t *testing.T) {
	var pubs []mqttPublish
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &pubs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, p := range pubs {
		if p.Topic == "tibber_cli/tibber_price_level/attributes" {
			var attrs map[string]any
			json.Unmarshal([]byte(p.Payload), &attrs)
			if attrs["min"] != 0.1 {
				t.Errorf("min = %v, want 0.1", attrs["min"])
			}
			if attrs["max"] != 1.0 {
				t.Errorf("max = %v, want 1.0", attrs["max"])
			}
			return
		}
	}
	t.Error("price_level attributes topic not found")
}

func TestPercentile(t *testing.T) {
	prices := []tibber.Price{
		{Total: 0.10}, {Total: 0.20}, {Total: 0.30}, {Total: 0.40}, {Total: 0.50},
	}
	tests := []struct {
		current float64
		want    int
	}{
		{0.10, 20},  // 1 out of 5 <= 0.10
		{0.30, 60},  // 3 out of 5 <= 0.30
		{0.50, 100}, // 5 out of 5 <= 0.50
		{0.05, 0},   // 0 out of 5 < 0.05
	}
	for _, tt := range tests {
		got := percentile(tt.current, prices)
		if got != tt.want {
			t.Errorf("percentile(%.2f) = %d, want %d", tt.current, got, tt.want)
		}
	}
}

func TestRunTibberError(t *testing.T) {
	var pubs []mqttPublish
	tibberSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &pubs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(pubs) != 0 {
		t.Errorf("HA received %d publishes, want 0", len(pubs))
	}
}
