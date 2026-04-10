package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/christofer-l/tibber-cli/tibber"
)

func PrintConsumptionTable(w io.Writer, nodes []tibber.ConsumptionNode) {
	if len(nodes) == 0 {
		fmt.Fprintln(w, "No consumption data available.")
		return
	}

	maxCons := 0.0
	for _, n := range nodes {
		if n.Consumption > maxCons {
			maxCons = n.Consumption
		}
	}

	header := fmt.Sprintf("%-18s %8s %10s %8s  %s", "Time", "kWh", "Price/kWh", "Cost", "")
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("─", 70))

	for _, n := range nodes {
		timeStr := n.From.Local().Format("2006-01-02 15:04")
		bar := renderBar(n.Consumption, maxCons, 20)
		fmt.Fprintf(w, "%-18s %8.2f %10.4f %8.2f  %s\n",
			timeStr,
			n.Consumption,
			n.UnitPrice,
			n.Cost,
			bar,
		)
	}
}

func PrintPriceTable(w io.Writer, prices []tibber.Price, label string) {
	if len(prices) == 0 {
		fmt.Fprintf(w, "No %s price data available.\n", label)
		return
	}

	maxPrice := 0.0
	for _, p := range prices {
		if p.Total > maxPrice {
			maxPrice = p.Total
		}
	}

	fmt.Fprintf(w, "\n%s\n", label)
	header := fmt.Sprintf("%-18s %8s %8s %8s  %s", "Time", "Total", "Energy", "Tax", "")
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("─", 65))

	for _, p := range prices {
		timeStr := p.StartsAt.Local().Format("2006-01-02 15:04")
		bar := renderBar(p.Total, maxPrice, 20)
		fmt.Fprintf(w, "%-18s %8.4f %8.4f %8.4f  %s\n",
			timeStr,
			p.Total,
			p.Energy,
			p.Tax,
			bar,
		)
	}
}

func PrintHomes(w io.Writer, homes []tibber.Home) {
	if len(homes) == 0 {
		fmt.Fprintln(w, "No homes found.")
		return
	}

	for _, h := range homes {
		fmt.Fprintf(w, "%-36s  %s — %s, %s %s\n",
			h.ID,
			h.AppNickname,
			h.Address.Address1,
			h.Address.PostalCode,
			h.Address.City,
		)
	}
}
