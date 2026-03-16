package labels

import "testing"

func TestSizeThresholds(t *testing.T) {
	set := DefaultSet()
	tests := []struct {
		lines   int
		symbols int
		want    string
	}{
		{lines: 0, symbols: 0, want: "size/XS"},
		{lines: 9, symbols: 999, want: "size/XS"},
		{lines: 10, symbols: 0, want: "size/S"},
		{lines: 0, symbols: 1000, want: "size/S"},
		{lines: 29, symbols: 0, want: "size/S"},
		{lines: 30, symbols: 0, want: "size/M"},
		{lines: 0, symbols: 3000, want: "size/M"},
		{lines: 99, symbols: 0, want: "size/M"},
		{lines: 100, symbols: 0, want: "size/L"},
		{lines: 0, symbols: 10000, want: "size/L"},
		{lines: 499, symbols: 0, want: "size/L"},
		{lines: 500, symbols: 0, want: "size/XL"},
		{lines: 0, symbols: 50000, want: "size/XL"},
		{lines: 999, symbols: 0, want: "size/XL"},
		{lines: 1000, symbols: 0, want: "size/XXL"},
	}

	for _, tt := range tests {
		if got := set.Select(tt.lines, tt.symbols).Name; got != tt.want {
			t.Fatalf("Select(lines=%d, symbols=%d) = %q, want %q", tt.lines, tt.symbols, got, tt.want)
		}
	}
}

func TestSelectUsesExplicitSymbolThresholds(t *testing.T) {
	set := DefaultSet().Clone()
	symbols := 250
	def := set["L"]
	def.Symbols = &symbols
	set["L"] = def

	if got := set.Select(1, 249).Name; got != "size/XS" {
		t.Fatalf("Select below explicit symbol threshold = %q, want size/XS", got)
	}
	if got := set.Select(1, 250).Name; got != "size/L" {
		t.Fatalf("Select at explicit symbol threshold = %q, want size/L", got)
	}
}
