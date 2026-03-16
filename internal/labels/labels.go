package labels

import "sort"

type Definition struct {
	Name    string `yaml:"name" json:"name"`
	Lines   int    `yaml:"lines" json:"lines"`
	Symbols *int   `yaml:"symbols,omitempty" json:"symbols,omitempty"`
	Color   string `yaml:"color" json:"color"`
	Comment string `yaml:"comment,omitempty" json:"comment,omitempty"`
}

type Set map[string]Definition

func DefaultSet() Set {
	return Set{
		"XS":  {Name: "size/XS", Lines: 0, Color: "2FBF6B"},
		"S":   {Name: "size/S", Lines: 10, Color: "55A84B"},
		"M":   {Name: "size/M", Lines: 30, Color: "7A9135"},
		"L":   {Name: "size/L", Lines: 100, Color: "9F6A27"},
		"XL":  {Name: "size/XL", Lines: 500, Color: "C44319"},
		"XXL": {Name: "size/XXL", Lines: 1000, Color: "E91C0B"},
	}
}

func (s Set) Clone() Set {
	cloned := make(Set, len(s))
	for key, def := range s {
		if def.Symbols != nil {
			symbols := *def.Symbols
			def.Symbols = &symbols
		}
		cloned[key] = def
	}
	return cloned
}

func (s Set) Ordered() []Definition {
	ordered := make([]Definition, 0, len(s))
	for _, def := range s {
		ordered = append(ordered, def)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Lines == ordered[j].Lines {
			if ordered[i].ResolvedSymbols() == ordered[j].ResolvedSymbols() {
				return ordered[i].Name < ordered[j].Name
			}
			return ordered[i].ResolvedSymbols() < ordered[j].ResolvedSymbols()
		}
		return ordered[i].Lines < ordered[j].Lines
	})
	return ordered
}

func (d Definition) ResolvedSymbols() int {
	if d.Symbols != nil {
		return *d.Symbols
	}
	return d.Lines * 100
}

func (s Set) Select(totalLines, totalSymbols int) Definition {
	ordered := s.Ordered()
	selected := ordered[0]
	for _, def := range ordered {
		if totalLines >= def.Lines || totalSymbols >= def.ResolvedSymbols() {
			selected = def
		}
	}
	return selected
}

func (s Set) Names() map[string]struct{} {
	names := make(map[string]struct{}, len(s))
	for _, def := range s {
		names[def.Name] = struct{}{}
	}
	return names
}
