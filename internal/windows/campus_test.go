package windows

import "testing"

func TestMatchesSSID(t *testing.T) {
	tests := []struct {
		name    string
		current string
		allowed []string
		want    bool
	}{
		{name: "exact match", current: "Campus-WiFi", allowed: []string{"Campus-WiFi"}, want: true},
		{name: "case and surrounding spaces", current: " campus-wifi ", allowed: []string{"Campus-WiFi"}, want: true},
		{name: "different network", current: "Phone-Hotspot", allowed: []string{"Campus-WiFi"}},
		{name: "empty allow list", current: "Campus-WiFi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesSSID(tt.current, tt.allowed); got != tt.want {
				t.Fatalf("matchesSSID(%q, %v) = %t, want %t", tt.current, tt.allowed, got, tt.want)
			}
		})
	}
}
