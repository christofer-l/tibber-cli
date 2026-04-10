package hasync

import (
	"fmt"
	"sort"

	"github.com/christofer-l/tibber-cli/homeassistant"
	"github.com/christofer-l/tibber-cli/tibber"
)

func percentile(current float64, prices []tibber.Price) int {
	if len(prices) == 0 {
		return 0
	}
	count := 0
	for _, p := range prices {
		if p.Total <= current {
			count++
		}
	}
	return count * 100 / len(prices)
}

func priceStats(prices []tibber.Price) (min, max, avg float64) {
	if len(prices) == 0 {
		return
	}
	min = prices[0].Total
	max = prices[0].Total
	sum := 0.0
	for _, p := range prices {
		if p.Total < min {
			min = p.Total
		}
		if p.Total > max {
			max = p.Total
		}
		sum += p.Total
	}
	avg = sum / float64(len(prices))
	return
}

func priceForecast(prices []tibber.Price) []map[string]any {
	sorted := make([]tibber.Price, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartsAt.Before(sorted[j].StartsAt)
	})
	result := make([]map[string]any, len(sorted))
	for i, p := range sorted {
		result[i] = map[string]any{
			"starts_at": p.StartsAt.Format("2006-01-02T15:04:05-07:00"),
			"total":     p.Total,
		}
	}
	return result
}

func Run(tc *tibber.Client, hac *homeassistant.Client) (int, error) {
	homes, err := tc.GetPrices()
	if err != nil {
		return 0, fmt.Errorf("fetch prices: %w", err)
	}
	if len(homes) == 0 {
		return 0, fmt.Errorf("no homes found")
	}

	home := homes[0]
	price := home.CurrentSubscription.PriceInfo.Current
	currency := price.Currency

	count := 0

	// Push current price
	err = hac.SetState("sensor.tibber_price_current", homeassistant.SensorState{
		State: fmt.Sprintf("%.4f", price.Total),
		Attributes: map[string]any{
			"unit_of_measurement": currency + "/kWh",
			"friendly_name":      "Tibber Current Price",
			"energy":             price.Energy,
			"tax":                price.Tax,
			"starts_at":          price.StartsAt.Format("2006-01-02T15:04:05-07:00"),
		},
	})
	if err != nil {
		return count, fmt.Errorf("set price sensor: %w", err)
	}
	count++

	// Push price level (percentile rank for today)
	todayPrices := home.CurrentSubscription.PriceInfo.Today
	pct := percentile(price.Total, todayPrices)
	minP, maxP, avgP := priceStats(todayPrices)

	attrs := map[string]any{
		"unit_of_measurement": "%",
		"friendly_name":      "Tibber Price Level",
		"percentile":         pct,
		"min":                minP,
		"max":                maxP,
		"avg":                avgP,
		"current_price":      price.Total,
		"currency":           currency,
		"today":              priceForecast(todayPrices),
	}
	tomorrowPrices := home.CurrentSubscription.PriceInfo.Tomorrow
	if len(tomorrowPrices) > 0 {
		attrs["tomorrow"] = priceForecast(tomorrowPrices)
	}

	err = hac.SetState("sensor.tibber_price_level", homeassistant.SensorState{
		State:      fmt.Sprintf("%d", pct),
		Attributes: attrs,
	})
	if err != nil {
		return count, fmt.Errorf("set price level sensor: %w", err)
	}
	count++

	// Fetch consumption
	consHomes, err := tc.GetConsumption(48)
	if err != nil {
		return count, fmt.Errorf("fetch consumption: %w", err)
	}

	if len(consHomes) == 0 || len(consHomes[0].Consumption.Nodes) == 0 {
		fmt.Println("No consumption data available, skipping consumption sensors")
		return count, nil
	}

	// Find the most recent non-zero consumption node
	nodes := consHomes[0].Consumption.Nodes
	var latest *tibber.ConsumptionNode
	for i := len(nodes) - 1; i >= 0; i-- {
		if nodes[i].Consumption > 0 {
			latest = &nodes[i]
			break
		}
	}

	if latest == nil {
		fmt.Println("No non-zero consumption data found, skipping consumption sensors")
		return count, nil
	}

	timeFormat := "2006-01-02T15:04:05-07:00"

	// Push consumption
	err = hac.SetState("sensor.tibber_consumption_hourly", homeassistant.SensorState{
		State: fmt.Sprintf("%.3f", latest.Consumption),
		Attributes: map[string]any{
			"unit_of_measurement": "kWh",
			"friendly_name":      "Tibber Hourly Consumption",
			"device_class":       "energy",
			"state_class":        "measurement",
			"from":               latest.From.Format(timeFormat),
			"to":                 latest.To.Format(timeFormat),
		},
	})
	if err != nil {
		return count, fmt.Errorf("set consumption sensor: %w", err)
	}
	count++

	// Push cost
	err = hac.SetState("sensor.tibber_cost_hourly", homeassistant.SensorState{
		State: fmt.Sprintf("%.2f", latest.Cost),
		Attributes: map[string]any{
			"unit_of_measurement": currency,
			"friendly_name":      "Tibber Hourly Cost",
			"device_class":       "monetary",
			"state_class":        "measurement",
			"from":               latest.From.Format(timeFormat),
			"to":                 latest.To.Format(timeFormat),
			"unit_price":         latest.UnitPrice,
		},
	})
	if err != nil {
		return count, fmt.Errorf("set cost sensor: %w", err)
	}
	count++

	return count, nil
}
