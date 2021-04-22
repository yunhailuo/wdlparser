package wdlparser

type symboler interface {
	GetName() string
	GetRaw() string
	GetType() string
	GetValue() interface{}
	IsInitialized() bool
}

type symbol struct {
	initialized bool
	name        string
	raw         string
	typ         string
	value       string
}

func newSymbol(name, raw, typ, value string, initialized bool) *symbol {
	s := symbol{initialized, name, raw, typ, value}
	return &s
}

func (s *symbol) GetName() string {
	return s.name
}

func (s *symbol) SetName(name string) {
	s.name = name
}

func (s *symbol) GetRaw() string {
	return s.raw
}

func (s *symbol) SetRaw(raw string) {
	s.raw = raw
}

func (s *symbol) GetType() string {
	return s.typ
}

func (s *symbol) SetType(t string) {
	s.typ = t
}

func (s *symbol) GetValue() interface{} {
	// TODO: compute value in GO based on s.typ
	return s.GetRaw()
}

func (s *symbol) IsInitialized() bool {
	return s.initialized
}

type scopedSymbol struct {
	scope
	symbol
}
