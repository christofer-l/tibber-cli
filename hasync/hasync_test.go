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
	return tibber.GraphQLResponse{
		Data: tibber.Data{Viewer: tibber.Viewer{Homes: []tibber.Home{{
			CurrentSubscription: tibber.Subscription{PriceInfo: tibber.PriceInfo{
				Current: tibber.Price{
					Total:    0.4523,
					Energy:   0.3200,
					Tax:      0.1323,
					Currency: "SEK",
					StartsAt: time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC),
				},
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
	if n != 3 {
		t.Errorf("synced %d sensors, want 3", n)
	}
	if len(haReqs) != 3 {
		t.Fatalf("HA received %d requests, want 3", len(haReqs))
	}

	paths := map[string]bool{}
	for _, r := range haReqs {
		paths[r.Path] = true
	}
	for _, want := range []string{
		"/api/states/sensor.tibber_price_current",
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
	if n != 1 {
		t.Errorf("synced %d sensors, want 1 (price only)", n)
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
