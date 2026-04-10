package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/christofer-l/tibber-cli/display"
	"github.com/christofer-l/tibber-cli/hasync"
	"github.com/christofer-l/tibber-cli/homeassistant"
	"github.com/christofer-l/tibber-cli/tibber"
)

func main() {
	token := os.Getenv("TIBBER_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "TIBBER_TOKEN environment variable is required.")
		fmt.Fprintln(os.Stderr, "Get your token at: https://developer.tibber.com/settings/access-token")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	client := tibber.NewClient(token)
	cmd := os.Args[1]

	switch cmd {
	case "homes":
		homes, err := client.GetHomes()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		display.PrintHomes(os.Stdout, homes)

	case "consumption":
		fs := flag.NewFlagSet("consumption", flag.ExitOnError)
		hours := fs.Int("hours", 24, "number of hours to fetch")
		fs.Parse(os.Args[2:])

		homes, err := client.GetConsumption(*hours)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, h := range homes {
			if h.AppNickname != "" {
				fmt.Printf("\n── %s ──\n\n", h.AppNickname)
			}
			display.PrintConsumptionTable(os.Stdout, h.Consumption.Nodes)
		}

	case "prices":
		homes, err := client.GetPrices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, h := range homes {
			pi := h.CurrentSubscription.PriceInfo
			fmt.Printf("\nCurrent price: %.4f %s/kWh\n",
				pi.Current.Total, pi.Current.Currency)
			display.PrintPriceTable(os.Stdout, pi.Today, "Today's prices")
			display.PrintPriceTable(os.Stdout, pi.Tomorrow, "Tomorrow's prices")
		}

	case "sync":
		haURL := os.Getenv("HA_URL")
		if haURL == "" {
			haURL = "http://homeassistant:8123"
		}
		haToken := os.Getenv("HA_TOKEN")
		if haToken == "" {
			fmt.Fprintln(os.Stderr, "HA_TOKEN environment variable is required.")
			os.Exit(1)
		}
		haClient := homeassistant.NewClient(haURL, haToken)
		n, err := hasync.Run(client, haClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Synced %d sensors to Home Assistant\n", n)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: tibber-cli <command> [options]

Commands:
  homes                    List your Tibber homes
  consumption [--hours N]  Show hourly consumption (default: 24h)
  prices                   Show today's and tomorrow's energy prices
  sync                     Push consumption & prices to Home Assistant

Environment:
  TIBBER_TOKEN   Your Tibber personal access token
  HA_URL         Home Assistant URL (default: http://homeassistant:8123)
  HA_TOKEN       Home Assistant long-lived access token`)
}
