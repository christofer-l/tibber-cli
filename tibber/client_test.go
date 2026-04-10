package tibber

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server, token string) *Client {
	return &Client{
		token:      token,
		apiURL:     srv.URL,
		httpClient: srv.Client(),
	}
}

func TestGetHomesSendsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		resp := GraphQLResponse{
			Data: Data{Viewer: Viewer{Homes: []Home{{ID: "home-1", AppNickname: "My House"}}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(srv, "my-secret-token")
	homes, err := c.GetHomes()
	if err != nil {
		t.Fatalf("GetHomes failed: %v", err)
	}

	if gotAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-secret-token")
	}
	if len(homes) != 1 || homes[0].ID != "home-1" {
		t.Errorf("unexpected homes: %+v", homes)
	}
}

func TestGetHomesGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GraphQLResponse{
			Errors: []Error{{Message: "token expired"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(srv, "bad-token")
	_, err := c.GetHomes()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "GraphQL error: token expired" {
		t.Errorf("error = %q, want %q", got, "GraphQL error: token expired")
	}
}

func TestGetHomesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	c := newTestClient(srv, "bad-token")
	_, err := c.GetHomes()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetConsumption(t *testing.T) {
	var gotQuery GraphQLRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotQuery)
		resp := GraphQLResponse{
			Data: Data{Viewer: Viewer{Homes: []Home{{
				Consumption: Consumption{Nodes: []ConsumptionNode{
					{Consumption: 1.5, Cost: 0.75, UnitPrice: 0.50},
				}},
			}}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(srv, "tok")
	homes, err := c.GetConsumption(48)
	if err != nil {
		t.Fatalf("GetConsumption failed: %v", err)
	}

	if gotQuery.Variables["last"] != float64(48) {
		t.Errorf("last variable = %v, want 48", gotQuery.Variables["last"])
	}
	if len(homes) != 1 || len(homes[0].Consumption.Nodes) != 1 {
		t.Fatalf("unexpected response: %+v", homes)
	}
	if homes[0].Consumption.Nodes[0].Consumption != 1.5 {
		t.Errorf("consumption = %f, want 1.5", homes[0].Consumption.Nodes[0].Consumption)
	}
}
