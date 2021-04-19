package wdlparser

type Symbol struct {
	Name  string
	Type  string
	Value string
}

type ScopedSymbol struct {
	Scope
	Symbol
}
