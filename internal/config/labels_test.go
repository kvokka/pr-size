package config

import "testing"

func TestLoadLabelSetMergesDefaults(t *testing.T) {
	content := `
S:
  name: custom/small
  color: 123456
  comment: keep it tiny
`

	set, err := LoadLabelSet(content)
	if err != nil {
		t.Fatalf("LoadLabelSet returned error: %v", err)
	}

	if got := set["S"].Name; got != "custom/small" {
		t.Fatalf("S name = %q, want custom/small", got)
	}
	if got := set["S"].Color; got != "123456" {
		t.Fatalf("S color = %q, want 123456", got)
	}
	if got := set["S"].Comment; got != "keep it tiny" {
		t.Fatalf("S comment = %q, want keep it tiny", got)
	}
	if got := set["S"].Lines; got != 10 {
		t.Fatalf("S lines = %d, want 10", got)
	}
	if set["S"].Symbols != nil {
		t.Fatal("S symbols should be nil when not explicitly configured")
	}
	if got := set["XL"].Name; got != "size/XL" {
		t.Fatalf("XL name = %q, want size/XL", got)
	}
}

func TestLoadLabelSetSupportsExplicitSymbols(t *testing.T) {
	content := `
L:
  symbols: 250
XS:
  symbols: 0
`

	set, err := LoadLabelSet(content)
	if err != nil {
		t.Fatalf("LoadLabelSet returned error: %v", err)
	}

	if set["L"].Symbols == nil || *set["L"].Symbols != 250 {
		t.Fatalf("L symbols = %v, want 250", set["L"].Symbols)
	}
	if set["XS"].Symbols == nil || *set["XS"].Symbols != 0 {
		t.Fatalf("XS symbols = %v, want 0", set["XS"].Symbols)
	}
}

func TestLoadLabelSetLeavesSymbolsUnsetWhenOnlyLinesAreOverridden(t *testing.T) {
	content := `
L:
  lines: 7
`

	set, err := LoadLabelSet(content)
	if err != nil {
		t.Fatalf("LoadLabelSet returned error: %v", err)
	}

	if got := set["L"].Lines; got != 7 {
		t.Fatalf("L lines = %d, want 7", got)
	}
	if set["L"].Symbols != nil {
		t.Fatal("L symbols should remain nil when only lines are overridden")
	}
	if got := set["L"].ResolvedSymbols(); got != 700 {
		t.Fatalf("L resolved symbols = %d, want 700", got)
	}
}
