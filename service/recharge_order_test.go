package service

import "testing"

func TestParseMoneyCents(t *testing.T) {
	cases := map[string]int{"10": 1000, "10.0": 1000, "10.00": 1000, "0.01": 1, "100.9": 10090}
	for input, want := range cases {
		got, err := parseMoneyCents(input)
		if err != nil || got != want { t.Fatalf("parseMoneyCents(%q) = %d, %v; want %d", input, got, err, want) }
	}
	for _, input := range []string{"", "-1", "1.001", "abc", "1.2.3"} {
		if _, err := parseMoneyCents(input); err == nil { t.Fatalf("parseMoneyCents(%q) should fail", input) }
	}
}
