package wdlparser

type Symbol struct {
	Name  string
	Type  string
	Value string
}

type ScopedSymbol struct {
	BaseScope
	Symbol
	Parent    Scope
	Children  []Scope
	SymbolMap map[string]Symbol
}
