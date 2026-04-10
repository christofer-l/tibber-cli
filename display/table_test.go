package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/christofer-l/tibber-cli/tibber"
)

func TestPrintConsumptionTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintConsumptionTable(&buf, nil)
	if !strings.Contains(buf.String(), "No consumption data") {
		t.Errorf("expected 'No consumption data', got %q", buf.String())
	}
}

func TestPrintConsumptionTable(t *testing.T) {
	var buf bytes.Buffer
	nodes := []tibber.ConsumptionNode{
		{
			From:        time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			To:          time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			Consumption: 2.5,
			UnitPrice:   0.45,
			Cost:        1.125,
		},
		{
			From:        time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			To:          time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Consumption: 1.0,
			UnitPrice:   0.50,
			Cost:        0.50,
		},
	}
	PrintConsumptionTable(&buf, nodes)
	out := buf.String()

	if !strings.Contains(out, "kWh") {
		t.Error("expected header with kWh")
	}
	if !strings.Contains(out, "2.50") {
		t.Error("expected consumption value 2.50")
	}
	if !strings.Contains(out, "0.4500") {
		t.Error("expected unit price 0.4500")
	}
}

func TestPrintPriceTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintPriceTable(&buf, nil, "Today")
	if !strings.Contains(buf.String(), "No Today price data") {
		t.Errorf("expected empty message, got %q", buf.String())
	}
}

func TestPrintHomes(t *testing.T) {
	var buf bytes.Buffer
	homes := []tibber.Home{
		{
			ID:          "abc-123",
			AppNickname: "Cabin",
			Address:     tibber.Address{Address1: "Main St 1", PostalCode: "12345", City: "Stockholm"},
		},
	}
	PrintHomes(&buf, homes)
	out := buf.String()
	if !strings.Contains(out, "abc-123") {
		t.Error("expected home ID")
	}
	if !strings.Contains(out, "Cabin") {
		t.Error("expected nickname")
	}
}
