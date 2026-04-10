package display

import "testing"

func TestRenderBar(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		max      float64
		width    int
		wantLen  int
		wantFull bool
	}{
		{"full bar", 10, 10, 10, 12, true},   // ▓ + 10 █ + ▏
		{"half bar", 5, 10, 10, 12, false},
		{"zero max", 5, 0, 10, 0, false},
		{"zero value", 0, 10, 10, 12, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBar(tt.value, tt.max, tt.width)
			if tt.max <= 0 {
				if got != "" {
					t.Errorf("renderBar(%f, %f, %d) = %q, want empty", tt.value, tt.max, tt.width, got)
				}
				return
			}
			if len([]rune(got)) == 0 {
				t.Errorf("expected non-empty bar")
			}
		})
	}
}
