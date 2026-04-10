package display

import "strings"

func renderBar(value, max float64, width int) string {
	if max <= 0 {
		return ""
	}
	ratio := value / max
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	return "▓" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "▏"
}
