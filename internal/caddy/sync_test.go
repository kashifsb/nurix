package caddy

import (
	"strings"
	"testing"
)

func TestGenerateBanner(t *testing.T) {
	banner := generateBanner("example.com")

	if !strings.Contains(banner, "EXAMPLE.COM") {
		t.Error("banner should contain uppercased domain name")
	}

	lines := strings.Split(strings.TrimSpace(banner), "\n")
	if len(lines) != 3 {
		t.Errorf("banner should have 3 lines, got %d", len(lines))
	}

	// Top and bottom borders should be equal length
	if len(lines[0]) != len(lines[2]) {
		t.Errorf("top border (%d) and bottom border (%d) should be same length",
			len(lines[0]), len(lines[2]))
	}

	// Middle line should be same length as borders
	if len(lines[1]) != len(lines[0]) {
		t.Errorf("middle line (%d) and border (%d) should be same length",
			len(lines[1]), len(lines[0]))
	}

	// All characters in borders should be '#'
	for _, ch := range lines[0] {
		if ch != '#' {
			t.Errorf("border should only contain '#', found '%c'", ch)
			break
		}
	}
}

func TestGenerateBanner_VariousDomains(t *testing.T) {
	testCases := []struct {
		domain   string
		expected string
	}{
		{"a.io", "A.IO"},
		{"example.com", "EXAMPLE.COM"},
		{"quantum-electronics.ltd", "QUANTUM-ELECTRONICS.LTD"},
		{"my.long.subdomain.example.org", "MY.LONG.SUBDOMAIN.EXAMPLE.ORG"},
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			banner := generateBanner(tc.domain)
			if !strings.Contains(banner, tc.expected) {
				t.Errorf("banner for '%s' should contain '%s'", tc.domain, tc.expected)
			}
		})
	}
}

func TestGenerateBanner_Symmetry(t *testing.T) {
	banner := generateBanner("acrux.ltd")
	lines := strings.Split(strings.TrimSpace(banner), "\n")

	middle := lines[1]

	// Should start with "##########" and end with "##########"
	if !strings.HasPrefix(middle, "##########") {
		t.Error("middle line should start with '##########'")
	}
	if !strings.HasSuffix(middle, "##########") {
		t.Error("middle line should end with '##########'")
	}
}
