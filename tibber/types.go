package tibber

import "time"

type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   Data    `json:"data"`
	Errors []Error `json:"errors,omitempty"`
}

type Error struct {
	Message string `json:"message"`
}

type Data struct {
	Viewer Viewer `json:"viewer"`
}

type Viewer struct {
	Homes []Home `json:"homes"`
}

type Home struct {
	ID                  string       `json:"id"`
	AppNickname         string       `json:"appNickname"`
	Address             Address      `json:"address"`
	CurrentSubscription Subscription `json:"currentSubscription"`
	Consumption         Consumption  `json:"consumption"`
}

type Address struct {
	Address1   string `json:"address1"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

type Subscription struct {
	PriceInfo PriceInfo `json:"priceInfo"`
}

type PriceInfo struct {
	Current  Price   `json:"current"`
	Today    []Price `json:"today"`
	Tomorrow []Price `json:"tomorrow"`
}

type Price struct {
	Total    float64   `json:"total"`
	Energy   float64   `json:"energy"`
	Tax      float64   `json:"tax"`
	StartsAt time.Time `json:"startsAt"`
	Currency string    `json:"currency"`
}

type Consumption struct {
	Nodes []ConsumptionNode `json:"nodes"`
}

type ConsumptionNode struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	Cost            float64   `json:"cost"`
	UnitPrice       float64   `json:"unitPrice"`
	UnitPriceVAT    float64   `json:"unitPriceVAT"`
	Consumption     float64   `json:"consumption"`
	ConsumptionUnit string    `json:"consumptionUnit"`
}
