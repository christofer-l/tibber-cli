package hasync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/christofer-l/tibber-cli/homeassistant"
	"github.com/christofer-l/tibber-cli/tibber"
)

type haRequest struct {
	Path string
	Body homeassistant.SensorState
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

func fakeHAServer(t *testing.T, requests *[]haRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body homeassistant.SensorState
		json.NewDecoder(r.Body).Decode(&body)
		*requests = append(*requests, haRequest{Path: r.URL.Path, Body: body})
		w.WriteHeader(http.StatusOK)
	}))
}

func makePriceResponse() tibber.GraphQLResponse {
	// 10 hourly prices: 0.10, 0.20, ..., 1.00
	// Current price is 0.90 (9th out of 10 = 80th percentile)
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

func TestRunPushesThreeSensors(t *testing.T) {
	var haReqs []haRequest
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &haReqs)
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
	if len(haReqs) != 4 {
		t.Fatalf("HA received %d requests, want 4", len(haReqs))
	}

	paths := map[string]bool{}
	for _, r := range haReqs {
		paths[r.Path] = true
	}
	for _, want := range []string{
		"/api/states/sensor.tibber_price_current",
		"/api/states/sensor.tibber_price_level",
		"/api/states/sensor.tibber_consumption_hourly",
		"/api/states/sensor.tibber_cost_hourly",
	} {
		if !paths[want] {
			t.Errorf("missing request to %s", want)
		}
	}
}

func TestRunUsesLatestConsumptionNode(t *testing.T) {
	var haReqs []haRequest
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &haReqs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, r := range haReqs {
		if r.Path == "/api/states/sensor.tibber_consumption_hourly" {
			if r.Body.State != "2.300" {
				t.Errorf("consumption state = %q, want %q", r.Body.State, "2.300")
			}
		}
	}
}

func TestRunNoConsumptionData(t *testing.T) {
	var haReqs []haRequest
	emptyConsumption := tibber.GraphQLResponse{
		Data: tibber.Data{Viewer: tibber.Viewer{Homes: []tibber.Home{{
			Consumption: tibber.Consumption{Nodes: []tibber.ConsumptionNode{}},
		}}}},
	}
	tibberSrv := fakeTibberServer(t, makePriceResponse(), emptyConsumption)
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &haReqs)
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
	var haReqs []haRequest
	tibberSrv := fakeTibberServer(t, makePriceResponse(), makeConsumptionResponse())
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &haReqs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	for _, r := range haReqs {
		if r.Path == "/api/states/sensor.tibber_price_level" {
			// Current price 0.90, prices are 0.10..1.00
			// 9 out of 10 are <= 0.90, percentile = 90
			if r.Body.State != "90" {
				t.Errorf("price_level state = %q, want %q", r.Body.State, "90")
			}
			if r.Body.Attributes["min"] != 0.10 {
				t.Errorf("min = %v, want 0.10", r.Body.Attributes["min"])
			}
			if r.Body.Attributes["max"] != 1.0 {
				t.Errorf("max = %v, want 1.00", r.Body.Attributes["max"])
			}
			return
		}
	}
	t.Error("price_level sensor not found in HA requests")
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
	var haReqs []haRequest
	tibberSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer tibberSrv.Close()
	haSrv := fakeHAServer(t, &haReqs)
	defer haSrv.Close()

	tc := tibber.NewTestClient("tok", tibberSrv.URL, tibberSrv.Client())
	hac := homeassistant.NewTestClient(haSrv.URL, "ha-tok", haSrv.Client())

	_, err := Run(tc, hac)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(haReqs) != 0 {
		t.Errorf("HA received %d requests, want 0", len(haReqs))
	}
}
